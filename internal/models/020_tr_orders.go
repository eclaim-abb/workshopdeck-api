package models

import "time"

type Order struct {
	OrderNo        uint      `gorm:"primaryKey;type:int(11);not null;autoIncrement" json:"order_no"`
	WorkshopNo     uint      `gorm:"type:int(11);not null" json:"workshop_no"`
	InsuranceNo    *uint     `gorm:"type:int(11);null" json:"insurance_no"`
	InvoiceNo      *uint     `gorm:"type:int(11);null" json:"invoice_no"`
	Status         string    `gorm:"type:enum('draft','incoming','negotiating','repairing','declined','proposed_additional','additional_work','repaired','delivered','completed');not null" json:"status"`
	OrderType      string    `gorm:"type:enum('insurance','manual');not null" json:"order_type"`
	ClientNo       uint      `gorm:"type:int(11);not null" json:"client_no"`
	ClaimDetails   string    `gorm:"not null" json:"claim_details"`
	Eta            time.Time `gorm:"column:ETA;type:date;null" json:"eta"`
	Notes          *string   `gorm:"column:notes;type:varchar(255);null" json:"notes"`
	DiscountType   string    `gorm:"type:enum('fixed','percent');null" json:"discount_type"`
	Discount       float64   `gorm:"type:float;default:0;not null" json:"discount"`
	IsStarted      bool      `gorm:"type:tinyint(1);default:0;not null" json:"is_started"`
	CompletedAt    time.Time `gorm:"column:completed_at;type:datetime;not null" json:"completed_at"`
	IsLocked       bool      `gorm:"type:tinyint(1);default:0;not null" json:"is_locked"`
	CreatedAt      time.Time `gorm:"column:created_date;type:datetime;default:CURRENT_TIMESTAMP" json:"created_date"`
	CreatedBy      *uint     `gorm:"column:created_by;type:int(11);null" json:"created_by"`
	UpdatedAt      time.Time `gorm:"column:last_modified_date;null;type:datetime;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"last_modified_date"`
	LastModifiedBy *uint     `gorm:"column:last_modified_by;null;type:int(11)" json:"last_modified_by"`

	Workshop           *UserProfile     `gorm:"foreignKey:WorkshopNo;references:UserProfileNo" json:"workshop,omitempty"`
	Insurance          *UserProfile     `gorm:"foreignKey:InsuranceNo;references:UserProfileNo" json:"insurance,omitempty"`
	Invoice            *Invoice         `gorm:"foreignKey:InvoiceNo;references:InvoiceNo" json:"invoice,omitempty"`
	Client             *Client          `gorm:"foreignKey:ClientNo;references:ClientNo" json:"client,omitempty"`
	CreatedByUser      *User            `gorm:"foreignKey:CreatedBy;references:UserNo" json:"created_by_user,omitempty"`
	LastModifiedByUser *User            `gorm:"foreignKey:LastModifiedBy;references:UserNo" json:"last_modified_by_user,omitempty"`
	WorkOrders         []WorkOrder      `gorm:"foreignKey:OrderNo;references:OrderNo;" json:"work_orders"`
	PickupReminders    []PickupReminder `gorm:"foreignKey:OrderNo;references:OrderNo;" json:"pickup_reminders"`
}

func (Order) TableName() string {
	return "tr_orders"
}
