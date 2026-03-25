package orders

import "github.com/gin-gonic/gin"

func RegisterRoutes(router *gin.RouterGroup, handler *Handler, authMiddleware gin.HandlerFunc) {
	orders := router.Group("/orders")
	{
		orders.Use(authMiddleware)
		{
			orders.GET("", handler.GetOrders)
			orders.POST("", handler.CreateOrder)

			orders.GET("/view/:id", handler.ViewOrderDetails)

			orders.POST("/propose", handler.ProposeAdditionalWork)
			orders.POST("/work-order", handler.CreateWorkOrder)

			incomingOrders := orders.Group("/incoming")
			{
				incomingOrders.GET("", handler.GetIncomingOrders)
				incomingOrders.POST("/negotiate", handler.SubmitNegotiation)
				incomingOrders.PUT("/accept/:id", handler.AcceptOrder)
				incomingOrders.PUT("/decline/:id", handler.DeclineOrder)
			}

			negotiatingOrders := orders.Group("/negotiating")
			{
				negotiatingOrders.GET("", handler.GetNegotiatingOrders)
				negotiatingOrders.PUT("/cancel", handler.CancelNegotiation)
				negotiatingOrders.PUT("/forward-additional", handler.ForwardAdditionalProposal)
			}

			repairingOrders := orders.Group("/repairing")
			{
				repairingOrders.GET("", handler.GetRepairingOrders)
				repairingOrders.PUT("/extend-deadline", handler.ExtendDeadline)
				repairingOrders.POST("/update-repair-status", handler.UpdateOrderPanelRepairStatus)
				repairingOrders.POST("/complete", handler.CompleteRepairs)

				// 	spareParts := repairingOrders.Group("/spare-parts")
				// 	{
				// 		spareParts.POST("/request", handler.RequestSpareParts)
				// 		spareParts.POST("/order", handler.OrderSpareParts)

				// 		sparePartsTracking := spareParts.Group("/tracking")
				// 		{
				// 			sparePartsTracking.GET("", handler.GetSparePartsTracking)
				// 			suppliers := sparePartsTracking.Group("/suppliers")
				// 			{
				// 				cancel := suppliers.Group("/cancel")
				// 				{
				// 					cancel.POST("/:id", handler.CancelSupplier)
				// 					cancel.POST("/overdue/:id", handler.CancelSupplierOverdue)
				// 					cancel.POST("/no-response/:id", handler.CancelSupplierOverdue)
				// 					cancel.POST("/remaining/:id", handler.CancelRemainingSuppliers)
				// 				}

				// 				suppliers.POST("/accept/:id", handler.AcceptSupplierOffer)
				// 				suppliers.POST("/negotiate/:id", handler.NegotiateOffer)
				// 			}
				// 		}
				// 	}
			}

			repairedOrders := orders.Group("/repaired")
			{
				repairedOrders.GET("", handler.GetRepairedOrders)
				// 	repairedOrders.POST("/set-unfinished", handler.SetRepairedAsUnfinished)
				// 	repairedOrders.POST("/remind", handler.RemindPickup)
				// 	repairedOrders.POST("/delivered", handler.SetAsDelivered)
			}

			deliveredOrders := orders.Group("/delivered")
			{
				deliveredOrders.GET("", handler.GetDeliveredOrders)
			}
		}
	}

	clients := router.Group("/clients")
	{
		clients.Use(authMiddleware)
		{
			clients.POST("", handler.AddClient)
		}
	}
}
