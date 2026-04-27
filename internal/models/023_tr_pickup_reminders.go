package models

import "time"

type PickupReminder struct {
	PickupReminderNo        uint      `gorm:"type:int(11);primaryKey;not null" json:"pickup_reminder_no"`
	OrderNo                 uint      `gorm:"type:int(11);not null" json:"order_no"`
	NextAvailableRemindDate time.Time `gorm:"type:datetime;default:CURRENT_TIMESTAMP;not null" json:"next_available_remind_date"`
	CreatedAt               time.Time `gorm:"column:created_date;type:datetime;default:CURRENT_TIMESTAMP" json:"created_date"`
	CreatedBy               *uint     `gorm:"column:created_by;type:int(11);not null" json:"created_by"`

	Order         Order `gorm:"foreignKey:OrderNo;references:OrderNo;" json:"order"`
	CreatedByUser User  `gorm:"foreignKey:CreatedBy;references:UserNo;" json:"created_by_user,omitempty"`
}

func (PickupReminder) TableName() string {
	return "tr_pickup_reminders"
}
