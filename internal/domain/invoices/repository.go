package invoices

import (
	"eclaim-workshop-deck-api/internal/models"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetDB() *gorm.DB { return r.db }

// FindOrdersByNos fetches the orders matching the given order numbers.
func (r *Repository) FindOrdersByNos(orderNos []uint) ([]models.Order, error) {
	var orders []models.Order
	if err := r.db.
		Where("order_no IN ?", orderNos).
		Find(&orders).Error; err != nil {
		return nil, err
	}
	return orders, nil
}

// GetOrdersFromWorkshopNo retrieves all orders and their details based on the workshop_no.
func (r *Repository) GetOrdersFromWorkshopNo(workshopNo uint) ([]models.Order, error) {
	var orders []models.Order

	err := r.db.
		Where("tr_orders.is_locked = ? AND tr_orders.workshop_no = ?", 0, workshopNo).
		Order("tr_orders.order_no").
		Find(&orders).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
	}

	return orders, err
}

// GetInvoices gets invoices from an array of invoice nos
func (r *Repository) GetInvoices(invoiceNos []uint) ([]models.Invoice, error) {
	var invoices []models.Invoice
	if err := r.db.
		Where("invoice_no IN ?", invoiceNos).
		Find(&invoices).Error; err != nil {
		return nil, err
	}
	return invoices, nil
}

func (r *Repository) FindInvoiceById(invoiceNo uint) (*models.Invoice, error) {
	var invoice models.Invoice
	if err := r.db.
		Preload("InvoiceInstallments").
		Preload("InvoiceInstallments.PaymentRecords").
		Preload("PaymentRecords").
		Where("invoice_no = ?", invoiceNo).
		Find(&invoice).Error; err != nil {
		return nil, err
	}
	return &invoice, nil
}

func (r *Repository) FindInstallmentById(installmentNo uint) (*models.InvoiceInstallment, error) {
	var installment models.InvoiceInstallment
	if err := r.db.
		Where("installment_no = ?", installmentNo).
		Find(&installment).Error; err != nil {
		return nil, err
	}
	return &installment, nil
}

// CreateInvoiceWithInstallments runs everything in a single transaction:
//  1. Insert the Invoice row and capture the auto-incremented InvoiceNo.
//  2. Attach InvoiceNo to every InvoiceInstallment and bulk-insert them
//     (skipped when the slice is empty, i.e. single-payment invoice).
//  3. Update every Order in orderNos to point at the new InvoiceNo.
//  4. Reload the Invoice with its InvoiceInstallments preloaded and return it.
func (r *Repository) CreateInvoiceWithInstallments(
	invoice *models.Invoice,
	installments []models.InvoiceInstallment,
	orderNos []uint,
) (*models.Invoice, error) {
	err := r.db.Transaction(func(tx *gorm.DB) error {

		// ── 1. Generate doc number inside the transaction (race-safe) ─────────
		now := time.Now()
		prefix := fmt.Sprintf("INV/%d/%02d/", now.Year(), int(now.Month()))

		var last models.Invoice
		err := tx.Set("gorm:query_option", "FOR UPDATE").
			Where("invoice_doc_number LIKE ?", prefix+"%").
			Order("invoice_doc_number DESC").
			First(&last).Error

		var nextSeq int
		if errors.Is(err, gorm.ErrRecordNotFound) {
			nextSeq = 1
		} else if err != nil {
			return fmt.Errorf("failed to fetch last invoice sequence: %w", err)
		} else {
			parts := strings.Split(last.InvoiceDocNumber, "/")
			if len(parts) == 4 {
				if seq, parseErr := strconv.Atoi(parts[3]); parseErr == nil {
					nextSeq = seq + 1
				}
			}
			if nextSeq == 0 {
				nextSeq = 1 // fallback if parse fails
			}
		}

		invoice.InvoiceDocNumber = fmt.Sprintf("INV/%d/%02d/%06d",
			now.Year(), int(now.Month()), nextSeq)

		// ── 2. Create the invoice record ──────────────────────────────────────
		if err := tx.Create(invoice).Error; err != nil {
			return err
		}

		// ── 3. Insert installments (if any) ───────────────────────────────────
		if len(installments) > 0 {
			for i := range installments {
				installments[i].InvoiceNo = invoice.InvoiceNo
			}
			if err := tx.Create(&installments).Error; err != nil {
				return err
			}
		}

		// ── 4. Link orders to the new invoice ─────────────────────────────────
		if err := tx.Model(&models.Order{}).
			Where("order_no IN ?", orderNos).
			Update("invoice_no", invoice.InvoiceNo).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// ── 5. Reload with associations ───────────────────────────────────────────
	var created models.Invoice
	if err := r.db.
		Preload("InvoiceInstallments").
		Preload("Client").
		First(&created, invoice.InvoiceNo).Error; err != nil {
		return nil, err
	}

	return &created, nil
}

// CreatePaymentRecord creates payment record with transaction
func (r *Repository) CreatePaymentRecord(tx *gorm.DB, paymentRecord *models.PaymentRecord) error {
	return tx.Create(paymentRecord).Error
}

// UpdateInvoiceTx updates invoice with transaction
func (r *Repository) UpdateInvoiceTx(tx *gorm.DB, invoice *models.Invoice) error {
	return tx.Save(invoice).Error
}

// UpdateInstallmentTx updates installment with transaction
func (r *Repository) UpdateInstallmentTx(tx *gorm.DB, installment *models.InvoiceInstallment) error {
	return tx.Save(installment).Error
}
