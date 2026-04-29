package invoices

import (
	"errors"
	"fmt"
	"math"
	"mime/multipart"
	"time"

	"eclaim-workshop-deck-api/internal/models"
	"eclaim-workshop-deck-api/pkg/utils"

	"gorm.io/gorm"
)

type Service struct {
	repo      *Repository
	jwtSecret string
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// CreateInvoice validates the request, optionally uploads the invoice file,
// then persists the invoice + installments + order linkage in one transaction.
func (s *Service) CreateInvoice(
	req CreateInvoiceRequest,
	fileHeader *multipart.FileHeader,
	uploadFn func(file multipart.File, header *multipart.FileHeader, folder string) (string, error),
) (*models.Invoice, error) {
	if req.CreatedBy == 0 {
		return nil, fmt.Errorf("created_by is required")
	}

	createdBy := req.CreatedBy

	// ── 1. Validate orders ────────────────────────────────────────────────────
	orders, err := s.repo.FindOrdersByNos(req.OrderNos)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch orders: %w", err)
	}
	if len(orders) != len(req.OrderNos) {
		return nil, errors.New("one or more order numbers are invalid")
	}
	for _, o := range orders {
		if o.InvoiceNo != nil {
			return nil, fmt.Errorf("order %d is already linked to an invoice", o.OrderNo)
		}
	}

	// ── 2. Validate & resolve installments ───────────────────────────────────
	var installments []models.InvoiceInstallment

	if len(req.Installments) > 0 {
		var installmentSum uint
		for _, inst := range req.Installments {
			installmentSum += inst.PaymentAmount
		}

		diff := math.Abs(float64(installmentSum) - float64(req.PaymentAmount))
		if diff > 1 {
			return nil, fmt.Errorf(
				"installment total (%d) does not match invoice payment amount (%d)",
				installmentSum, req.PaymentAmount,
			)
		}

		for _, inst := range req.Installments {
			dueDate, err := time.Parse("2006-01-02", inst.DueDate)
			if err != nil {
				return nil, fmt.Errorf(
					"invalid due_date for installment %d: expected YYYY-MM-DD",
					inst.InstallmentSequence,
				)
			}
			installments = append(installments, models.InvoiceInstallment{
				InstallmentSequence: inst.InstallmentSequence,
				PaymentAmount:       inst.PaymentAmount,
				DueDate:             dueDate,
				IsPaid:              false,
				CreatedBy:           &createdBy,
			})
		}
	} else {
		if req.DueDate == "" {
			return nil, errors.New("due_date is required for single-payment invoices")
		}
		if _, err := time.Parse("2006-01-02", req.DueDate); err != nil {
			return nil, errors.New("invalid due_date format: expected YYYY-MM-DD")
		}
	}

	// ── 3. Validate & upload file (manual invoices only) ─────────────────────
	invoiceFileURL := ""
	if !req.IsSystemGenerated {
		if fileHeader == nil {
			return nil, errors.New("invoice_file is required when is_system_generated is false")
		}

		allowedTypes := map[string]bool{
			"application/pdf": true,
			"image/jpeg":      true,
			"image/jpg":       true,
			"image/png":       true,
			"image/webp":      true,
		}
		maxSize := int64(10 << 20) // 10 MB

		if fileHeader.Size > maxSize {
			return nil, fmt.Errorf("file %s exceeds the 10 MB limit", fileHeader.Filename)
		}
		contentType := fileHeader.Header.Get("Content-Type")
		if !allowedTypes[contentType] {
			return nil, fmt.Errorf("unsupported file type %s", contentType)
		}

		folder := fmt.Sprintf(
			"invoices/%d/%d%02d%02d",
			createdBy,
			time.Now().Year(),
			time.Now().Month(),
			time.Now().Day(),
		)

		file, err := fileHeader.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open invoice file: %w", err)
		}
		defer file.Close()

		invoiceFileURL, err = uploadFn(file, fileHeader, folder)
		if err != nil {
			return nil, fmt.Errorf("failed to upload invoice file: %w", err)
		}
	}

	// ── 4. Build the Invoice model ────────────────────────────────────────────
	invoice := &models.Invoice{
		ClientNo:           req.ClientNo,
		InvoiceDocNumber:   "",
		ReferenceDocNumber: req.ReferenceDocNumber,
		PaymentStatus:      "unpaid",
		PaymentAmount:      req.PaymentAmount,
		InvoiceFileUrl:     invoiceFileURL,
		IsLocked:           false,
		CreatedBy:          &createdBy,
	}

	if len(req.Installments) == 0 && req.DueDate != "" {
		dueDate, err := time.Parse("2006-01-02", req.DueDate)
		if err != nil {
			return nil, errors.New("invalid due_date format: expected YYYY-MM-DD")
		}

		invoice.SingleDueDate = &dueDate
	}
	// ── 5. Persist in one transaction ─────────────────────────────────────────
	created, err := s.repo.CreateInvoiceWithInstallments(invoice, installments, req.OrderNos)
	if err != nil {
		return nil, fmt.Errorf("failed to persist invoice: %w", err)
	}

	return created, nil
}

