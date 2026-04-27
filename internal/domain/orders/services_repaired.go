package orders

import (
	"eclaim-workshop-deck-api/internal/models"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

func (s *Service) GetRepairedOrders(workshopId uint) ([]models.Order, error) {
	return s.repo.GetRepairedOrders(workshopId)
}

func (s *Service) SetRepairedAsUnfinished(req CancelNegotiationRequest) (*models.Order, error) {
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

	orderPanels := workOrder.OrderPanels

	err = s.repo.WithTransaction(func(tx *gorm.DB) error {
		for _, oP := range orderPanels {
			repairHistory := &models.RepairHistory{
				OrderPanelNo: oP.OrderPanelNo,
				Status:       "incomplete",
				Note:         "Repairs set to incomplete",
				CreatedBy:    &req.LastModifiedBy,
			}

			err = s.repo.CreateRepairHistoryTx(tx, repairHistory)
			if err != nil {
				return fmt.Errorf("failed to create repair history for order panel %d: %w", oP.OrderPanelNo, err)
			}

			oP.CompletionStatus = "incomplete"
			oP.LastModifiedBy = &req.LastModifiedBy

			err = s.repo.UpdateOrderPanelTx(tx, &oP)
			if err != nil {
				return fmt.Errorf("failed to update order panel %d: %w", oP.OrderPanelNo, err)
			}

		}

		order.Status = "repairing"
		err = s.repo.UpdateOrderTx(tx, order)
		if err != nil {
			return fmt.Errorf("failed to update order %d: %w", order.OrderNo, err)
		}

		return nil
	})

	return order, nil
}

func (s *Service) RemindPickup(req RemindPickupRequest) ([]models.PickupReminder, error) {
	if req.LastModifiedBy == 0 {
		return nil, errors.New("last modified by is required")
	}

	if len(req.OrderNos) <= 0 {
		return nil, fmt.Errorf("order no is required: %d", len(req.OrderNos))
	}

	if req.NextRemindDate.IsZero() {
		return nil, errors.New("next remind pickup date is required")
	}

	var pickupReminders []models.PickupReminder
	err := s.repo.WithTransaction(func(tx *gorm.DB) error {
		for _, o := range req.OrderNos {
			remindDelivery := &models.PickupReminder{
				OrderNo:                 o,
				CreatedBy:               &req.LastModifiedBy,
				NextAvailableRemindDate: req.NextRemindDate,
			}

			err := s.repo.CreatePickupReminderTx(tx, remindDelivery)
			if err != nil {
				return fmt.Errorf("failed to create pickup reminder for order %d: %w", o, err)
			}

			pickupReminders = append(pickupReminders, *remindDelivery)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to remind pickup:", err)
	}

	return pickupReminders, nil
}

func (s *Service) SetAsDelivered() {

}
