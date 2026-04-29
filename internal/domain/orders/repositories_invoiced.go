package orders

import "eclaim-workshop-deck-api/internal/models"

func (r *Repository) GetInvoicedOrders(id uint) ([]models.Order, error) {
	var orders []models.Order

	err := r.db.
		Preload("Workshop").
		Preload("Insurance").
		Preload("Client").
		Preload("Invoice").
		Preload("Invoice.InvoiceInstallments").
		Preload("Invoice.InvoiceInstallments.PaymentRecords").
		Preload("Invoice.PaymentRecords").
		Preload("Invoice.CreatedByUser").
		Preload("WorkOrders", "is_locked = 0").
		Preload("WorkOrders.OrderPanels", "is_locked = 0").
		Preload("WorkOrders.OrderPanels.InsurerPanelPricing.Measurements", "is_locked = ?", false).
		Preload("WorkOrders.OrderPanels.WorkshopPanelPricing.Measurements", "is_locked = ?", false).
		Preload("WorkOrders.OrderPanels.InsurerMeasurement").
		Preload("WorkOrders.OrderPanels.WorkshopMeasurement").
		Preload("WorkOrders.OrderPanels.FinalMeasurement").
		Preload("Client").
		Preload("Client.City").
		Where(`
			tr_orders.is_locked = ?
			AND tr_orders.workshop_no = ?
			AND tr_orders.invoice_no IS NOT NULL
			AND tr_orders.status IN ?
		`, 0, id, []string{"repaired", "delivered", "completed"}).
		Order("tr_orders.order_no").
		Find(&orders).Error

	return orders, err
}
