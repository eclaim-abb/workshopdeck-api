package invoices

import "github.com/gin-gonic/gin"

// RegisterRoutes sets up the routes for order-related endpoints, applying auth middleware to protect them.
func RegisterRoutes(router *gin.RouterGroup, handler *Handler, authMiddleware gin.HandlerFunc) {
	invoices := router.Group("/invoices")
	{
		invoices.Use(authMiddleware)
		{
			invoices.GET("/:id", handler.GetInvoices)
			invoices.POST("", handler.CreateInvoice)

			invoices.POST("/add-payment", handler.CreatePayment)
		}
	}
}
