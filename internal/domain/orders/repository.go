package orders

import (
	"eclaim-workshop-deck-api/internal/models"
	"errors"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) WithTransaction(fn func(tx *gorm.DB) error) error {
	tx := r.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (r *Repository) GetOrders() ([]models.Order, error) {
	var orders []models.Order

	err := r.db.
		Preload("Workshop").
		Preload("Insurance").
		Preload("Client").
		Where("tr_orders.is_locked = ?", 0).
		Order("tr_orders.order_no").
		Find(&orders).Error

	return orders, err
}

func (r *Repository) GetOrderPanelWithLock(tx *gorm.DB, orderPanelNo uint) (*models.OrderPanel, error) {
	var orderPanel models.OrderPanel

	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&orderPanel, orderPanelNo).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("order panel not found")
		}
		return nil, err
	}

	return &orderPanel, nil
}

func (r *Repository) GetLatestNegotiationHistory(db *gorm.DB, orderPanelNo uint) (*models.NegotiationHistory, error) {
	var history models.NegotiationHistory

	err := db.Where("order_panel_no = ? AND is_locked = 0", orderPanelNo).
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

func (r *Repository) GetLatestRepairHistory(db *gorm.DB, orderPanelNo uint) (*models.RepairHistory, error) {
	var history models.RepairHistory

	err := db.Where("order_panel_no = ? AND is_locked = 0", orderPanelNo).
		First(&history).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // No negotiation history yet
		}
		return nil, err
	}

	return &history, nil
}

func (r *Repository) GetSpecificNegotiationHistoryRound(db *gorm.DB, orderPanelNo, roundNo uint) (*models.NegotiationHistory, error) {
	var history models.NegotiationHistory

	err := db.Where("order_panel_no = ? AND round_count = ? AND is_locked = 0", orderPanelNo, roundNo).
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

func (r *Repository) GetLatestAcceptedNegotiationHistory(db *gorm.DB, orderPanelNo uint) (*models.NegotiationHistory, error) {
	var history models.NegotiationHistory

	err := db.
		Preload("OldPanelPricing").
		Preload("ProposedPanelPricing").
		Preload("ProposedPanelPricing.WorkshopPanels").
		Where("order_panel_no = ? AND insurance_decision = ? AND is_locked = 0", orderPanelNo, "accepted").
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

func (r *Repository) ViewOrderDetails(id uint) (models.Order, error) {
	var order models.Order

	err := r.db.
		Preload("Workshop").
		Preload("Insurance").
		Preload("Invoice").
		Preload("Client").
		Preload("WorkOrders").
		Preload("WorkOrders.OrderPanels", "is_locked = 0").
		Preload("WorkOrders.OrderPanels.InsurerPanelPricing.Measurements").
		Preload("WorkOrders.OrderPanels.WorkshopPanelPricing.Measurements").
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
		Where("tr_orders.is_locked = ? AND tr_orders.order_no = ?", 0, id).
		Order("tr_orders.order_no").
		Find(&order).Error

	return order, err
}

func (r *Repository) FindClientById(id uint) (*models.Client, error) {
	var client models.Client

	err := r.db.
		Preload("City").
		Preload("CreatedByUser").
		Preload("LastModifiedByUser").
		Where("client_no = ? AND is_locked = 0", id).
		First(&client).Error

	return &client, err
}

func (r *Repository) FindOrderById(id uint) (*models.Order, error) {
	var order models.Order

	err := r.db.
		Preload("Workshop").
		Preload("Insurance").
		Preload("Invoice").
		Preload("Client").
		Preload("CreatedByUser").
		Preload("LastModifiedByUser").
		Where("order_no = ? AND is_locked = 0", id).
		First(&order).Error

	return &order, err
}

func (r *Repository) FindOrderPanelById(id uint) (*models.OrderPanel, error) {
	var orderPanel models.OrderPanel

	err := r.db.
		Preload("InsurerPanelPricing").
		Preload("WorkshopPanelPricing").
		Preload("FinalPanelPricing").
		Preload("InsurerMeasurement").
		Preload("WorkshopMeasurement").
		Preload("FinalMeasurement").
		Preload("CreatedByUser").
		Preload("LastModifiedByUser").
		Preload("RepairHistory").
		Preload("NegotiationHistory").
		Where("order_panel_no = ?", id).
		Find(&orderPanel).Error

	if err != nil {
		return nil, err
	}

	return &orderPanel, nil
}

func (r *Repository) FindWorkOrderById(id uint) (*models.WorkOrder, error) {
	var workOrder models.WorkOrder

	err := r.db.
		Preload("Order").
		Preload("OrderPanels").
		Preload("CreatedByUser").
		Preload("LastModifiedByUser").
		Where("work_order_no = ? AND is_locked = 0", id).
		First(&workOrder).Error

	return &workOrder, err
}

func (r *Repository) GetWorkOrder(db *gorm.DB, workOrderNo uint) (*models.WorkOrder, error) {
	var workOrder models.WorkOrder

	err := db.First(&workOrder, workOrderNo).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("work order not found")
		}
		return nil, err
	}

	return &workOrder, nil
}

