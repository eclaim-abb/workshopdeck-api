package models

import "time"

type RepairPhoto struct {
	RepairPhotoNo        uint      `gorm:"primaryKey;not null;autoIncrement;type:int(11)" json:"repair_photo_no"`
	RepairHistoryNo      *uint     `gorm:"null;type:int(11)" json:"repair_history_no"`
	NegotiationHistoryNo *uint     `gorm:"null;type:int(11)" json:"negotiation_history_no"`
	PhotoType            string    `gorm:"type:enum('initial','repair','replacement','epoxy','completed');not null" json:"photo_type"`
	PhotoCaption         string    `gorm:"type:varchar(255);null" json:"photo_caption"`
	PhotoUrl             string    `gorm:"type:varchar(255);not null" json:"photo_url"`
	IsLocked             bool      `gorm:"type:tinyint(1);default:0;not null" json:"is_locked"`
	CreatedAt            time.Time `gorm:"column:created_date;type:datetime;default:CURRENT_TIMESTAMP" json:"created_date"`
	CreatedBy            *uint     `gorm:"column:created_by;type:int(11);null" json:"created_by"`
	UpdatedAt            time.Time `gorm:"column:last_modified_date;null;type:datetime;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"last_modified_date"`
	LastModifiedBy       *uint     `gorm:"column:last_modified_by;null;type:int(11)" json:"last_modified_by"`

	RepairHistory      *RepairHistory      `gorm:"foreignKey:RepairHistoryNo;references:RepairHistoryNo" json:"repair_history"`
	NegotiationHistory *NegotiationHistory `gorm:"foreignKey:NegotiationHistoryNo;references:NegotiationHistoryNo" json:"negotiation_history"`
	CreatedByUser      *User               `gorm:"foreignKey:CreatedBy;references:UserNo" json:"created_by_user,omitempty"`
	LastModifiedByUser *User               `gorm:"foreignKey:LastModifiedBy;references:UserNo" json:"last_modified_by_user,omitempty"`
}

func (RepairPhoto) TableName() string {
	return "tr_repair_photos"
}
