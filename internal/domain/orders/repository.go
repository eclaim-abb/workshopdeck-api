package orders

import (
	"eclaim-workshop-deck-api/internal/models"
	"errors"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repository provides methods to interact with the database for
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new instance of Repository with the given database connection.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// WithTransaction executes the given function within a rollback-able database transaction
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

// GetOrders retrieves all orders and their details.
func (r *Repository) GetOrders() ([]models.Order, error) {
	var orders []models.Order

	err := r.db.
		Preload("Workshop").
		Preload("Insurance").
		Preload("Client").
		Preload("PickupReminders").
		Where("tr_orders.is_locked = ?", 0).
		Order("tr_orders.order_no").
		Find(&orders).Error

	return orders, err
}

// GetOrderPanelWithLock retrieves an order panel by its ID.
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

// FindOrderPanelsByWorkOrderNo retrieves all order panels associated with a given work order number.
func (r *Repository) FindOrderPanelsByWorkOrderNo(workOrderNo uint) ([]models.OrderPanel, error) {
	var orderPanels []models.OrderPanel

	err := r.db.Where("work_order_no = ? AND is_locked = 0", workOrderNo).Find(&orderPanels).Error

	if err != nil {
		return nil, err
	}

	return orderPanels, nil
}

// GetLatestNegotiationHistory retrieves the latest negotiation history for a given order panel number.
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

// GetLatestRepairHistory retrieves the latest repair history for a given order panel number.
func (r *Repository) GetLatestRepairHistory(db *gorm.DB, orderPanelNo uint) (*models.RepairHistory, error) {
	var history models.RepairHistory

	err := db.Where("order_panel_no = ? AND is_locked = 0", orderPanelNo).
		Order("created_date DESC").
		First(&history).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // No repair history yet
		}
		return nil, err
	}

	return &history, nil
}

// GetSpecificNegotiationHistoryRound retrieves a specific round of negotiation history for a given order panel number and round number.
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

// GetLatestAcceptedNegotiationHistory retrieves the latest accepted negotiation history for a given order panel number.
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

// ViewOrderDetails retrieves detailed information about an order, including related entities such as workshop, insurance, client, work orders, and order panels.
func (r *Repository) ViewOrderDetails(id uint) (models.Order, error) {
	var order models.Order

	err := r.db.
		Preload("Workshop").
		Preload("Insurance").
		Preload("Invoice").
		Preload("Client").
		Preload("WorkOrders").
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
		Preload("PickupReminders").
		Where("tr_orders.is_locked = ? AND tr_orders.order_no = ?", 0, id).
		Order("tr_orders.order_no").
		Find(&order).Error

	return order, err
}

// FindClientById retrieves a client by its ID.
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

// FindOrderById retrieves an order by its ID.
func (r *Repository) FindOrderById(id uint) (*models.Order, error) {
	var order models.Order

	err := r.db.
		Preload("Workshop").
		Preload("Insurance").
		Preload("Invoice").
		Preload("Client").
		Preload("PickupReminders").
		Preload("CreatedByUser").
		Preload("LastModifiedByUser").
		Where("order_no = ? AND is_locked = 0", id).
		First(&order).Error

	return &order, err
}

// FindOrderPanelById retrieves an order panel by its ID, including related entities such as pricing, measurements, users, and histories.
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

// FindWorkOrderById retrieves a work order by its ID, including related entities such as order, order panels, and users.
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

// GetWorkOrder retrieves a work order by its ID without the is_locked condition, used for internal operations where locking is handled separately.
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

// FindWorkOrderFromOrderNo retrieves a work order associated with a given order number, including related entities such as order, order panels, and users.
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

// GetOrderPanelsGroupFromWorkOrderNo retrieves order panels associated with a given work order number and group number, filtering out locked panels.
func (r *Repository) GetOrderPanelsGroupFromWorkOrderNo(id, woGroup uint) ([]models.OrderPanel, error) {
	var orderPanels []models.OrderPanel

	err := r.db.Where("work_order_no = ? AND is_locked = 0 AND work_order_group_number = ?", id, woGroup).Find(&orderPanels).Error

	return orderPanels, err
}

// GetOrderPanelsGroupFromWorkOrderNoTx retrieves order panels associated with a given work order number and group number within a transaction, filtering out locked panels.
func (r *Repository) GetOrderPanelsGroupFromWorkOrderNoTx(tx *gorm.DB, id, woGroup uint) ([]models.OrderPanel, error) {
	var orderPanels []models.OrderPanel

	err := tx.Where("work_order_no = ? AND is_locked = 0 AND work_order_group_number = ?", id, woGroup).Find(&orderPanels).Error

	return orderPanels, err
}

