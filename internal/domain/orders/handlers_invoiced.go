package orders

import (
	"eclaim-workshop-deck-api/internal/common/response"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
)

func (h *Handler) GetInvoicedOrders(c *gin.Context) {
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

	orders, err := h.service.GetInvoicedOrders(uint(woID))

	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	for _, o := range orders {
		AttachFullPhotoURLs(&o, os.Getenv("BASE_URL"))
	}
	response.Success(c, http.StatusOK, "Orders Retrieved Successfully", gin.H{"orders": orders})
}
