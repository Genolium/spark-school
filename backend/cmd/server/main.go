package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/spark-school/backend/internal/auth"
	"github.com/spark-school/backend/internal/bot"
	"github.com/spark-school/backend/internal/config"
	"github.com/spark-school/backend/internal/database"
	"github.com/spark-school/backend/internal/handlers"
	"github.com/spark-school/backend/internal/models"
	"github.com/spark-school/backend/internal/outbox"
	"github.com/spark-school/backend/internal/security"
	"github.com/spark-school/backend/internal/services"
)

func main() {
	cfg := config.Load()

	// Validate production security configuration
	for _, warning := range cfg.Validate() {
		log.Println("[SECURITY ALERT]", warning)
	}

	// Disable verbose debug mode in production
	isProd := os.Getenv("ENV") == "production" || os.Getenv("GIN_MODE") == "release" || (cfg.Domain != "localhost" && cfg.Domain != "127.0.0.1")
	if isProd {
		gin.SetMode(gin.ReleaseMode)
	}

	// Connect Database
	db := database.Init(cfg)

	// Services
	affiliateService := services.NewAffiliateService(db)
	botService := bot.NewBotService(cfg, db)

	// Outbox Dispatcher for guaranteed async event delivery (Telegram notifications, admin review cards)
	outboxDispatcher := outbox.NewDispatcher(db, botService.SendNotification, func(adminID int64, payload outbox.AdminAlertPayload) error {
		var receipt models.PaymentReceipt
		if err := db.First(&receipt, payload.ReceiptID).Error; err == nil {
			return botService.SendAdminReceiptCard(adminID, &receipt)
		}
		return nil
	})
	outboxDispatcher.Start()
	defer outboxDispatcher.Stop()

	// Note: Standalone Telegram bot worker runs via cmd/bot/main.go.
	// If RUN_EMBEDDED_BOT=true is set, run it embedded for simple development.
	if os.Getenv("RUN_EMBEDDED_BOT") == "true" {
		log.Println("Running embedded Telegram bot worker...")
		go botService.Start()
	}

	// Handlers
	authHandler := handlers.NewAuthHandler(db, cfg, affiliateService)
	affiliateHandler := handlers.NewAffiliateHandler(db, cfg, affiliateService)
	adminHandler := handlers.NewAdminHandler(db, cfg, affiliateService, botService.SendNotification, botService.CreateOneTimeInviteLink)
	promoHandler := handlers.NewPromoHandler(db)
	statsHandler := handlers.NewStatsHandler(db)
	receiptHandler := handlers.NewReceiptHandler(db, func(receipt *models.PaymentReceipt) error {
		if cfg.AdminTelegramID != 0 {
			return botService.SendAdminReceiptCard(cfg.AdminTelegramID, receipt)
		}
		return nil
	})

	// Gin Router
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Security Middleware: Headers & Body Payload Limit (2MB)
	router.Use(security.SecurityHeadersMiddleware())
	router.Use(security.MaxBodySizeMiddleware(2 * 1024 * 1024))

	// CORS Setup with strict origin whitelisting
	allowedOrigins := []string{
		"http://localhost:3000",
		"http://127.0.0.1:3000",
		"https://" + cfg.Domain,
		"https://www." + cfg.Domain,
	}
	router.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length", "Retry-After"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// API v1 Routes with general Rate Limiting (120 req/min per IP)
	api := router.Group("/api/v1")
	api.Use(security.RateLimitMiddleware(120, 1*time.Minute, "Превышен лимит запросов к API. Пожалуйста, подождите."))
	{
		// Healthcheck
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status": "ok",
				"time":   time.Now().Format(time.RFC3339),
			})
		})

		// Public Stats: Live places counter with auto-doubling capacity
		api.GET("/stats/places", statsHandler.GetPlacesStats)

		// Public Receipts: Anti-fraud receipt submission & deduplication check
		receiptLimiter := security.RateLimitMiddleware(30, 1*time.Minute, "Слишком много запросов проверки чеков. Пожалуйста, подождите.")
		api.POST("/receipts/verify", receiptLimiter, receiptHandler.VerifyOrSubmit)

		// Public Auth with brute-force protection
		authGroup := api.Group("/auth")
		authLimiter := security.RateLimitMiddleware(20, 1*time.Minute, "Слишком много попыток авторизации. Пожалуйста, подождите 60 секунд.")
		{
			authGroup.POST("/telegram-widget", authLimiter, authHandler.TelegramWidgetLogin)
			authGroup.GET("/me", auth.AuthMiddleware(cfg.JWTSecret), authHandler.GetMe)
		}

		// Affiliate API
		affGroup := api.Group("/affiliate")
		affGroup.Use(auth.AuthMiddleware(cfg.JWTSecret))
		payoutLimiter := security.RateLimitMiddleware(10, 5*time.Minute, "Слишком много запросов на вывод средств. Пожалуйста, подождите перед повторной отправкой.")
		{
			affGroup.GET("/stats", affiliateHandler.GetStats)
			affGroup.POST("/payout", payoutLimiter, affiliateHandler.SubmitPayout)
		}

		// Promo Codes Public API
		api.POST("/promo/validate", promoHandler.Validate)
		api.POST("/promo/redeem", promoHandler.Redeem)

		// Admin Public Login (Username + Password -> JWT) with strict brute-force rate limiting (5 req/min)
		adminLoginLimiter := security.RateLimitMiddleware(5, 1*time.Minute, "Слишком много попыток входа (максимум 5 в минуту). Пожалуйста, подождите 60 секунд.")
		api.POST("/admin/login", adminLoginLimiter, adminHandler.Login)

		// Admin API (Protected by verified JWT role=admin)
		adminGroup := api.Group("/admin")
		adminGroup.Use(auth.AdminMiddleware(cfg.JWTSecret))
		{
			adminGroup.GET("/stats", adminHandler.GetStats)
			adminGroup.GET("/users", adminHandler.ListUsers)
			adminGroup.POST("/users", adminHandler.CreateUser)
			adminGroup.PUT("/users/:id", adminHandler.UpdateUser)
			adminGroup.DELETE("/users/:id", adminHandler.DeleteUser)
			adminGroup.POST("/users/:id/access", adminHandler.ToggleAccess)
			adminGroup.GET("/payouts", adminHandler.ListPayouts)
			adminGroup.POST("/payouts/:id/approve", adminHandler.ApprovePayout)
			adminGroup.POST("/payouts/:id/reject", adminHandler.RejectPayout)
			adminGroup.GET("/promos", promoHandler.List)
			adminGroup.POST("/promos", promoHandler.Create)
			adminGroup.PUT("/promos/:id", promoHandler.Update)
			adminGroup.DELETE("/promos/:id", promoHandler.Delete)
			adminGroup.POST("/promos/:id/toggle", promoHandler.ToggleStatus)
		}
	}

	// HTTP Server with Graceful Shutdown
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	go func() {
		log.Printf("проект «так называемый SPARK» Backend listening on port %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting successfully.")
}
