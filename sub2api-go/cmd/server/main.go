package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"sub2api-go/internal/config"
	"sub2api-go/internal/handler"
	"sub2api-go/internal/middleware"
	"sub2api-go/internal/service"
	"sub2api-go/internal/store"
)

func main() {
	// Load .env file if exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Load configuration
	cfg := config.Load()

	// Initialize stores
	var dataStore store.Store
	var redisStore *store.RedisStore
	var sqliteStore *store.SQLiteStore
	var syncer *store.Syncer
	var storeType string

	// Initialize SQLite first (needed for user data)
	sqliteStore, err := store.NewSQLiteStore("./data/sub2api.db")
	if err != nil {
		log.Printf("SQLite not available: %v", err)
	}

	// Try to connect to Redis
	redisStore, err = store.NewRedisStore("redis://localhost:6379", sqliteStore)
	if err != nil {
		log.Printf("Redis not available: %v, using memory store", err)
		dataStore = store.NewMemoryStore()
		storeType = "memory"
	} else {
		dataStore = redisStore
		storeType = "redis"

		if sqliteStore != nil {
			storeType = "redis+sqlite"
			
			// Start syncer (sync every minute)
			syncer = store.NewSyncer(redisStore, sqliteStore, 1*time.Minute)
			syncer.Start()
		}
	}

	// Initialize services
	providerService := service.NewProviderService(cfg)
	billingService := service.NewBillingService(dataStore)
	
	// Initialize Stripe service (optional)
	var stripeService *service.StripeService
	if cfg.StripeSecretKey != "" {
		stripeService = service.NewStripeService(
			cfg.StripeSecretKey,
			cfg.StripeWebhookSecret,
			cfg.StripeSuccessURL,
			cfg.StripeCancelURL,
		)
		log.Println("Stripe payment enabled")
	}

	// Initialize handlers
	chatHandler := handler.NewChatHandler(providerService, billingService, dataStore)
	adminHandler := handler.NewAdminHandler(dataStore)
	userHandler := handler.NewUserHandler(dataStore)
	paymentHandler := handler.NewPaymentHandler(stripeService, dataStore)
	authHandler := handler.NewAuthHandler(dataStore, cfg.JWTSecret, cfg.InviteCode)
	dashboardHandler := handler.NewDashboardHandler(dataStore)

	// Setup Gin
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.RequestIDMiddleware())

	// Health endpoint
	r.GET("/health", func(c *gin.Context) {
		handler.HealthHandlerWithStore(c, storeType)
	})

	// Auth endpoints (no auth required)
	auth := r.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
	}

	// User dashboard endpoints (require JWT auth)
	dashboard := r.Group("/dashboard")
	dashboard.Use(authHandler.JWTAuthMiddleware())
	{
		dashboard.GET("/me", authHandler.GetMe)
		dashboard.GET("/keys", func(c *gin.Context) {
			userID, _ := c.Get("user_id")
			keys, err := dataStore.ListKeys(c.Request.Context(), userID.(string))
			if err != nil {
				c.JSON(500, gin.H{"error": "Failed to list keys"})
				return
			}
			c.JSON(200, gin.H{"keys": keys})
		})
		// Key 管理（创建 / 更新 / 删除）
		dashboard.POST("/keys", dashboardHandler.CreateKey)
		dashboard.PATCH("/keys/:id", dashboardHandler.UpdateKeySettings)
		dashboard.DELETE("/keys/:id", dashboardHandler.DeleteKey)
	}

	// OpenAI compatible endpoints (require API key auth + rate limit + IP whitelist)
	v1 := r.Group("/v1")
	v1.Use(middleware.AuthMiddleware(dataStore))
	// 频次限制和 IP 白名单（仅在 Redis 可用时启用）
	if redisStore != nil {
		v1.Use(middleware.RateLimitMiddleware(redisStore.Client()))
	} else {
		v1.Use(middleware.RateLimitMiddleware(nil))
	}
	{
		v1.POST("/chat/completions", chatHandler.ChatCompletions)
		v1.GET("/models", userHandler.GetModels)
		v1.GET("/usage", userHandler.GetUsage)
		v1.GET("/transactions", userHandler.GetTransactions)
		// Payment endpoints
		v1.POST("/payment/checkout", paymentHandler.CreateCheckout)
		v1.GET("/payment/status/:session_id", paymentHandler.GetPaymentStatus)
	}

	// Stripe webhook (no auth - verified by signature)
	r.POST("/webhook/stripe", paymentHandler.HandleWebhook)

	// Admin endpoints (require admin key)
	admin := r.Group("/admin")
	admin.Use(middleware.AdminAuthMiddleware(cfg))
	{
		admin.POST("/keys", adminHandler.CreateKey)
		admin.GET("/keys", adminHandler.ListKeys)
		admin.GET("/keys/:id", adminHandler.GetKey)
		admin.POST("/keys/:id/topup", adminHandler.TopupKey)
	}

	// Print startup info
	log.Println("===========================================")
	log.Println("  Sub2API Server Starting")
	log.Println("===========================================")
	log.Printf("  Port:       %s", cfg.Port)
	log.Printf("  Store:      %s", storeType)
	log.Printf("  Providers:  %d configured", len(cfg.Providers))
	for _, p := range cfg.Providers {
		log.Printf("    - %s: %d models", p.Name, len(p.Models))
	}
	log.Println("===========================================")
	log.Println("  Endpoints:")
	log.Println("    POST /v1/chat/completions  - Chat API")
	log.Println("    GET  /v1/models            - List models")
	log.Println("    GET  /v1/usage             - Get usage stats")
	log.Println("    GET  /v1/transactions      - Get transaction history")
	log.Println("    POST /v1/payment/checkout  - Create Stripe checkout")
	log.Println("    GET  /v1/payment/status    - Check payment status")
	log.Println("    POST /admin/keys           - Create API key")
	log.Println("    GET  /admin/keys           - List API keys")
	log.Println("    POST /admin/keys/:id/topup - Topup balance")
	log.Println("    POST /webhook/stripe       - Stripe webhook")
	log.Println("    GET  /health               - Health check")
	log.Println("===========================================")

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		log.Println("Shutting down...")
		
		if syncer != nil {
			syncer.Stop()
		}
		if sqliteStore != nil {
			sqliteStore.Close()
		}
		if redisStore != nil {
			redisStore.Close()
		}

		os.Exit(0)
	}()

	// Start server
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
