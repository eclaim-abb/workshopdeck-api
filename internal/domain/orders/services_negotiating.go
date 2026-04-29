package orders

import (
	"eclaim-workshop-deck-api/internal/models"
	"eclaim-workshop-deck-api/pkg/utils"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

func (s *Service) GetNegotiatingOrders(workshopId uint) ([]models.Order, error) {
	return s.repo.GetNegotiatingOrders(workshopId)
}

func (s *Service) CancelNegotiation(req CancelNegotiationRequest) (*models.Order, error) {
	if req.LastModifiedBy == 0 {
		return nil, errors.New("last modified by is required")
	}

	if req.OrderNo == 0 {
		return nil, errors.New("order no is required")
	}

	order, err := s.repo.FindOrderById(req.OrderNo)
	if err != nil {
		return nil, err
	}

	workOrder, err := s.repo.FindWorkOrderFromOrderNo(req.OrderNo)
	if err != nil {
		return nil, err
	}

	addWONo := workOrder.AdditionalWorkOrderCount
	err = utils.WithTransaction(s.repo, func(tx *gorm.DB) error {
		switch order.Status {
		case "negotiating":
			// Workshop cancels their negotiation proposal
			for _, o := range workOrder.OrderPanels {
				orderPanel, err := s.repo.GetOrderPanelWithLock(tx, o.OrderPanelNo)
				if err != nil {
					return fmt.Errorf("failed to lock order panel %d: %w", o.OrderPanelNo, err)
				}

				// Only process panels that are actually in negotiating state
				if orderPanel.NegotiationStatus != "negotiating" {
					continue
				}

				// Get the current round's negotiation history (the one being cancelled)
				currentNegotiation, err := s.repo.GetSpecificNegotiationHistoryRound(tx, o.OrderPanelNo, orderPanel.CurrentRound)
				if err != nil {
					return fmt.Errorf("failed to get negotiation history for panel %d round %d: %w",
						o.OrderPanelNo, orderPanel.CurrentRound, err)
				}

				if currentNegotiation == nil {
					return fmt.Errorf("no negotiation found for panel %d at round %d",
						o.OrderPanelNo, orderPanel.CurrentRound)
				}

				curTime := time.Now()
				// Lock/cancel the current negotiation
				currentNegotiation.IsLocked = true
				currentNegotiation.LastModifiedBy = &req.LastModifiedBy
				currentNegotiation.InsuranceDecision = "declined"
				currentNegotiation.InsuranceNotes = "Cancelled by workshop"
				currentNegotiation.CompletedDate = &curTime

				err = s.repo.UpdateNegotiationHistoryTx(tx, currentNegotiation)
				if err != nil {
					return fmt.Errorf("failed to update negotiation history: %w", err)
				}

				// Find previous ACCEPTED negotiation (if any)
				var previousAcceptedNego *models.NegotiationHistory
				if orderPanel.CurrentRound > 1 {
					previousAcceptedNego, err = s.repo.GetLatestAcceptedNegotiationHistory(
						tx, o.OrderPanelNo,
					)
					if err != nil {
						return fmt.Errorf("failed to get previous accepted negotiation: %w", err)
					}
				}

				// Restore order panel state
				if previousAcceptedNego != nil {
					// Restore to previous accepted terms
					orderPanel.WorkshopPanelPricingNo = &previousAcceptedNego.ProposedPanelPricingNo
					orderPanel.WorkshopPrice = &previousAcceptedNego.ProposedPrice
					orderPanel.WorkshopMeasurementNo = previousAcceptedNego.ProposedMeasurementNo
					orderPanel.WorkshopServiceType = &previousAcceptedNego.ProposedServiceType
					orderPanel.WorkshopQty = &previousAcceptedNego.ProposedQty

					// Load panel name if needed
					if previousAcceptedNego.ProposedPanelPricing != nil {
						orderPanel.WorkshopPanelName = &previousAcceptedNego.ProposedPanelPricing.WorkshopPanels.PanelName
					}
				} else {
					// No previous accepted negotiation
					if orderPanel.InitialProposer == "insurer" {
						// Insurer-initiated: nullify workshop fields (revert to insurer terms)
						orderPanel.WorkshopPanelPricingNo = nil
						orderPanel.WorkshopPrice = nil
						orderPanel.WorkshopMeasurementNo = nil
						orderPanel.WorkshopServiceType = nil
						orderPanel.WorkshopQty = nil
						orderPanel.WorkshopPanelName = nil
					}
				}

				// Update order panel state
				orderPanel.NegotiationStatus = "pending_workshop"
				orderPanel.LastModifiedBy = &req.LastModifiedBy
				// CurrentRound stays the same!

				err = s.repo.UpdateOrderPanelTx(tx, orderPanel)
				if err != nil {
					return fmt.Errorf("failed to update order panel: %w", err)
				}
			}

		case "proposed_additional", "additional_work":
			panelsToReject, err := s.repo.GetOrderPanelsGroupFromWorkOrderNoTx(tx, workOrder.WorkOrderNo, addWONo)
			if err != nil {
				return fmt.Errorf("failed to get panels for group %d: %w", addWONo, err)
			}
			// Workshop cancels additional work proposal
			for _, o := range panelsToReject {
				_, err := s.rejectOrderPanelTx(tx, o.OrderPanelNo, req.LastModifiedBy, addWONo)

				if err != nil {
					fmt.Printf("ERROR: Failed to reject panel %d: %v\n", o.OrderPanelNo, err)
					return err
				}

			}

			workOrder.AdditionalWorkOrderCount -= 1
			err = s.repo.UpdateWorkOrderTx(tx, workOrder)
			if err != nil {
				return fmt.Errorf("failed to update work order: %w", err)
			}
		default:
			return errors.New("order is not eligible to have its negotiation/work order proposition cancelled")
		}

		// Update order status
		if order.IsStarted {
			order.Status = "repairing"
		} else {
			order.Status = "incoming"
		}

		err = s.repo.UpdateOrderTx(tx, order)
		if err != nil {
			return fmt.Errorf("failed to update order: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return order, nil
}

func (s *Service) ForwardAdditionalProposal(req ApproveAdditionalProposalRequest) (*models.WorkOrder, error) {
	if req.LastModifiedBy == 0 {
		return nil, errors.New("last_modified_by is required")
	}
	if req.OrderNo == 0 {
		return nil, errors.New("order_no is required")
	}

	if req.ETA.IsZero() {
		return nil, errors.New("eta is required")
	}

	order, err := s.repo.FindOrderById(req.OrderNo)
	if err != nil {
		return nil, err
	}

	// Validate order is in correct status
	if order.Status != "proposed_additional" {
		return nil, fmt.Errorf("order is not in proposed_additional status (current: %s)", order.Status)
	}

	workOrder, err := s.repo.FindWorkOrderFromOrderNo(uint(req.OrderNo))
	if err != nil {
		return nil, err
	}

	currentGroup := workOrder.AdditionalWorkOrderCount

	// Get old panels (groups before current)
	var oldPanels []models.OrderPanel
	if currentGroup > 0 {
		oldPanels, err = s.repo.GetOrderPanelsBeforeGroup(workOrder.WorkOrderNo, currentGroup)

		if err != nil {
			return nil, err
		}
	}

	// Get the new additional panels (current group)
	additionalPanels, err := s.repo.GetOrderPanelsGroupFromWorkOrderNo(workOrder.WorkOrderNo, currentGroup)
	if err != nil {
		return nil, err
	}

	if len(additionalPanels) == 0 {
		return nil, errors.New("no additional panels found in current group")
	}

	err = utils.WithTransaction(s.repo, func(tx *gorm.DB) error {
		for _, op := range oldPanels {
			_, err := s.acceptOrderPanelTx(tx, op.OrderPanelNo, req.LastModifiedBy)

			if err != nil {
				return err
			}
		}

		for _, ap := range additionalPanels {
			_, err := s.forwardOrderPanelProposalTx(tx, ap.OrderPanelNo, req.LastModifiedBy)

			if err != nil {
				return err
			}
		}

		if req.DiscountType != "" && req.Discount != 0 {
			order.DiscountType = req.DiscountType
			order.Discount = req.Discount
		}

		order.Eta = &req.ETA
		order.IsStarted = true
		order.Status = "additional_work"
		order.LastModifiedBy = &req.LastModifiedBy

		if err := s.repo.UpdateOrderTx(tx, order); err != nil {
			return fmt.Errorf("failed to update order status: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return workOrder, nil
}
