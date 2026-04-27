package orders

import (
	"eclaim-workshop-deck-api/internal/common/response"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
)

// GetRepairedOrders retrieves repaired orders for a given workshop number.
func (h *Handler) GetRepairedOrders(c *gin.Context) {
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

	orders, err := h.service.GetRepairedOrders(uint(woID))

	for _, o := range orders {
		AttachFullPhotoURLs(&o, os.Getenv("BASE_URL"))
	}

	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Orders Retrieved Successfully", gin.H{"orders": orders})
}

// SetRepairedAsUnfinished marks a repaired vehicle as unfinished, allowing for further repairs or adjustments.
func (h *Handler) SetRepairedAsUnfinished(c *gin.Context) {
	var req CancelNegotiationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	order, err := h.service.SetRepairedAsUnfinished(req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Set Order as Unfinished Successfully", gin.H{"order": order})
}

// RemindPickup sends a reminder for pickup to the customer for an vehicle that has been repaired and is ready for pickup.
func (h *Handler) RemindPickup(c *gin.Context) {
	var req RemindPickupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	pickupReminders, err := h.service.RemindPickup(req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Remind Pickup done successfully", gin.H{"pickupReminders": pickupReminders})
}

// SetAsDelivered marks a repaired order as delivered, indicating that the customer has picked up the vehicle.
func (h *Handler) SetAsDelivered(c *gin.Context) {

}
