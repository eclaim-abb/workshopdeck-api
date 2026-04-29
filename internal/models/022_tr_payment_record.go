package models

import "time"

type PaymentRecord struct {
	PaymentRecordNo uint      `gorm:"type:int(11);primaryKey;not null;autoIncrement" json:"payment_record_no"`
	InvoiceNo       uint      `gorm:"type:int(11);not null" json:"invoice_no"`
	InstallmentNo   *uint     `gorm:"type:int(11);null" json:"installment_no"`
	PaidAmount      uint      `gorm:"type:int(11);not null" json:"paid_amount"`
	PaidDate        time.Time `gorm:"default:CURRENT_TIMESTAMP;not null;type:datetime" json:"paid_date"`
	PaymentProofUrl string    `gorm:"type:varchar(255);not null" json:"payment_proof_url"`
	CreatedAt       time.Time `gorm:"column:created_date;type:datetime;default:CURRENT_TIMESTAMP" json:"created_date"`
	CreatedBy       *uint     `gorm:"column:created_by;type:int(11);null" json:"created_by"`
	UpdatedAt       time.Time `gorm:"column:last_modified_date;null;type:datetime;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"last_modified_date"`
	LastModifiedBy  *uint     `gorm:"column:last_modified_by;null;type:int(11)" json:"last_modified_by"`

	Invoice            *Invoice            `gorm:"foreignKey:InvoiceNo;references:InvoiceNo" json:"invoice,omitempty"`
	Installment        *InvoiceInstallment `gorm:"foreignKey:InstallmentNo;references:InstallmentNo" json:"installment,omitempty"`
	CreatedByUser      *User               `gorm:"foreignKey:CreatedBy;references:UserNo" json:"created_by_user,omitempty"`
	LastModifiedByUser *User               `gorm:"foreignKey:LastModifiedBy;references:UserNo" json:"last_modified_by_user,omitempty"`
}

func (PaymentRecord) TableName() string {
	return "tr_payment_record"
}
