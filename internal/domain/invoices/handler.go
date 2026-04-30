package invoices

import (
	"eclaim-workshop-deck-api/internal/common/response"
	"eclaim-workshop-deck-api/pkg/utils"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Handler struct {
	service *Service
	log     *zap.Logger
	storage *utils.LocalStorage
}

func NewHandler(
	service *Service,
	log *zap.Logger,
	storage *utils.LocalStorage,
) *Handler {
	return &Handler{service: service, log: log, storage: storage}
}

// GetInvoices gets invoices for a workshop
func (h *Handler) GetInvoices(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid user profile no")
		return
	}

	invoices, err := h.service.GetInvoices(uint(id))

	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Invoices Retrieved Successfully", gin.H{"invoices": invoices})
}

// CreateInvoice handles POST /invoices.
//
// The request is multipart/form-data with two fields:
//   - payload      — JSON string of CreateInvoiceRequest
//   - invoice_file — optional file (required when is_system_generated = false)
func (h *Handler) CreateInvoice(c *gin.Context) {
	uploadFn := func(file multipart.File, header *multipart.FileHeader, folder string) (string, error) {
		return h.storage.Upload(file, header, folder)
	}

	// ── 1. Parse the JSON payload field
	rawPayload := c.PostForm("payload")
	if rawPayload == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Missing required field: payload",
		})
		return
	}

	var req CreateInvoiceRequest
	if err := json.Unmarshal([]byte(rawPayload), &req); err != nil {
		h.log.Warn("CreateInvoice: failed to parse payload", zap.Error(err))
		response.Error(c, http.StatusBadRequest, "Invalid payload JSON: "+err.Error())
		return
	}

	// ── 2. Extract the optional file ───
	var fileHeader *multipart.FileHeader
	if !req.IsSystemGenerated {
		fh, err := c.FormFile("invoice_file")
		if err != nil {
			response.Error(c, http.StatusBadRequest, "invoice_file is required when is_system_generated is false")
			return
		}
		fileHeader = fh
	}

	// ── 3. Delegate to service ─────────
	invoice, err := h.service.CreateInvoice(req, fileHeader, uploadFn)
	if err != nil {
		h.log.Error("CreateInvoice: service error", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, err.Error())

		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Invoice created successfully",
		"data":    invoice,
	})
}

func (h *Handler) CreatePayment(c *gin.Context) {
	uploadFn := func(file multipart.File, header *multipart.FileHeader, folder string) (string, error) {
		return h.storage.Upload(file, header, folder)
	}

	// ── 1. Parse the JSON payload field
	rawPayload := c.PostForm("payload")
	if rawPayload == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Missing required field: payload",
		})
		return
	}

	var req AddPaymentRequest
	if err := json.Unmarshal([]byte(rawPayload), &req); err != nil {
		h.log.Warn("CreatePayment: failed to parse payload", zap.Error(err))
		response.Error(c, http.StatusBadRequest, "Invalid payload JSON: "+err.Error())

		return
	}

	// ── 2. Extract the proof file ───
	var fileHeader *multipart.FileHeader
	fh, err := c.FormFile("payment_file")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "payment file is required: "+err.Error())

		return
	}
	fileHeader = fh

	// ── 3. Delegate to service ─────────
	paymentRecord, err := h.service.CreatePayment(req, fileHeader, uploadFn)
	if err != nil {
		h.log.Error("CreatePayment: service error", zap.Error(err))
		response.Error(c, http.StatusInternalServerError, err.Error())

		return
	}

	AttachFullPhotoURLs(paymentRecord, "http://localhost:4124")

	response.Success(c, http.StatusCreated, "Payment Record Created Successfully", gin.H{"payment_record": paymentRecord})
}
