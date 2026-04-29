package orders

import (
	"eclaim-workshop-deck-api/internal/domain/email"
	"eclaim-workshop-deck-api/internal/models"
	"eclaim-workshop-deck-api/pkg/utils"
	"errors"
	"fmt"
	"mime/multipart"
	"time"

	"gorm.io/gorm"
)

func createEmailService() *email.EmailService {
	return email.NewEmailService()
}

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

	err = utils.WithTransaction(s.repo, func(tx *gorm.DB) error {
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
		order.CompletedAt = nil
		order.LastModifiedBy = &req.LastModifiedBy
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

	firstOrderNo := req.OrderNos[0]
	var pickupReminders []models.PickupReminder
	err := utils.WithTransaction(s.repo, func(tx *gorm.DB) error {
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

	firstOrder, err := s.repo.FindOrderById(firstOrderNo)
	if err != nil {
		return nil, fmt.Errorf("failed to find order %d: %w", firstOrderNo, err)
	}

	client := firstOrder.Client

	emailService := createEmailService()
	if err := emailService.SendPickupReminder(
		client.ClientEmail,
		client.ClientName,
		client.VehicleBrandName,
		client.VehicleSeriesName,
		client.VehicleLicensePlate,
		client.VehicleChassisNo,
	); err != nil {
		fmt.Printf("Warning: failed to send pickup reminder email to %s: %v\n", client.ClientEmail, err)
	}

	return pickupReminders, nil
}

func (s *Service) SetAsDelivered(
	req SetAsDeliveredRequest,
	proofPhoto *multipart.FileHeader,
	uploadFn func(file multipart.File, header *multipart.FileHeader, folder string) (string, error),
) (*models.Delivery, error) {
	if req.LastModifiedBy == 0 {
		return nil, errors.New("last_modified_by is required")
	}
	if len(req.InvoiceNos) == 0 {
		return nil, errors.New("invoice_nos is required")
	}

	allowedTypes := map[string]bool{
		"image/jpeg": true,
		"image/jpg":  true,
		"image/png":  true,
		"image/webp": true,
	}
	maxSize := int64(10 << 20)

	if proofPhoto != nil {
		if proofPhoto.Size > maxSize {
			return nil, fmt.Errorf("proof photo exceeds 10MB limit")
		}
		contentType := proofPhoto.Header.Get("Content-Type")
		if !allowedTypes[contentType] {
			return nil, fmt.Errorf("invalid file type: %s", contentType)
		}
	}

	// --- Pre-transaction: gather data we need to build the Delivery record ---

	// Get client from the first invoice's orders
	firstInvoice, err := s.repo.FindInvoiceById(req.InvoiceNos[0])
	if err != nil || firstInvoice == nil {
		return nil, fmt.Errorf("failed to find invoice %d", req.InvoiceNos[0])
	}

	firstOrders, err := s.repo.FindOrdersFromInvoiceNo(firstInvoice.InvoiceNo)
	if err != nil || len(firstOrders) == 0 {
		return nil, fmt.Errorf("failed to find orders for invoice %d", firstInvoice.InvoiceNo)
	}
	clientNo := firstOrders[0].ClientNo

	// Find the latest completed_at across all orders of all invoices
	var lastRepairedDate time.Time
	for _, invoiceNo := range req.InvoiceNos {
		orders, err := s.repo.FindOrdersFromInvoiceNo(invoiceNo)
		if err != nil {
			return nil, fmt.Errorf("failed to find orders for invoice %d: %w", invoiceNo, err)
		}
		for _, o := range orders {
			if o.CompletedAt.After(lastRepairedDate) {
				lastRepairedDate = *o.CompletedAt
			}
		}
	}

	deliveryId, err := s.repo.GenerateDeliveryId()
	if err != nil {
		return nil, fmt.Errorf("failed to generate delivery id: %w", err)
	}

	// Upload proof photo before transaction (avoids holding tx open during I/O)
	var photoURL string
	if proofPhoto != nil {
		file, err := proofPhoto.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open proof photo: %w", err)
		}
		defer file.Close()

		folder := fmt.Sprintf("delivery/%d%02d%02d",
			time.Now().Year(),
			time.Now().Month(),
			time.Now().Day(),
		)

		photoURL, err = uploadFn(file, proofPhoto, folder)
		if err != nil {
			return nil, fmt.Errorf("failed to upload proof photo: %w", err)
		}
	}

	var delivery *models.Delivery

	err = utils.WithTransaction(s.repo, func(tx *gorm.DB) error {
		// 1. Create the single Delivery record
		newDelivery := &models.Delivery{
			ClientNo:         clientNo,
			DeliveryId:       deliveryId,
			LastRepairedDate: lastRepairedDate,
			DeliveredAt:      req.DeliveredAt,
			PhotoUrl:         photoURL,
			CreatedBy:        &req.LastModifiedBy,
		}

		if err := s.repo.CreateDeliveryTx(tx, newDelivery); err != nil {
			return fmt.Errorf("failed to create delivery: %w", err)
		}

		// 2. For each invoice: link it to the delivery, update its orders
		for _, invoiceNo := range req.InvoiceNos {
			invoice, err := s.repo.FindInvoiceByIdTx(tx, invoiceNo)
			if err != nil {
				return fmt.Errorf("failed to find invoice %d: %w", invoiceNo, err)
			}
			if invoice == nil {
				return fmt.Errorf("invoice %d not found", invoiceNo)
			}

			invoice.DeliveryNo = &newDelivery.DeliveryNo
			invoice.LastModifiedBy = &req.LastModifiedBy
			if err := s.repo.UpdateInvoiceTx(tx, invoice); err != nil {
				return fmt.Errorf("failed to link invoice %d to delivery: %w", invoiceNo, err)
			}

			orders, err := s.repo.FindOrdersFromInvoiceNoTx(tx, invoiceNo)
			if err != nil {
				return fmt.Errorf("failed to find orders for invoice %d: %w", invoiceNo, err)
			}

			for _, o := range orders {
				o.Status = "delivered"
				o.LastModifiedBy = &req.LastModifiedBy
				if err := s.repo.UpdateOrderTx(tx, &o); err != nil {
					return fmt.Errorf("failed to update order %d status: %w", o.OrderNo, err)
				}
			}
		}

		delivery = newDelivery
		return nil
	})

	if err != nil {
		return nil, err
	}

	return delivery, nil
}
