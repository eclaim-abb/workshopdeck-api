package orders

import (
	"eclaim-workshop-deck-api/internal/models"

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
		Preload("WorkOrders.OrderPanels.RepairHistory").
		Preload("WorkOrders.OrderPanels.RepairHistory.CreatedByUser").
		Preload("WorkOrders.OrderPanels.NegotiationHistory").
		Preload("WorkOrders.OrderPanels.NegotiationHistory.CreatedByUser").
		Preload("WorkOrders.OrderPanels.NegotiationHistory.OldMeasurement").
		Preload("WorkOrders.OrderPanels.NegotiationHistory.ProposedMeasurement").
		Preload("WorkOrders.OrderPanels.RepairHistory.RepairPhotos").
		Preload("WorkOrders.OrderPanels.RepairHistory.OrdersAndRequests").
		Preload("WorkOrders.OrderPanels.RepairHistory.OrdersAndRequests.SparePartQuotes").
		Preload("WorkOrders.OrderPanels.RepairHistory.OrdersAndRequests.SparePartQuotes.SparePartNegotiationHistory").
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
