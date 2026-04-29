package models

import "time"

type Invoice struct {
	InvoiceNo          uint       `gorm:"type:int(11);primaryKey;autoIncrement;not null" json:"invoice_no"`
	DeliveryNo         *uint      `gorm:"type:int(11);null" json:"delivery_no"`
	ClientNo           uint       `gorm:"type:int(11);not null" json:"client_no"`
	InvoiceDocNumber   string     `gorm:"type:varchar(35);not null" json:"invoice_doc_number"`
	ReferenceDocNumber string     `gorm:"type:varchar(35);null" json:"reference_doc_number"`
	PaymentStatus      string     `gorm:"type:enum('draft','unpaid','void','partial',paid');not null" json:"payment_status"`
	PaymentAmount      uint       `gorm:"type:int(11);not null" json:"payment_amount"`
	SingleDueDate      *time.Time `gorm:"type:datetime;column:single_due_date;" json:"single_due_date"`
	InvoiceFileUrl     string     `gorm:"type:varchar(255);not null" json:"invoice_file_url"`
	IsLocked           bool       `gorm:"type:tinyint(1);default:0;not null" json:"is_locked"`
	CreatedAt          time.Time  `gorm:"column:created_date;type:datetime;default:CURRENT_TIMESTAMP" json:"created_date"`
	CreatedBy          *uint      `gorm:"column:created_by;type:int(11);null" json:"created_by"`
	UpdatedAt          time.Time  `gorm:"column:last_modified_date;null;type:datetime;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"last_modified_date"`
	LastModifiedBy     *uint      `gorm:"column:last_modified_by;null;type:int(11)" json:"last_modified_by"`

	Delivery           *Delivery `gorm:"foreignKey:DeliveryNo;references:DeliveryNo" json:"delivery,omitempty"`
	Client             *Client   `gorm:"foreignKey:ClientNo;references:ClientNo" json:"client,omitempty"`
	CreatedByUser      *User     `gorm:"foreignKey:CreatedBy;references:UserNo" json:"created_by_user,omitempty"`
	LastModifiedByUser *User     `gorm:"foreignKey:LastModifiedBy;references:UserNo" json:"last_modified_by_user,omitempty"`

	InvoiceInstallments []InvoiceInstallment `gorm:"foreignKey:InvoiceNo;references:InvoiceNo" json:"invoice_installments,omitempty"`
	PaymentRecords      []PaymentRecord      `gorm:"foreignKey:InvoiceNo;references:InvoiceNo" json:"payment_records,omitempty"`
}

func (Invoice) TableName() string {
	return "tr_invoices"
}