// GetOrderPanelsBeforeGroup retrieves order panels associated with a given work order number that belong to groups before a specified group number, filtering out locked panels.
func (r *Repository) GetOrderPanelsBeforeGroup(workOrderNo uint, beforeGroup uint) ([]models.OrderPanel, error) {
	var orderPanels []models.OrderPanel

	err := r.db.Where("work_order_no = ? AND is_locked = 0 AND work_order_group_number < ?",
		workOrderNo, beforeGroup).Find(&orderPanels).Error

	return orderPanels, err
}

// GetOrderAndRequestsByRepairHistoryNo retrieves all orders and requests associated with a given repair history number.
func (r *Repository) GetOrderAndRequestsByRepairHistoryNo(repairHistoryNo uint) ([]models.OrderAndRequest, error) {
	var ordersAndRequests []models.OrderAndRequest

	err := r.db.
		Preload("SparePartQuotes").
		Preload("SparePartQuotes.SparePartNegotiationHistory").
		Where("repair_history_no = ?", repairHistoryNo).
		Find(&ordersAndRequests).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // No negotiation history yet
		}

		return nil, err
	}

	return ordersAndRequests, err
}

/** Create */

// CreateNegotiationHistory creates a new negotiation history.
func (r *Repository) CreateNegotiationHistory(tx *gorm.DB, history *models.NegotiationHistory) error {
	return tx.Create(history).Error
}

// CreateNegotiationPhotos creates new negotiation photos.
func (r *Repository) CreateNegotiationPhotos(tx *gorm.DB, photos []models.NegotiationPhotos) error {
	if len(photos) == 0 {
		return nil
	}
	return tx.Create(&photos).Error
}

// CreateOrder creates a new order in the database.
func (r *Repository) CreateOrder(order *models.Order) error {
	return r.db.Create(order).Error
}

// AddClient adds a new client to the database.
func (r *Repository) AddClient(client *models.Client) error {
	return r.db.Create(client).Error
}

// CreateWorkOrder creates a new work order in the database.
func (r *Repository) CreateWorkOrder(workOrder *models.WorkOrder) error {
	return r.db.Create(workOrder).Error
}

// CreateOrderPanel creates a new order panel in the database.
func (r *Repository) CreateOrderPanel(orderPanel *models.OrderPanel) error {
	return r.db.Create(orderPanel).Error
}

// CreateOrderPanelsBatch creates multiple order panels in a single batch operation, improving performance when adding multiple panels at once.
func (r *Repository) CreateOrderPanelsBatch(orderPanels []*models.OrderPanel) error {
	if len(orderPanels) == 0 {
		return nil
	}
	return r.db.Create(&orderPanels).Error
}

// CreateRepairHistory creates a new repair history record in the database, associating it with an order panel and capturing details about the repair process.
func (r *Repository) CreateRepairHistory(tx *gorm.DB, history *models.RepairHistory) error {
	return tx.Create(history).Error
}

// Update

// UpdateWorkOrder updates an existing work order.
func (r *Repository) UpdateWorkOrder(workOrder *models.WorkOrder) error {
	return r.db.Save(workOrder).Error
}

// UpdateOrderPanel updates an existing order panel.
func (r *Repository) UpdateOrderPanel(orderPanel *models.OrderPanel) error {
	return r.db.Save(orderPanel).Error
}

// UpdateOrderPanelTx updates an existing order panel within a transaction, allowing for atomic operations when multiple related updates are needed.
func (r *Repository) UpdateOrderPanelTx(tx *gorm.DB, orderPanel *models.OrderPanel) error {
	return tx.Save(orderPanel).Error
}

// UpdateOrder updates an existing order in the database, allowing for changes to order details such as workshop, insurance, client information, and associated work orders.
func (r *Repository) UpdateOrder(order *models.Order) error {
	return r.db.Save(order).Error
}

// UpdateOrderTx updates an existing order within a transaction
func (r *Repository) UpdateOrderTx(tx *gorm.DB, order *models.Order) error {
	return tx.Save(order).Error
}

//UpdateOrderPanelNegotiation updates the negotiation status and related fields of an order panel within a transaction.
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

// UpdateNegotiationHistoryTx updates an exisitng negotiation history record within a transaction.
func (r *Repository) UpdateNegotiationHistoryTx(tx *gorm.DB, negotiationHistory *models.NegotiationHistory) error {
	return tx.Save(negotiationHistory).Error
}

// BulkAcceptPanelsByGroupRangeTx updates the negotiation status of order panels within a specified group range to "accepted" within a transaction.
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

// CreateOrderPanelsBatchTx creates multiple order panels in a single batch operation within a transaction.
func (r *Repository) CreateOrderPanelsBatchTx(tx *gorm.DB, orderPanels []*models.OrderPanel) error {
	if len(orderPanels) == 0 {
		return nil
	}
	return tx.Create(&orderPanels).Error
}

// UpdateWorkOrderTx updates an existing work order within a transaction.
func (r *Repository) UpdateWorkOrderTx(tx *gorm.DB, workOrder *models.WorkOrder) error {
	return tx.Save(workOrder).Error
}
