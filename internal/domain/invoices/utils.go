package invoices

import (
	"eclaim-workshop-deck-api/internal/models"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

// GetLastInvoiceDocNumberForMonth retrieves the highest sequential number
// used in invoice doc numbers for the given year and month prefix.
// e.g. prefix = "INV/2026/04/"
func (r *Repository) GetLastInvoiceSequenceForMonth(year int, month int) (int, error) {
	prefix := fmt.Sprintf("INV/%d/%02d/", year, month)

	var last models.Invoice
	err := r.db.
		Where("invoice_doc_number LIKE ?", prefix+"%").
		Order("invoice_doc_number DESC").
		First(&last).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	// Parse the sequence number from the tail: "INV/2026/04/0022" → "0022"
	parts := strings.Split(last.InvoiceDocNumber, "/")
	if len(parts) != 4 {
		return 0, nil
	}
	seq, err := strconv.Atoi(parts[3])
	if err != nil {
		return 0, nil
	}
	return seq, nil
}
