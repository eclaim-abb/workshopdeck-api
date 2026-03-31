package orders

import (
	"eclaim-workshop-deck-api/internal/models"
	"errors"

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

func (r *Repository) CreateOrderAndRequestTx(tx *gorm.DB, orderAndRequest *models.OrderAndRequest) error {
	return tx.Create(orderAndRequest).Error
}

func (r *Repository) CreateSparePartQuoteTx(tx *gorm.DB, sparePartQuote *models.SparePartQuote) error {
	return tx.Create(sparePartQuote).Error
}

func (r *Repository) CreateSparePartNegotiationHistoryTx(tx *gorm.DB, sparePartNegotiationHistory *models.SparePartNegotiationHistory) error {
	return tx.Create(sparePartNegotiationHistory).Error
}

func (r *Repository) GetSparePartQuoteTx(tx *gorm.DB, orderRequestNo uint) (*models.SparePartQuote, error) {
	var quote models.SparePartQuote

	err := tx.Where("order_request_no = ?", orderRequestNo).First(&quote).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &quote, nil
}

func (r *Repository) GetLatestSparePartNegotiationHistory(db *gorm.DB, spare_part_quotes_no uint) (*models.SparePartNegotiationHistory, error) {
	var history models.SparePartNegotiationHistory

	err := db.Where("spare_part_quotes_no = ? AND is_locked = 0", spare_part_quotes_no).
		Order("round_count DESC").
		First(&history).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // No negotiation history yet
		}
		return nil, err
	}

	return &history, nil
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

func (r *Repository) UpdateSparePartQuoteTx(tx *gorm.DB, sparePartQuote *models.SparePartQuote) error {
	return tx.Save(sparePartQuote).Error
}
