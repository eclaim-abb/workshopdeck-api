package orders

import (
	"eclaim-workshop-deck-api/internal/common/response"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

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

// RequestSpareParts handles POST /repairing/spare-parts/request
//
// Expects a multipart/form-data body with:
//   - "data"    – JSON-encoded RequestOrderSparePartRequest
//   - "files[]" – one file per entry in req.Photos, indexed by SparePartPhotoMetadata.FileIndex
func (h *Handler) RequestSpareParts(c *gin.Context) {
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		response.Error(c, http.StatusBadRequest, "Failed to parse multipart form")
		return
	}

	dataStr := c.PostForm("data")
	if dataStr == "" {
		response.Error(c, http.StatusBadRequest, "Missing 'data' field in form")
		return
	}

	var req RequestOrderSparePartRequest
	if err := json.Unmarshal([]byte(dataStr), &req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid JSON in 'data' field: "+err.Error())
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

	order, err := h.service.RequestSparePart(&req, files, uploadFn)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusCreated, "spare part request created successfully", gin.H{"order": order})
}

func (h *Handler) OrderSpareParts(c *gin.Context) {
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		response.Error(c, http.StatusBadRequest, "Failed to parse multipart form")
		return
	}

	dataStr := c.PostForm("data")
	if dataStr == "" {
		response.Error(c, http.StatusBadRequest, "Missing 'data' field in form")
		return
	}

	var req RequestOrderSparePartRequest
	if err := json.Unmarshal([]byte(dataStr), &req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid JSON in 'data' field: "+err.Error())
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

	order, err := h.service.OrderSparePart(&req, files, uploadFn)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusCreated, "spare part order created successfully", gin.H{"order": order})
}

// GetSparePartsTracking handles GET /repairing/spare-parts/tracking
func (h *Handler) GetSparePartsTracking(c *gin.Context) {
	woIDStr := c.Query("workshop_no")
	if woIDStr == "" {
		response.Error(c, http.StatusBadRequest, "workshop_no is required")
		return
	}

	woID, err := strconv.ParseUint(woIDStr, 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid workshop_no format")
		return
	}

	tracking, err := h.service.GetSparePartsTracking(uint(woID))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Spare parts tracking retrieved successfully", gin.H{"tracking": tracking})
}
