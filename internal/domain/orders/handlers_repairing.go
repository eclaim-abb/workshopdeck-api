package orders

import (
	"eclaim-workshop-deck-api/internal/common/response"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

var errMissingDataField = &missingDataFieldError{}

type missingDataFieldError struct{}

func (e *missingDataFieldError) Error() string { return "missing 'data' field in form" }

// GetRepairingOrders retrieves repairing orders for a given workshop number.
func (h *Handler) GetRepairingOrders(c *gin.Context) {
	woIDStr := c.Query("workshop_no")

	if woIDStr == "" {
		response.Error(c, http.StatusBadRequest, "workshop no is needed")
		return
	}

	woID, err := strconv.ParseUint(woIDStr, 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid workshop no format")
		return
	}

	orders, err := h.service.GetRepairingOrders(uint(woID))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Orders Retrieved Successfully", gin.H{"orders": orders})
}

// GetSparePartsTracking retrieves the spare parts tracking information for a given order no.
func (h *Handler) GetSparePartsTracking(c *gin.Context) {
	orderIDStr := c.Query("order_no")
	if orderIDStr == "" {
		response.Error(c, http.StatusBadRequest, "order_no is needed")
		return
	}

	orderID, err := strconv.ParseUint(orderIDStr, 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid order_id format")
		return
	}

	tracking, err := h.service.GetSparePartsTracking(uint(orderID))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Tracking data retrieved successfully", gin.H{"tracking": tracking})
}

// ExtendDeadline extends the deadline of a repairing order.
func (h *Handler) ExtendDeadline(c *gin.Context) {
	var req ExtendDeadlineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	order, err := h.service.ExtendDeadline(req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusCreated, "deadline extended successfully", gin.H{"order": order})
}

// UpdateOrderPanelRepairStatus updates the repair status of an order panel, allowing for optional file uploads.
func (h *Handler) UpdateOrderPanelRepairStatus(c *gin.Context) {
	err := c.Request.ParseMultipartForm(32 << 20)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Failed to parse multipart form")
		return
	}

	dataStr := c.PostForm("data")
	if dataStr == "" {
		response.Error(c, http.StatusBadRequest, "Missing 'data' field in form")
		return
	}

	var req AddOrderPanelRepairStatus
	if err := json.Unmarshal([]byte(dataStr), &req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid JSON in 'data' field")
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Failed to get multipart form")
		return
	}
	files := form.File["files"]

	uploadFn := func(file multipart.File, header *multipart.FileHeader, folder string) (string, error) {
		return h.storage.Upload(file, header, folder)
	}

	order, err := h.service.UpdateOrderPanelRepairStatus(&req, files, uploadFn)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusCreated, "order panel's status updated successfully", gin.H{"order": order})
}

// CompleteRepairs marks a repairing order as completed, allowing for optional file uploads.
func (h *Handler) CompleteRepairs(c *gin.Context) {
	err := c.Request.ParseMultipartForm(32 << 20)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Failed to parse multipart form")
		return
	}

	dataStr := c.PostForm("data")
	if dataStr == "" {
		response.Error(c, http.StatusBadRequest, "Missing 'data' field in form")
		return
	}

	var req CompleteRepairsRequest
	if err := json.Unmarshal([]byte(dataStr), &req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid JSON in 'data' field")
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Failed to get multipart form")
		return
	}
	files := form.File["files"]

	uploadFn := func(file multipart.File, header *multipart.FileHeader, folder string) (string, error) {
		return h.storage.Upload(file, header, folder)
	}

	order, err := h.service.CompleteRepairs(&req, files, uploadFn)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusCreated, "repairs completed successfully", gin.H{"order": order})
}

// RequestSpareParts handles the request to create a spare part request for a repairing order, allowing for optional file uploads.
func (h *Handler) RequestSpareParts(c *gin.Context) {
	req, files, err := parseSparePartForm(c)
	if err != nil {
		if err == errMissingDataField {
			response.Error(c, http.StatusBadRequest, err.Error())
		} else {
			response.Error(c, http.StatusBadRequest, "Failed to parse form: "+err.Error())
		}
		return
	}

	uploadFn := func(file multipart.File, header *multipart.FileHeader, folder string) (string, error) {
		return h.storage.Upload(file, header, folder)
	}

	order, err := h.service.RequestSparePart(req, files, uploadFn)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusCreated, "spare part request created successfully", gin.H{"order": order})
}

// OrderSpareParts handles the request to create a spare part order for a repairing order, allowing for optional file uploads.
func (h *Handler) OrderSpareParts(c *gin.Context) {
	req, files, err := parseSparePartForm(c)
	if err != nil {
		if err == errMissingDataField {
			response.Error(c, http.StatusBadRequest, err.Error())
		} else {
			response.Error(c, http.StatusBadRequest, "Failed to parse form: "+err.Error())
		}
		return
	}

	uploadFn := func(file multipart.File, header *multipart.FileHeader, folder string) (string, error) {
		return h.storage.Upload(file, header, folder)
	}

	order, err := h.service.OrderSparePart(req, files, uploadFn)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusCreated, "spare part order created successfully", gin.H{"order": order})
}

// CancelSupplier handles the cancel supplier request for a spare part order.
func (h *Handler) CancelSupplier(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		response.Error(c, http.StatusBadRequest, "supplier id is needed")
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid supplier id format")
		return
	}

	_, err = h.service.GetSparePartsTracking(uint(id))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
}
