package models

import "time"

type SparePartNegotiationHistory struct {
	SparePartNegotiationHistoryNo uint       `gorm:"type:int(11);primaryKey;not null;autoIncrement" json:"spare_part_negotiation_history_no"`
	SparePartQuotesNo             uint       `gorm:"type:int(11);not null" json:"spare_part_quotes_no"`
	RoundCount                    uint       `gorm:"type:tinyint(3);default:1;not null" json:"round_count"`
	OldRequestedStock             *uint      `gorm:"type:int(11);null" json:"old_requested_stock"`
	OldUnitPrice                  *uint      `gorm:"null;type:int(11)" json:"old_unit_price"`
	NewRequestedStock             *uint      `gorm:"null;type:int(11)" json:"new_requested_stock"`
	NewUnitPrice                  *uint      `gorm:"null;type:int(11)" json:"new_unit_price"`
	EstimatedDeliveryDate         *time.Time `gorm:"null;type:date" json:"estimated_delivery_date"`
	Status                        string     `gorm:"not null;type:enum('pending','accepted','rejected')" json:"status"`
	CreatedAt                     time.Time  `gorm:"column:created_date;type:datetime;default:CURRENT_TIMESTAMP" json:"created_date"`
	CreatedBy                     *uint      `gorm:"column:created_by;type:int(11);null" json:"created_by"`
	UpdatedAt                     *time.Time `gorm:"column:last_modified_date;null;type:datetime;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"last_modified_date"`
	LastModifiedBy                *uint      `gorm:"column:last_modified_by;null;type:int(11)" json:"last_modified_by"`

	SparePartQuote     *SparePartQuote `gorm:"foreignKey:SparePartQuotesNo;references:SparePartQuoteNo" json:"spare_part_quote,omitempty"`
	CreatedByUser      *User           `gorm:"foreignKey:CreatedBy;references:UserNo" json:"created_by_user,omitempty"`
	LastModifiedByUser *User           `gorm:"foreignKey:LastModifiedBy;references:UserNo" json:"last_modified_by_user,omitempty"`
}

func (SparePartNegotiationHistory) TableName() string {
	return "tr_spare_part_negotiation_history"
}
