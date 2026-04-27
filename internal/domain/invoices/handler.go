package invoices

import (
	"eclaim-workshop-deck-api/pkg/utils"
	"encoding/json"
	"mime/multipart"
	"net/http"

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

// CreateInvoice handles POST /invoices.
//
// The request is multipart/form-data with two fields:
//   - payload      — JSON string of CreateInvoiceRequest
//   - invoice_file — optional file (required when is_system_generated = false)
func (h *Handler) CreateInvoice(c *gin.Context) {
	uploadFn := func(file multipart.File, header *multipart.FileHeader, folder string) (string, error) {
		return h.storage.Upload(file, header, folder)
	}

	// ── 1. Parse the JSON payload field ───────────────────────────────────────
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
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid payload JSON: " + err.Error(),
		})
		return
	}

	// ── 2. Extract the optional file ──────────────────────────────────────────
	var fileHeader *multipart.FileHeader
	if !req.IsSystemGenerated {
		fh, err := c.FormFile("invoice_file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": "invoice_file is required when is_system_generated is false",
			})
			return
		}
		fileHeader = fh
	}

	// ── 3. Delegate to service ────────────────────────────────────────────────
	invoice, err := h.service.CreateInvoice(req, fileHeader, uploadFn)
	if err != nil {
		h.log.Error("CreateInvoice: service error", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Invoice created successfully",
		"data":    invoice,
	})
}
