package orders

import (
	"eclaim-workshop-deck-api/internal/models"

	"gorm.io/gorm"
)

func (r *Repository) GetRepairingOrders(id uint) ([]models.Order, error) {
	var orders []models.Order

	err := r.db.
		Preload("Workshop").
		Preload("Insurance").
		Preload("Client").
		Where("tr_orders.is_locked = ? AND tr_orders.workshop_no = ? AND tr_orders.status = ?", 0, id, "repairing").
		Order("tr_orders.order_no").
		Find(&orders).Error

	return orders, err
}

func (r *Repository) CreateRepairHistoryTx(tx *gorm.DB, history *models.RepairHistory) error {
	return tx.Create(history).Error
}

func (r *Repository) CreateRepairPhotosTx(tx *gorm.DB, photos []models.RepairPhoto) error {
	if len(photos) == 0 {
		return nil
	}
	return tx.Create(&photos).Error
}

func (r *Repository) FindSupplierFromID(id uint) (models.Supplier, error) {
	var supplier models.Supplier

	err := r.db.
		Preload("Workshop").
		Preload("Province").
		Preload("City").
		Where("r_suppliers.supplier_no = ? AND is_locked = 0", id).
		Find(&supplier).Error

	return supplier, err
}