// CreatePayment validates the request, creates a payment record and uploads the payment proof file,
func (s *Service) CreatePayment(
	req AddPaymentRequest,
	fileHeader *multipart.FileHeader,
	uploadFn func(file multipart.File, header *multipart.FileHeader, folder string) (string, error),
) (*models.PaymentRecord, error) {
	if req.CreatedBy == 0 {
		return nil, fmt.Errorf("created_by is required")
	}

	if req.PaidDate == "" {
		return nil, fmt.Errorf("paid date is required")
	}

	createdBy := req.CreatedBy
	now := time.Now().Local()

	if req.InvoiceNo == 0 {
		return nil, fmt.Errorf("invoice_no is required")
	}

	invoice, err := s.repo.FindInvoiceById(req.InvoiceNo)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch invoice %d: %w", req.InvoiceNo, err)
	}

	allInstallments := invoice.InvoiceInstallments
	var invoiceInstallment *models.InvoiceInstallment
	if req.InstallmentNo != nil && *req.InstallmentNo != 0 && len(allInstallments) > 0 {
		for idx := range allInstallments {
			if allInstallments[idx].InstallmentNo == *req.InstallmentNo {
				invoiceInstallment = &allInstallments[idx] // ✅ points into the slice
				break
			}
		}

		if invoiceInstallment == nil {
			return nil, fmt.Errorf("installment %d not found", *req.InstallmentNo)
		}
	}

	// ── Validate & upload file ─────────────────────
	paymentFileURL := ""
	if fileHeader == nil {
		return nil, errors.New("payment_file is required when is_system_generated is false")
	}

	allowedTypes := map[string]bool{
		"application/pdf": true,
		"image/jpeg":      true,
		"image/jpg":       true,
		"image/png":       true,
		"image/webp":      true,
	}
	maxSize := int64(10 << 20) // 10 MB

	if fileHeader.Size > maxSize {
		return nil, fmt.Errorf("file %s exceeds the 10 MB limit", fileHeader.Filename)
	}
	contentType := fileHeader.Header.Get("Content-Type")
	if !allowedTypes[contentType] {
		return nil, fmt.Errorf("unsupported file type %s", contentType)
	}

	folder := fmt.Sprintf(
		"invoices/%d/payments/%d/%d%02d%02d",
		req.InvoiceNo,
		createdBy,
		time.Now().Year(),
		time.Now().Month(),
		time.Now().Day(),
	)

	file, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open invoice file: %w", err)
	}
	defer file.Close()

	paymentFileURL, err = uploadFn(file, fileHeader, folder)
	if err != nil {
		return nil, fmt.Errorf("failed to upload invoice file: %w", err)
	}

	paidDate, err := time.Parse("2006-01-02", req.PaidDate)
	paymentRecord := &models.PaymentRecord{
		InvoiceNo:       req.InvoiceNo,
		PaidAmount:      req.PaymentAmount,
		PaidDate:        paidDate,
		CreatedBy:       &req.CreatedBy,
		CreatedAt:       now,
		PaymentProofUrl: paymentFileURL,
	}

	if req.InstallmentNo != nil && *req.InstallmentNo != 0 {
		paymentRecord.InstallmentNo = req.InstallmentNo
	}

	if err = utils.WithTransaction(s.repo, func(tx *gorm.DB) error {
		err := s.repo.CreatePaymentRecord(tx, paymentRecord)
		if err != nil {
			return fmt.Errorf("failed to create payment record for invoice no %d", req.InvoiceNo)
		}

		if req.IsFullPayment {
			if req.InstallmentNo != nil {
				invoiceInstallment.IsPaid = true
				invoiceInstallment.PaidDate = paidDate
				invoiceInstallment.LastModifiedBy = &req.CreatedBy

				err = s.repo.UpdateInstallmentTx(tx, invoiceInstallment)
				if err != nil {
					return fmt.Errorf("failed to update invoice installment %d: %w", invoiceInstallment.InstallmentNo, err)
				}

				allPaid := true
				for _, i := range allInstallments {
					if i.IsPaid == false {
						allPaid = false
						break
					}
				}

				if allPaid {
					invoice.PaymentStatus = "paid"
					invoice.LastModifiedBy = &req.CreatedBy
				} else {
					if invoice.PaymentStatus == "unpaid" {
						invoice.PaymentStatus = "partial"
					}
				}

				err = s.repo.UpdateInvoiceTx(tx, invoice)
				if err != nil {
					return fmt.Errorf("failed to update invoice %d: %w", invoice.InvoiceNo, err)
				}
			} else {
				invoice.PaymentStatus = "paid"
				invoice.LastModifiedBy = &req.CreatedBy

				err = s.repo.UpdateInvoiceTx(tx, invoice)
				if err != nil {
					return fmt.Errorf("failed to update invoice %d: %w", invoice.InvoiceNo, err)
				}
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return paymentRecord, nil
}

// GetInvoices gets invoices based on the user profile no
func (s *Service) GetInvoices(userProfileNo uint) ([]models.Invoice, error) {
	allOrders, err := s.repo.GetOrdersFromWorkshopNo(userProfileNo)
	if err != nil {
		return nil, fmt.Errorf("no orders for workshop %d found: %w", userProfileNo, err)
	}

	var invoiceNos []uint
	if len(allOrders) > 0 {
		for _, aO := range allOrders {
			if aO.InvoiceNo != nil {
				invoiceNos = append(invoiceNos, *aO.InvoiceNo)
			}
		}
	}

	var allInvoices []models.Invoice
	if len(invoiceNos) > 0 {
		allInvoices, err = s.repo.GetInvoices(invoiceNos)
		if err != nil {
			return nil, fmt.Errorf("problem retrieving invoices: %w", err)
		}
	}

	return allInvoices, nil
}
