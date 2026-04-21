package models

import "time"

// UserToken maps to tr_user_tokens table
type UserToken struct {
	UserTokenNo      uint       `gorm:"primaryKey;autoIncrement;column:user_token_no" json:"user_token_no"`
	UserNo           uint       `gorm:"not null;column:user_no" json:"user_no"`
	UserToken        string     `gorm:"not null;column:user_token" json:"user_token"`
	ExpiryDate       time.Time  `gorm:"not null;column:expiry_date" json:"expiry_date"`
	CreatedDate      time.Time  `gorm:"autoCreateTime;column:created_date" json:"created_date"`
	CreatedBy        uint       `gorm:"not null;column:created_by" json:"created_by"`
	LastModifiedDate *time.Time `gorm:"column:last_modified_date" json:"last_modified_date"`
	LastModifiedBy   *uint      `gorm:"column:last_modified_by" json:"last_modified_by"`

	// Association
	User User `gorm:"foreignKey:UserNo;references:UserNo" json:"-"`
}

func (UserToken) TableName() string {
	return "tr_user_tokens"
}
