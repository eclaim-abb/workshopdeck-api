package models

import "time"

type Delivery struct {
	DeliveryNo       uint      `gorm:"type:int(11);primaryKey;autoIncrement;not null" json:"delivery_no"`
	ClientNo         uint      `gorm:"type:int(11);not null" json:"client_no"`
	DeliveryId       string    `gorm:"type:varchar(35);not null" json:"delivery_id"`
	LastRepairedDate time.Time `gorm:"type:datetime;not null" json:"last_repairerd_date"`
	DeliveredAt      time.Time `gorm:"date;not null" json:"delivered_at"`
	PhotoUrl         string    `gorm:"type:varchar(255);not null" json:"photo_url"`
	CreatedBy        *uint     `gorm:"column:created_by;type:int(11);null" json:"created_by"`
	CreatedAt        time.Time `gorm:"column:created_date;type:datetime;default:CURRENT_TIMESTAMP" json:"created_date"`
	UpdatedAt        time.Time `gorm:"column:last_modified_date;null;type:datetime;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"last_modified_date"`
	LastModifiedBy   *uint     `gorm:"column:last_modified_by;null;type:int(11)" json:"last_modified_by"`

	Client             *Client `gorm:"foreignKey:ClientNo;references:ClientNo" json:"client,omitempty"`
	CreatedByUser      *User   `gorm:"foreignKey:CreatedBy;references:UserNo" json:"created_by_user,omitempty"`
	LastModifiedByUser *User   `gorm:"foreignKey:LastModifiedBy;references:UserNo" json:"last_modified_by_user,omitempty"`

	Invoices []Invoice `gorm:"foreignKey:DeliveryNo;references:DeliveryNo" json:"invoices,omitempty"`
}

func (Delivery) TableName() string {
	return "tr_deliveries"
}
