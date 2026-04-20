package models

import "time"

type SparePartQuote struct {
	SparePartQuoteNo uint  `gorm:"type:int(11);primaryKey;not null;autoIncrement" json:"spare_part_quote_no"`
	OrderRequestNo   uint  `gorm:"type:int(11);not null" json:"order_request_no"`
	SupplierNo       *uint `gorm:"type:int(11);null" json:"supplier_no"`
	InsuranceNo      *uint `gorm:"type:int(11);null" json:"insurance_no"`

	CurrentRound          uint       `gorm:"type:tinyint(3);null" json:"current_round"`
	SupplierStatus        string     `gorm:"type:enum('waiting','processing','replied','delivering','received');not null" json:"supplier_status"`
	AvailableStock        *uint      `gorm:"type:int(11);null" json:"available_stock"`
	InitialUnitPrice      *uint      `gorm:"type:int(11);null" json:"initial_unit_price"`
	RequestedStock        *uint      `gorm:"type:int(11);null" json:"requested_stock"`
	RequestedUnitPrice    *uint      `gorm:"type:int(11);null" json:"requested_unit_price"`
	OrderedStock          *uint      `gorm:"type:int(11);null" json:"ordered_stock"`
	OrderedDate           *time.Time `gorm:"type:date;null" json:"ordered_date"`
	OrderedUnitPrice      *uint      `gorm:"type:int(11);null" json:"ordered_unit_price"`
	EstimatedDeliveryDate *time.Time `gorm:"type:date;null" json:"estimated_delivery_date"`
	CourierName           *string    `gorm:"type:varchar(50);null" json:"courier_name"`
	CourierTrackingNo     *string    `gorm:"type:varchar(50);null" json:"courier_tracking_no"`

	CreatedAt      time.Time  `gorm:"column:created_date;type:datetime;default:CURRENT_TIMESTAMP" json:"created_date"`
	CreatedBy      *uint      `gorm:"column:created_by;type:int(11);null" json:"created_by"`
	UpdatedAt      *time.Time `gorm:"column:last_modified_date;null;type:datetime;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"last_modified_date"`
	LastModifiedBy *uint      `gorm:"column:last_modified_by;null;type:int(11)" json:"last_modified_by"`

	OrderAndRequest    *OrderAndRequest `gorm:"foreignKey:OrderRequestNo;references:OrderRequestNo" json:"order_and_request,omitempty"`
	Supplier           *Supplier        `gorm:"foreignKey:SupplierNo;references:SupplierNo" json:"supplier,omitempty"`
	Insurance          *UserProfile     `gorm:"foreignKey:InsuranceNo;references:UserProfileNo" json:"insurance,omitempty"`
	CreatedByUser      *User            `gorm:"foreignKey:CreatedBy;references:UserNo" json:"created_by_user,omitempty"`
	LastModifiedByUser *User            `gorm:"foreignKey:LastModifiedBy;references:UserNo" json:"last_modified_by_user,omitempty"`

	SparePartNegotiationHistory []SparePartNegotiationHistory `gorm:"foreignKey:SparePartQuotesNo;references:SparePartQuoteNo" json:"spare_part_negotiation_history,omitempty"`
}

func (SparePartQuote) TableName() string {
	return "tr_spare_part_quotes"
}