func (r *Repository) FindWorkOrderFromOrderNo(id uint) (*models.WorkOrder, error) {
	var workOrder models.WorkOrder

	err := r.db.
		Preload("Order").
		Preload("OrderPanels").
		Preload("CreatedByUser").
		Preload("LastModifiedByUser").
		Where("order_no = ? AND is_locked = 0", id).
		First(&workOrder).Error

	return &workOrder, err
}

func (r *Repository) GetOrderPanelsGroupFromWorkOrderNo(id, woGroup uint) ([]models.OrderPanel, error) {
	var orderPanels []models.OrderPanel

	err := r.db.Where("work_order_no = ? AND is_locked = 0 AND work_order_group_number = ?", id, woGroup).Find(&orderPanels).Error

	return orderPanels, err
}

func (r *Repository) GetOrderPanelsGroupFromWorkOrderNoTx(tx *gorm.DB, id, woGroup uint) ([]models.OrderPanel, error) {
	var orderPanels []models.OrderPanel

	err := tx.Where("work_order_no = ? AND is_locked = 0 AND work_order_group_number = ?", id, woGroup).Find(&orderPanels).Error

	return orderPanels, err
}

func (r *Repository) GetOrderPanelsBeforeGroup(workOrderNo uint, beforeGroup uint) ([]models.OrderPanel, error) {
	var orderPanels []models.OrderPanel

	err := r.db.Where("work_order_no = ? AND is_locked = 0 AND work_order_group_number < ?",
		workOrderNo, beforeGroup).Find(&orderPanels).Error

	return orderPanels, err
}

// Create
func (r *Repository) CreateNegotiationHistory(tx *gorm.DB, history *models.NegotiationHistory) error {
	return tx.Create(history).Error
}

func (r *Repository) CreateNegotiationPhotos(tx *gorm.DB, photos []models.NegotiationPhotos) error {
	if len(photos) == 0 {
		return nil
	}
	return tx.Create(&photos).Error
}

func (r *Repository) CreateOrder(order *models.Order) error {
	return r.db.Create(order).Error
}

func (r *Repository) AddClient(client *models.Client) error {
	return r.db.Create(client).Error
}

func (r *Repository) CreateWorkOrder(workOrder *models.WorkOrder) error {
	return r.db.Create(workOrder).Error
}

func (r *Repository) CreateOrderPanel(orderPanel *models.OrderPanel) error {
	return r.db.Create(orderPanel).Error
}

func (r *Repository) CreateOrderPanelsBatch(orderPanels []*models.OrderPanel) error {
	if len(orderPanels) == 0 {
		return nil
	}
	return r.db.Create(&orderPanels).Error
}

func (r *Repository) CreateRepairHistory(tx *gorm.DB, history *models.RepairHistory) error {
	return tx.Create(history).Error
}

// Update
func (r *Repository) UpdateWorkOrder(workOrder *models.WorkOrder) error {
	return r.db.Save(workOrder).Error
}

func (r *Repository) UpdateOrderPanel(orderPanel *models.OrderPanel) error {
	return r.db.Save(orderPanel).Error
}

func (r *Repository) UpdateOrderPanelTx(tx *gorm.DB, orderPanel *models.OrderPanel) error {
	return tx.Save(orderPanel).Error
}

func (r *Repository) UpdateOrder(order *models.Order) error {
	return r.db.Save(order).Error
}

func (r *Repository) UpdateOrderTx(tx *gorm.DB, order *models.Order) error {
	return tx.Save(order).Error
}

func (r *Repository) UpdateOrderPanelNegotiation(tx *gorm.DB, orderPanelNo uint, updates map[string]interface{}) error {
	result := tx.Model(&models.OrderPanel{}).
		Where("order_panel_no = ?", orderPanelNo).
		Updates(updates)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("order panel not found or no changes made")
	}

	return nil
}

func (r *Repository) UpdateNegotiationHistoryTx(tx *gorm.DB, negotiationHistory *models.NegotiationHistory) error {
	return tx.Save(negotiationHistory).Error
}

func (r *Repository) BulkAcceptPanelsByGroupRangeTx(
	tx *gorm.DB,
	workOrderNo uint,
	startGroup, endGroup uint,
	lastModifiedBy uint,
) error {
	result := tx.
		Model(&models.OrderPanel{}).
		Where("work_order_no = ? AND work_order_group_number >= ? AND work_order_group_number < ? AND negotiation_status = ?",
			workOrderNo, startGroup, endGroup, "pending_workshop").
		Updates(map[string]interface{}{
			"negotiation_status": "accepted",
			"last_modified_by":   lastModifiedBy,
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("no panels found to accept for work order %d", workOrderNo)
	}

	return nil
}

func (r *Repository) CreateOrderPanelsBatchTx(tx *gorm.DB, orderPanels []*models.OrderPanel) error {
	if len(orderPanels) == 0 {
		return nil
	}
	return tx.Create(&orderPanels).Error
}

func (r *Repository) UpdateWorkOrderTx(tx *gorm.DB, workOrder *models.WorkOrder) error {
	return tx.Save(workOrder).Error
}
