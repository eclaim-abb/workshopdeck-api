package invoices

// InstallmentDTO mirrors InstallmentPayload from the frontend.
// PaymentAmount is always a resolved fixed currency amount.
type InstallmentDTO struct {
	InstallmentSequence uint   `json:"installment_sequence" binding:"required,min=1"`
	PaymentAmount       uint   `json:"payment_amount"       binding:"required,min=1"`
	DueDate             string `json:"due_date"             binding:"required"` // "YYYY-MM-DD"
}

// CreateInvoiceRequest is unmarshalled from the `payload` field of the
// multipart form. The actual invoice file (if any) is read separately via
// c.FormFile("invoice_file") in the handler.
type CreateInvoiceRequest struct {
	OrderNos      []uint `json:"order_nos"            binding:"required,min=1"`
	ClientNo      uint   `json:"client_no"            binding:"required"`
	DeliveryNo    uint   `json:"delivery_no"          binding:"required"`
	PaymentAmount uint   `json:"payment_amount"       binding:"required,min=1"`
	DueDate       string `json:"due_date"` // required when Installments is empty
	// InvoiceDocNumber   string           `json:"invoice_doc_number"   binding:"required"`
	ReferenceDocNumber string           `json:"reference_doc_number"`
	IsSystemGenerated  bool             `json:"is_system_generated"`
	Installments       []InstallmentDTO `json:"installments"` // empty = single payment
	CreatedBy          uint             `json:"created_by" binding:"required"`
}
