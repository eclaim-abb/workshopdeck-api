package orders

import "eclaim-workshop-deck-api/internal/models"

func (r *Repository) GetDeliveredOrders(id uint) ([]models.Order, error) {
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
		Preload("Invoice.Client.City").
		Preload("Invoice.PaymentRecords").
		Preload("Invoice.InvoiceInstallments").
		Preload("Invoice.InvoiceInstallments.PaymentRecords").
		Preload("Invoice.Delivery").
		Preload("Client").
		Preload("Client.City").
		Preload("PickupReminders").
		Where("tr_orders.is_locked = ? AND tr_orders.workshop_no = ? AND tr_orders.status = ?", 0, id, "delivered").
		Order("tr_orders.order_no").
		Find(&orders).Error

	return orders, err
}
