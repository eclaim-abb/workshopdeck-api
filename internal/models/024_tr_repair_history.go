package models

import "time"

type RepairHistory struct {
	RepairHistoryNo uint      `gorm:"primaryKey;not null;autoIncrement;type:int(11)" json:"repair_history_no"`
	OrderPanelNo    uint      `gorm:"not null;type:int(11)" json:"order_panel_no"`
	Status          string    `gorm:"column:repair_status;type:enum('incomplete','completed','requesting','ordering');not null" json:"repair_status"`
	Note            string    `gorm:"type:varchar(255);null" json:"note"`
	CreatedAt       time.Time `gorm:"column:created_date;type:datetime;default:CURRENT_TIMESTAMP" json:"created_date"`
	CreatedBy       *uint     `gorm:"column:created_by;type:int(11);null" json:"created_by"`
	UpdatedAt       time.Time `gorm:"column:last_modified_date;null;type:datetime;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"last_modified_date"`
	LastModifiedBy  *uint     `gorm:"column:last_modified_by;null;type:int(11)" json:"last_modified_by"`

	OrderPanel         *OrderPanel `gorm:"foreignKey:OrderPanelNo;references:OrderPanelNo;" json:"order_panel,omitempty"`
	CreatedByUser      *User       `gorm:"foreignKey:CreatedBy;references:UserNo" json:"created_by_user,omitempty"`
	LastModifiedByUser *User       `gorm:"foreignKey:LastModifiedBy;references:UserNo" json:"last_modified_by_user,omitempty"`

	OrderAndRequests  []OrderAndRequest `gorm:"foreignKey:RepairHistoryNo;references:RepairHistoryNo" json:"order_and_requests,omitempty"`
	RepairPhotos      []RepairPhoto     `gorm:"foreignKey:RepairHistoryNo;references:RepairHistoryNo" json:"repair_photos,omitempty"`
	OrdersAndRequests []OrderAndRequest `gorm:"foreignKey:RepairHistoryNo;references:RepairHistoryNo" json:"orders_and_requests,omitempty"`
}

func (RepairHistory) TableName() string {
	return "tr_repair_history"
}
