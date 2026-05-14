package main

import (
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"sub2api-go/internal/config"
	"sub2api-go/internal/handler"
	"sub2api-go/internal/middleware"
	"sub2api-go/internal/service"
	"sub2api-go/internal/store"
	"sub2api-go/internal/telemetry"
)

func main() {
	// Load .env file if exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Load configuration
	cfg := config.Load()
	if err := cfg.ValidateProductionSecrets(); err != nil {
		log.Fatalf("Invalid production configuration: %v", err)
	}

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
	redisStore, err = store.NewRedisStore(cfg.RedisURL, sqliteStore)
	if err != nil {
		if cfg.IsProduction() {
			log.Fatalf("Redis required in production (set REDIS_URL and ensure Redis is reachable): %v", err)
		}
		if !cfg.AllowMemoryStore {
			log.Fatalf("Redis unavailable and ALLOW_MEMORY_STORE=false: %v", err)
		}
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
	chatHandler := handler.NewChatHandler(providerService, billingService, dataStore, cfg)
	adminHandler := handler.NewAdminHandler(dataStore)
	userHandler := handler.NewUserHandler(dataStore)
	paymentHandler := handler.NewPaymentHandler(stripeService, dataStore)
	authHandler := handler.NewAuthHandler(
		dataStore,
		cfg.JWTSecret,
		cfg.InviteCode,
		cfg.EmailVerifyEnabled,
		cfg.SMTPHost,
		cfg.SMTPPort,
		cfg.SMTPUsername,
		cfg.SMTPPassword,
		cfg.SMTPFrom,
	)
	dashboardHandler := handler.NewDashboardHandler(dataStore)

	// Setup Gin
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	if len(cfg.TrustedProxies) > 0 {
		if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
			log.Fatalf("Invalid TRUSTED_PROXIES: %v", err)
		}
		log.Printf("Trusted proxies (Gin): %v", cfg.TrustedProxies)
	}
	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.RequestIDMiddleware())
	r.Use(func(c *gin.Context) {
		p := c.Request.URL.Path
		if p == "/metrics" || p == "/health" || p == "/health/ready" || strings.HasPrefix(p, "/webhook/") {
			c.Next()
			return
		}
		telemetry.IncHTTPRequest()
		c.Next()
	})

	healthDeps := handler.HealthDeps{StoreType: storeType}
	if redisStore != nil {
		healthDeps.Redis = redisStore.Client()
	}
	if sqliteStore != nil {
		healthDeps.SQLite = sqliteStore.DB()
	}

	r.GET("/health", func(c *gin.Context) {
		handler.DetailedHealth(c, healthDeps)
	})
	r.GET("/health/ready", func(c *gin.Context) {
		handler.ReadyHealth(c, healthDeps)
	})
	r.GET("/metrics", gin.WrapH(telemetry.MetricsHandler()))

	// Auth endpoints (no auth required)
	auth := r.Group("/auth")
	if redisStore != nil {
		auth.Use(middleware.AuthRateLimitMiddleware(redisStore.Client()))
	}
	{
		auth.GET("/config", authHandler.AuthConfig)
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/send-register-code", authHandler.SendRegisterCode)
		auth.POST("/send-reset-password-code", authHandler.SendResetPasswordCode)
		auth.POST("/reset-password", authHandler.ResetPassword)
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
		dashboard.GET("/usage-daily", dashboardHandler.GetUsageDaily)
		dashboard.GET("/request-logs", dashboardHandler.ListRequestLogs)
	}

	// OpenAI compatible endpoints (require API key auth + rate limit + IP whitelist)
	v1 := r.Group("/v1")
	v1.Use(middleware.AuthMiddleware(dataStore))
	// 频次限制和 IP 白名单（Redis 不可用时中间件内部跳过计数，仅 IP 白名单仍生效）
	if redisStore != nil {
		v1.Use(middleware.RateLimitMiddleware(redisStore.Client(), cfg.RateLimitRedisFailOpen))
	} else {
		v1.Use(middleware.RateLimitMiddleware(nil, cfg.RateLimitRedisFailOpen))
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
	log.Printf("  Redis URL:  %s", cfg.RedisURL)
	log.Printf("  Memory fallback: %v (disallowed in production without Redis)", cfg.AllowMemoryStore)
	log.Printf("  Rate limit Redis fail-open: %v", cfg.RateLimitRedisFailOpen)
	log.Printf("  Allow unknown model pricing: %v", cfg.AllowUnknownModelPricing)
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
	log.Println("    GET  /health               - Health (JSON + dependency checks)")
	log.Println("    GET  /health/ready         - Readiness (503 if Redis down)")
	log.Println("    GET  /metrics              - OpenMetrics text (Prometheus scrape)")
	log.Println("    GET  /dashboard/usage-daily   - Daily consume aggregates (JWT)")
	log.Println("    GET  /dashboard/request-logs  - Recent chat audit rows (JWT)")
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
