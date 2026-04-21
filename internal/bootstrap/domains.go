package bootstrap

import (
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
	"eclaim-workshop-deck-api/pkg/utils"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Domains struct {
	AuthDemoHandler       *authdemo.Handler
	PostsHandler          *posts.Handler
	AuthHandler           *auth.Handler
	AdminsHandler         *admin.Handler
	PanelsHandler         *panels.Handler
	SettingsHandler       *settings.Handler
	LocationHandler       *location.Handler
	SupplierHandler       *suppliers.Handler
	UserManagementHandler *usermanagement.Handler
	OrdersHandler         *orders.Handler
}

func InitDomains(db *gorm.DB, cfg *config.Config, log *zap.Logger, localStorage *utils.LocalStorage, dbAlt *gorm.DB) *Domains {
	// Auth Demo
	authDemoRepo := authdemo.NewRepository(db)
	authDemoService := authdemo.NewService(authDemoRepo, cfg.JWTSecret)
	authDemoHandler := authdemo.NewHandler(authDemoService, log)

	// Posts
	postsRepo := posts.NewRepository(db)
	postsService := posts.NewService(postsRepo)
	postsHandler := posts.NewHandler(postsService, log)

	// Auth
	authRepo := auth.NewRepository(db, dbAlt)
	authService := auth.NewService(authRepo, cfg.JWTSecret)
	authHandler := auth.NewHandler(authService, log)

	adminsRepo := admin.NewRepository(db)
	adminsService := admin.NewService(adminsRepo)
	adminsHandler := admin.NewHandler(adminsService, log)

	// Panels
	panelsRepo := panels.NewRepository(db)
	panelsService := panels.NewService(panelsRepo)
	panelsHandler := panels.NewHandler(panelsService, log)

	// Settings
	settingsRepo := settings.NewRepository(db)
	settingsService := settings.NewService(settingsRepo)
	settingsHandler := settings.NewHandler(settingsService, log)

	// Location
	locationRepo := location.NewRepository(db)
	locationService := location.NewService(locationRepo)
	locationHandler := location.NewHandler(locationService, log)

	// Suppliers
	supplierRepo := suppliers.NewRepository(db)
	supplierService := suppliers.NewService(supplierRepo)
	supplierHandler := suppliers.NewHandler(supplierService, log)

	// User Management
	userManagementRepo := usermanagement.NewRepository(db)
	userManagementService := usermanagement.NewService(userManagementRepo)
	userManagementHandler := usermanagement.NewHandler(userManagementService, log)

	// Orders
	ordersRepo := orders.NewRepository(db)
	ordersService := orders.NewService(ordersRepo)
	ordersHandler := orders.NewHandler(ordersService, log, localStorage)

	return &Domains{
		AuthDemoHandler:       authDemoHandler,
		PostsHandler:          postsHandler,
		AuthHandler:           authHandler,
		AdminsHandler:         adminsHandler,
		PanelsHandler:         panelsHandler,
		SettingsHandler:       settingsHandler,
		LocationHandler:       locationHandler,
		SupplierHandler:       supplierHandler,
		UserManagementHandler: userManagementHandler,
		OrdersHandler:         ordersHandler,
	}
}
