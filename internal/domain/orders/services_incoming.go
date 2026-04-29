package orders

import (
	"eclaim-workshop-deck-api/internal/models"
	"eclaim-workshop-deck-api/pkg/utils"
	"errors"
	"fmt"
	"mime/multipart"
	"time"

	"gorm.io/gorm"
)

func (s *Service) GetIncomingOrders(workshopId uint) ([]models.Order, error) {
	return s.repo.GetIncomingOrders(workshopId)
}

func (s *Service) SubmitNegotiation(
	req *SubmitNegotiationRequest,
	files []*multipart.FileHeader,
	uploadFn func(file multipart.File, header *multipart.FileHeader, folder string) (string, error),
) (*models.WorkOrder, error) {
	if req.WorkOrderNo == 0 {
		return nil, errors.New("work_order_no is required")
	}
	if len(req.OrderPanels) == 0 {
		return nil, errors.New("order_panels cannot be empty")
	}
	if req.Reason == "" {
		return nil, errors.New("reason is required")
	}
	if len(req.Photos) != len(files) {
		return nil, fmt.Errorf("file count mismatch: expected %d, got %d", len(req.Photos), len(files))
	}

	// Validate files
	allowedTypes := map[string]bool{
		"image/jpeg": true,
		"image/jpg":  true,
		"image/png":  true,
		"image/webp": true,
	}
	maxSize := int64(10 << 20)

	for _, fh := range files {
		if fh.Size > maxSize {
			return nil, fmt.Errorf("file %s exceeds 10MB", fh.Filename)
		}
		contentType := fh.Header.Get("Content-Type")
		if !allowedTypes[contentType] {
			return nil, fmt.Errorf("invalid file type %s for %s", contentType, fh.Filename)
		}
	}

	workOrder, err := s.repo.FindWorkOrderById(req.WorkOrderNo)
	if err != nil {
		return nil, err
	}
	if workOrder == nil {
		return nil, errors.New("work order not found")
	}

	order, err := s.repo.FindOrderById(workOrder.OrderNo)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, errors.New("order not found")
	}

	// Build photo mapping
	photosByPanel := make(map[uint][]*multipart.FileHeader)
	for _, photoMeta := range req.Photos {
		if photoMeta.FileIndex >= 0 && photoMeta.FileIndex < len(files) {
			photosByPanel[photoMeta.OrderPanelNo] = append(
				photosByPanel[photoMeta.OrderPanelNo],
				files[photoMeta.FileIndex],
			)
		}
	}

	negotiationsCreated := make([]NegotiationCreatedInfo, 0)

	err = utils.WithTransaction(s.repo, func(tx *gorm.DB) error {
		for _, panelReq := range req.OrderPanels {
			orderPanel, err := s.repo.GetOrderPanelWithLock(tx, panelReq.OrderPanelNo)
			if err != nil {
				return fmt.Errorf("failed to lock order panel %d: %w", panelReq.OrderPanelNo, err)
			}

			// Validate status
			if orderPanel.NegotiationStatus != "pending_workshop" {
				return fmt.Errorf("order panel %d is not pending workshop action (current status: %s)",
					panelReq.OrderPanelNo, orderPanel.NegotiationStatus)
			}

			newRound := orderPanel.CurrentRound + 1

			negotiationHistory := &models.NegotiationHistory{
				OrderPanelNo:           panelReq.OrderPanelNo,
				RoundCount:             newRound,
				ProposedPanelPricingNo: panelReq.WorkshopPanelPricingNo,
				ProposedPrice:          panelReq.WorkshopPrice,
				ProposedServiceType:    panelReq.WorkshopServiceType,
				ProposedQty:            panelReq.WorkshopQty,
				WorkshopNotes:          req.Reason,
				InsuranceDecision:      "pending",
				CreatedBy:              &req.LastModifiedBy,
			}

			if orderPanel.InitialProposer == "insurer" {
				if orderPanel.InsurancePanelPricingNo != nil {
					negotiationHistory.OldPanelPricingNo = orderPanel.InsurancePanelPricingNo
				}
				if orderPanel.InsurerPrice != nil {
					negotiationHistory.OldPrice = *orderPanel.InsurerPrice
				}
				if orderPanel.InsurerMeasurementNo != nil {
					negotiationHistory.OldMeasurementNo = orderPanel.InsurerMeasurementNo
				}
				if orderPanel.InsurerServiceType != nil {
					negotiationHistory.OldServiceType = *orderPanel.InsurerServiceType
				}
				if orderPanel.InsurerQty != nil {
					negotiationHistory.OldQty = *orderPanel.InsurerQty
				}
			} else {
				if orderPanel.WorkshopPanelPricingNo != nil {
					negotiationHistory.OldPanelPricingNo = orderPanel.WorkshopPanelPricingNo
				}
				if orderPanel.WorkshopPrice != nil {
					negotiationHistory.OldPrice = *orderPanel.WorkshopPrice
				}
				if orderPanel.WorkshopMeasurementNo != nil {
					negotiationHistory.OldMeasurementNo = orderPanel.WorkshopMeasurementNo
				}
				if orderPanel.WorkshopServiceType != nil {
					negotiationHistory.OldServiceType = *orderPanel.WorkshopServiceType
				}
				if orderPanel.WorkshopQty != nil {
					negotiationHistory.OldQty = *orderPanel.WorkshopQty
				}
			}

			if panelReq.WorkshopMeasurementNo != 0 {
				negotiationHistory.ProposedMeasurementNo = &panelReq.WorkshopMeasurementNo
			}

			err = s.repo.CreateNegotiationHistory(tx, negotiationHistory)
			if err != nil {
				return fmt.Errorf("failed to create negotiation history for panel %d: %w",
					panelReq.OrderPanelNo, err)
			}

			// Update order panel
			orderPanel.CurrentRound = newRound
			orderPanel.NegotiationStatus = "negotiating"
			orderPanel.WorkshopPanelPricingNo = &panelReq.WorkshopPanelPricingNo
			orderPanel.WorkshopPanelName = &panelReq.WorkshopPanelName
			orderPanel.WorkshopPrice = &panelReq.WorkshopPrice
			orderPanel.WorkshopServiceType = &panelReq.WorkshopServiceType
			orderPanel.WorkshopQty = &panelReq.WorkshopQty
			orderPanel.IsIncluded = panelReq.IsIncluded
			orderPanel.IsSpecialRepair = panelReq.IsSpecialRepair
			orderPanel.LastModifiedBy = &req.LastModifiedBy

			if panelReq.WorkshopMeasurementNo != 0 {
				orderPanel.WorkshopMeasurementNo = &panelReq.WorkshopMeasurementNo
			}

			err = s.repo.UpdateOrderPanelTx(tx, orderPanel)
			if err != nil {
				return fmt.Errorf("failed to update order panel %d: %w", panelReq.OrderPanelNo, err)
			}

			// Upload photos
			photosUploaded := 0
			panelPhotos := photosByPanel[panelReq.OrderPanelNo]

			if len(panelPhotos) > 0 {
				photoRecords := make([]models.NegotiationPhotos, 0)

				for _, fileHeader := range panelPhotos {
					file, err := fileHeader.Open()
					if err != nil {
						return fmt.Errorf("failed to open file %s: %w", fileHeader.Filename, err)
					}

					folder := fmt.Sprintf("nego/%d/cost/%d%02d%02d",
						order.OrderNo,
						time.Now().Year(),
						time.Now().Month(),
						time.Now().Day(),
					)

					photoURL, err := uploadFn(file, fileHeader, folder)
					file.Close()

					if err != nil {
						return fmt.Errorf("failed to upload file %s: %w", fileHeader.Filename, err)
					}

					photoRecords = append(photoRecords, models.NegotiationPhotos{
						NegotiationHistoryNo: negotiationHistory.NegotiationHistoryNo,
						PhotoUrl:             photoURL,
						CreatedBy:            &req.LastModifiedBy,
					})

					photosUploaded++
				}

				if len(photoRecords) > 0 {
					err = s.repo.CreateNegotiationPhotos(tx, photoRecords)
					if err != nil {
						return fmt.Errorf("failed to save photo records for panel %d: %w",
							panelReq.OrderPanelNo, err)
					}
				}
			}

			negotiationsCreated = append(negotiationsCreated, NegotiationCreatedInfo{
				OrderPanelNo:         panelReq.OrderPanelNo,
				NegotiationHistoryNo: negotiationHistory.NegotiationHistoryNo,
				RoundCount:           newRound,
				PhotosUploaded:       photosUploaded,
			})
		}

		// Update order status
		order.Status = "negotiating"
		order.LastModifiedBy = &req.LastModifiedBy
		err = s.repo.UpdateOrderTx(tx, order)
		if err != nil {
			return fmt.Errorf("failed to update order status: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return workOrder, nil
}

func (s *Service) AcceptOrder(id uint, req AcceptDeclineOrder) (*models.Order, error) {
	order, err := s.repo.ViewOrderDetails(id)
	if err != nil {
		return nil, errors.New("order not found")
	}

	if order.Status != "incoming" {
		return nil, fmt.Errorf("order is not in incoming status (current: %s)", order.Status)
	}

	workOrder := order.WorkOrders[0]
	groupNo := workOrder.AdditionalWorkOrderCount

	err = utils.WithTransaction(s.repo, func(tx *gorm.DB) error {
		// Get panels in current group
		var orderPanels []models.OrderPanel
		for _, op := range workOrder.OrderPanels {
			if op.WorkOrderGroupNumber <= groupNo {
				orderPanels = append(orderPanels, op)
			}
		}

		for _, op := range orderPanels {
			lockedPanel, err := s.repo.GetOrderPanelWithLock(tx, op.OrderPanelNo)
			if err != nil {
				return fmt.Errorf("failed to lock panel %d: %w", op.OrderPanelNo, err)
			}

			// Workshop accepts insurer's original terms
			// Copy insurer → final (no negotiation happened)
			if lockedPanel.InitialProposer == "insurer" {
				lockedPanel.FinalPanelPricingNo = lockedPanel.InsurancePanelPricingNo
				lockedPanel.FinalPanelName = lockedPanel.InsurancePanelName
				lockedPanel.FinalPrice = lockedPanel.InsurerPrice
				lockedPanel.FinalServiceType = lockedPanel.InsurerServiceType
				lockedPanel.FinalMeasurementNo = lockedPanel.InsurerMeasurementNo
				lockedPanel.FinalQty = lockedPanel.InsurerQty
			}

			lockedPanel.NegotiationStatus = "accepted"
			lockedPanel.LastModifiedBy = &req.LastModifiedBy

			err = s.repo.UpdateOrderPanelTx(tx, lockedPanel)
			if err != nil {
				return fmt.Errorf("failed to update panel %d: %w", op.OrderPanelNo, err)
			}
		}

		// Update order
		order.IsStarted = true
		order.Status = "repairing"
		order.LastModifiedBy = &req.LastModifiedBy

		if !req.ETA.IsZero() {
			order.Eta = &req.ETA
		}

		if req.DiscountType != "" {
			order.DiscountType = req.DiscountType
			order.Discount = req.Discount
		}

		err = s.repo.UpdateOrderTx(tx, &order)
		if err != nil {
			return fmt.Errorf("failed to update order: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &order, nil
}

func (s *Service) DeclineOrder(id uint, req AcceptDeclineOrder) (*models.Order, error) {
	order, err := s.repo.ViewOrderDetails(id)
	if err != nil {
		return nil, errors.New("order not found")
	}

	if order.Status != "incoming" {
		return nil, fmt.Errorf("order is not in incoming status (current: %s)", order.Status)
	}

	workOrder := order.WorkOrders[0]
	groupNo := workOrder.AdditionalWorkOrderCount

	err = utils.WithTransaction(s.repo, func(tx *gorm.DB) error {
		// Get panels in current group
		var orderPanels []models.OrderPanel
		for _, op := range workOrder.OrderPanels {
			if op.WorkOrderGroupNumber <= groupNo {
				orderPanels = append(orderPanels, op)
			}
		}

		for _, op := range orderPanels {
			lockedPanel, err := s.repo.GetOrderPanelWithLock(tx, op.OrderPanelNo)
			if err != nil {
				return fmt.Errorf("failed to lock panel %d: %w", op.OrderPanelNo, err)
			}

			lockedPanel.IsLocked = true
			lockedPanel.NegotiationStatus = "rejected"
			lockedPanel.LastModifiedBy = &req.LastModifiedBy

			err = s.repo.UpdateOrderPanelTx(tx, lockedPanel)
			if err != nil {
				return fmt.Errorf("failed to update panel %d: %w", op.OrderPanelNo, err)
			}
		}

		// Lock work order
		workOrder.IsLocked = true
		workOrder.LastModifiedBy = &req.LastModifiedBy
		err = s.repo.UpdateWorkOrderTx(tx, &workOrder)
		if err != nil {
			return fmt.Errorf("failed to update work order: %w", err)
		}

		// Lock order
		order.Status = "declined"
		order.LastModifiedBy = &req.LastModifiedBy
		order.IsLocked = true
		err = s.repo.UpdateOrderTx(tx, &order)
		if err != nil {
			return fmt.Errorf("failed to update order: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &order, nil
}
