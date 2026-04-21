package auth

import "github.com/gin-gonic/gin"

func RegisterRoutes(router *gin.RouterGroup, handler *Handler, authMiddleware gin.HandlerFunc) {
	auth := router.Group("/auth")
	{
		// Public routes
		auth.GET("/get-user-by-email", handler.GetUserByEmail)

		auth.POST("/register", handler.Register)
		auth.POST("/login", handler.Login)
		auth.POST("/verify-2fa", handler.VerifyTwoFactor)
		auth.POST("/refresh", handler.RefreshToken)
		auth.POST("/reset-password", handler.ResetPassword)
		auth.POST("/generate-api-key", handler.GenerateAPIKey)
		auth.POST("/change-password", handler.ChangePassword)

		// Protected routes
		auth.Use(authMiddleware)
		{
			auth.PUT("/update", handler.UpdateAccount)
		}
	}
}
