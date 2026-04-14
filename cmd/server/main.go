package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"eclaim-workshop-deck-api/internal/bootstrap"
	"eclaim-workshop-deck-api/internal/config"
	"eclaim-workshop-deck-api/internal/domain/admin"
	"eclaim-workshop-deck-api/internal/domain/auth"
	"eclaim-workshop-deck-api/internal/domain/authdemo"
	"eclaim-workshop-deck-api/internal/domain/location"
	"eclaim-workshop-deck-api/internal/domain/orders"
	"eclaim-workshop-deck-api/internal/domain/panels"
	"eclaim-workshop-deck-api/internal/domain/posts"
	"eclaim-workshop-deck-api/internal/domain/settings"
	"eclaim-workshop-deck-api/internal/domain/suppliers"
	"eclaim-workshop-deck-api/internal/domain/usermanagement"
	"eclaim-workshop-deck-api/internal/middleware"
	"eclaim-workshop-deck-api/pkg/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	cfg := config.LoadConfig()
	if err := cfg.Validate(); err != nil {
		log.Fatal("invalid config: ", err)
	}

	logger, err := config.NewLogger(cfg.Env)
	if err != nil {
		log.Fatal("failed to initialize logger: ", err)
	}
	defer logger.Sync()

	gin.SetMode(cfg.GinMode)

	db := config.ConnectDB(cfg)

	localStorage, err := utils.NewLocalStorage(cfg.UploadPath, cfg.BaseURL)
	if err != nil {
		logger.Fatal("failed to initialize local storage", zap.Error(err))
	}

	domains := bootstrap.InitDomains(db, cfg, logger, localStorage)

	r := gin.New()

	rateLimiter, err := middleware.RateLimiter("100-M") // 100 requests per minute per IP
	if err != nil {
		logger.Fatal("failed to initialize rate limiter", zap.Error(err))
	}

	r.Use(middleware.RequestID())
	r.Use(middleware.Logger(logger))
	r.Use(gin.Recovery())
	r.Use(middleware.CORSMiddleware(cfg.FrontendURLs))
	r.Use(rateLimiter)
	r.Use(middleware.SecurityHeaders(cfg.Env))

	r.StaticFS("/api/uploads", http.Dir(cfg.UploadPath))

	// Health check — outside /api
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now()})
	})

	api := r.Group("/api")
	api.Use(middleware.APIKeyMiddleware(db))

	authMiddleware := middleware.AuthMiddleware(cfg.JWTSecret)

	authdemo.RegisterRoutes(api, domains.AuthDemoHandler, authMiddleware)
	posts.RegisterRoutes(api, domains.PostsHandler, authMiddleware)
	auth.RegisterRoutes(api, domains.AuthHandler, authMiddleware)
	admin.RegisterRoutes(api, domains.AdminsHandler, authMiddleware)
	panels.RegisterRoutes(api, domains.PanelsHandler, authMiddleware)
	settings.RegisterRoutes(api, domains.SettingsHandler, authMiddleware)
	location.RegisterRoutes(api, domains.LocationHandler, authMiddleware)
	suppliers.RegisterRoutes(api, domains.SupplierHandler, authMiddleware)
	usermanagement.RegisterRoutes(api, domains.UserManagementHandler, authMiddleware)
	orders.RegisterRoutes(api, domains.OrdersHandler, authMiddleware)

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		logger.Info("server starting", zap.String("port", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server failed to start", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutdown signal received, draining in-flight requests...")

	// Give 10 seconds to finish remaining requests
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("forced shutdown due to timeout", zap.Error(err))
	}

	logger.Info("server shut down cleanly")
}
