package orders

import (
	"eclaim-workshop-deck-api/internal/models"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

func (r *Repository) GetRepairedOrders(id uint) ([]models.Order, error) {
	var orders []models.Order

	err := r.db.
		Preload("Workshop").
		Preload("Insurance").
		Preload("Client").
		Preload("WorkOrders", "is_locked = 0").
		Preload("WorkOrders.OrderPanels", "is_locked = 0").
		Preload("WorkOrders.OrderPanels.InsurerPanelPricing.Measurements", "is_locked = ?", false).
		Preload("WorkOrders.OrderPanels.WorkshopPanelPricing.Measurements", "is_locked = ?", false).
		Preload("WorkOrders.OrderPanels.InsurerMeasurement").
		Preload("WorkOrders.OrderPanels.WorkshopMeasurement").
		Preload("WorkOrders.OrderPanels.FinalMeasurement").
		Preload("Invoice").
		Preload("Invoice.Client").
		Preload("Invoice.PaymentRecords").
		Preload("Invoice.InvoiceInstallments").
		Preload("Invoice.InvoiceInstallments.PaymentRecords").
		Preload("Invoice.Client.City").
		Preload("Client").
		Preload("Client.City").
		Preload("PickupReminders").
		Where("tr_orders.is_locked = ? AND tr_orders.workshop_no = ? AND tr_orders.status = ?", 0, id, "repaired").
		Order("tr_orders.order_no").
		Find(&orders).Error

	return orders, err
}

func (r *Repository) CreatePickupReminderTx(tx *gorm.DB, pickupReminder *models.PickupReminder) error {
	return tx.Create(pickupReminder).Error
}

func (r *Repository) CreateDeliveryTx(tx *gorm.DB, delivery *models.Delivery) error {
	return tx.Create(delivery).Error
}

func (r *Repository) FindInvoiceById(invoiceNo uint) (*models.Invoice, error) {
	var invoice models.Invoice
	err := r.db.Where("invoice_no = ? AND is_locked = ?", invoiceNo, false).First(&invoice).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &invoice, err
}

// GenerateDeliveryId generates the next DEL/YYYY/MM/XXXXXX reference number.
func (r *Repository) GenerateDeliveryId() (string, error) {
	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	// Count deliveries in the current month to get the next sequence number
	var count int64
	err := r.db.Model(&models.Delivery{}).
		Where("YEAR(created_date) = ? AND MONTH(created_date) = ?", year, month).
		Count(&count).Error
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("DEL/%d/%02d/%06d", year, month, count+1), nil
}
