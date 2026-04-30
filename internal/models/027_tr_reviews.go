package models

import "time"

type Review struct {
	ReviewNo              uint       `gorm:"primaryKey;autoIncrement;not null;type:int(11)" json:"review_no"`
	WorkshopNo            uint       `gorm:"type:int(11);not null" json:"workshop_no"`
	Rating                uint       `gorm:"type:tinyint(2);not null" json:"rating"`
	ReviewerName          string     `gorm:"type:varchar(255);not null" json:"reviewer_name"`
	ReviewText            string     `gorm:"type:mediumtext;not null" json:"review_text"`
	CreatedAt             time.Time  `gorm:"column:created_date;type:datetime;default:CURRENT_TIMESTAMP" json:"created_date"`
	ReplyText             string     `gorm:"type:mediumtext;null" json:"reply_text"`
	ReplyCreatedDate      *time.Time `gorm:"type:datetime;null" json:"reply_created_date"`
	ReplyCreatedBy        *uint      `gorm:"type:int(11);null" json:"reply_created_by"`
	ReplyLastModifiedDate *time.Time `gorm:"type:datetime;null" json:"reply_last_modified_date"`
	ReplyLastModifiedBy   *uint      `gorm:"type:int(11);null" json:"reply_last_modified_by"`

	Workshop                *UserProfile `gorm:"foreignKey:WorkshopNo;references:UserProfileNo" json:"workshop,omitempty"`
	ReplyCreatedByUser      *User        `gorm:"foreignKey:ReplyCreatedBy;references:UserNo" json:"reply_created_by_user,omitempty"`
	ReplyLastModifiedByUser *User        `gorm:"foreignKey:ReplyLastModifiedBy;references:UserNo" json:"reply_last_modified_by_user,omitempty"`
}

func (Review) TableName() string {
	return "tr_reviews"
}
