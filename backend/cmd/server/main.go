package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/mymmrac/telego"
	"github.com/spark-school/backend/internal/auth"
	"github.com/spark-school/backend/internal/bot"
	"github.com/spark-school/backend/internal/config"
	"github.com/spark-school/backend/internal/database"
	"github.com/spark-school/backend/internal/handlers"
	"github.com/spark-school/backend/internal/models"
	"github.com/spark-school/backend/internal/monitoring"
	"github.com/spark-school/backend/internal/outbox"
	"github.com/spark-school/backend/internal/privacy"
	"github.com/spark-school/backend/internal/readmodels"
	"github.com/spark-school/backend/internal/security"
	"github.com/spark-school/backend/internal/services"
	"gorm.io/gorm"
)

func main() {
	cfg := config.Load()

	// Fail closed: production must never start with missing or known default secrets.
	if err := cfg.Validate(); err != nil {
		log.Fatalf("[FATAL] Refusing to start: %v", err)
	}
	for _, warning := range cfg.Warnings() {
		log.Println("[SECURITY WARNING]", warning)
	}

	isProd := cfg.IsProduction()
	if isProd {
		gin.SetMode(gin.ReleaseMode)
	}

	if cfg.SentryDSN != "" {
		if err := sentry.Init(sentry.ClientOptions{
			Dsn:              cfg.SentryDSN,
			Environment:      cfg.Env,
			TracesSampleRate: 0.05,
			SendDefaultPII:   false,
		}); err != nil {
			log.Printf("[Sentry] initialization failed: %v", err)
		} else {
			defer sentry.Flush(2 * time.Second)
			log.Println("[Sentry] error tracking enabled")
		}
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
	}, botService.DeliverPaymentNotice)
	outboxDispatcher.Start()
	defer outboxDispatcher.Stop()

	metrics := security.NewMetrics()
	monitor := monitoring.New(db, cfg.MonitoringTelegramID, botService.SendNotification, metrics)
	monitor.Start()
	defer monitor.Stop()

	cleaner := privacy.NewCleaner(db, cfg.AnalyticsRetentionDays)
	cleaner.Start()
	defer cleaner.Stop()

	revenueWorker := readmodels.NewRevenueWorker(db)
	revenueWorker.Start()
	defer revenueWorker.Stop()

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
	adminHandler.SetReceiptSender(botService.SendFiscalReceipt)
	adminHandler.SetDigestSender(func() error {
		_, err := botService.SendCuratorDailyDigest()
		return err
	})
	promoHandler := handlers.NewPromoHandler(db)
	linkHandler := handlers.NewCampaignLinkHandler(db)
	pricingHandler := handlers.NewPricingHandler(db)
	studentHandler := handlers.NewStudentHandler(db, cfg)
	diagnosticHandler := handlers.NewDiagnosticHandler(db, cfg)
	statsHandler := handlers.NewStatsHandler(db)
	receiptHandler := handlers.NewReceiptHandler(db, func(receipt *models.PaymentReceipt) error {
		if cfg.AdminTelegramID != 0 {
			return botService.SendAdminReceiptCard(cfg.AdminTelegramID, receipt)
		}
		return nil
	})
	paymentWebhookHandler := handlers.NewPaymentWebhookHandler(
		db, cfg, affiliateService, botService.SendNotification, botService.CreateOneTimeInviteLink,
	)
	yooKassaService := services.NewYooKassaService(cfg)
	yooKassaHandler := handlers.NewYooKassaHandler(
		db, cfg, yooKassaService, affiliateService, botService.SendNotification, botService.CreateOneTimeInviteLink,
	)

	// Gin Router
	router := gin.New()
	// Only these proxies may supply X-Forwarded-For. Without this, a client
	// could spoof ClientIP and bypass per-IP rate limits.
	if err := router.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		log.Fatalf("[FATAL] invalid TRUSTED_PROXIES: %v", err)
	}
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Security Middleware: Headers & Body Payload Limit (2MB) - applied globally to ALL routes
	if cfg.SentryDSN != "" {
		router.Use(sentrygin.New(sentrygin.Options{Repanic: true}))
	}
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

	// Shared PostgreSQL rate limiting: limits apply across all backend instances.
	limit := func(n int, window time.Duration, prefix, message string) gin.HandlerFunc {
		return security.DatabaseRateLimitMiddleware(db, n, window, prefix, message)
	}

	// Webhook Rate Limiter (60 req/min) to prevent DoS floods on payment gateways
	webhookLimiter := limit(60, 1*time.Minute, "webhook", "Превышен лимит запросов к платежному шлюзу. Пожалуйста, подождите.")

	// Top-level direct webhook routes (matches reverse proxy for /api/payments/*)
	router.Any("/api/payments/webhook", security.WebhookMetricsMiddleware(metrics), webhookLimiter, paymentWebhookHandler.HandleAntilopayWebhook)
	router.Any("/payments/webhook", security.WebhookMetricsMiddleware(metrics), webhookLimiter, paymentWebhookHandler.HandleAntilopayWebhook)
	router.Any("/api/payments/tribute", security.WebhookMetricsMiddleware(metrics), webhookLimiter, paymentWebhookHandler.HandleTributeWebhook)
	router.Any("/payments/tribute", security.WebhookMetricsMiddleware(metrics), webhookLimiter, paymentWebhookHandler.HandleTributeWebhook)
	router.Any("/api/payments/yookassa", security.WebhookMetricsMiddleware(metrics), webhookLimiter, yooKassaHandler.HandleWebhook)
	router.Any("/payments/yookassa", security.WebhookMetricsMiddleware(metrics), webhookLimiter, yooKassaHandler.HandleWebhook)

	// Kubernetes/Compose probe endpoints (also reachable directly, not only through Caddy).
	router.GET("/health/live", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now().Format(time.RFC3339)})
	})
	router.GET("/health/ready", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()
		version, err := database.Ready(ctx, db)
		if err != nil {
			metrics.RecordReadinessFailure()
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready", "migration_version": version, "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready", "migration_version": version, "time": time.Now().Format(time.RFC3339)})
	})

	// Public campaign links: /r/<slug> records an anonymous click and redirects.
	// HEAD is mapped too because link checkers and chat previews probe with it.
	// This is a public writing endpoint, so it gets its own generous in-memory
	// per-IP limit (600/min) instead of the shared database limiter used by the
	// API: the redirect must stay cheap and must not touch the DB for throttling.
	campaignLinkLimiter := security.RateLimitMiddleware(600, 1*time.Minute, "Слишком много запросов по коротким ссылкам. Пожалуйста, подождите.")
	router.GET("/r/:slug", campaignLinkLimiter, handlers.RedirectCampaignLink(db, cfg.FrontendURL, cfg.TelegramInviteLink))
	router.HEAD("/r/:slug", campaignLinkLimiter, handlers.RedirectCampaignLink(db, cfg.FrontendURL, cfg.TelegramInviteLink))

	// API v1 Routes with general Rate Limiting (120 req/min per IP)
	api := router.Group("/api/v1")
	api.Use(limit(120, 1*time.Minute, "api", "Превышен лимит запросов к API. Пожалуйста, подождите."))
	{
		// Legacy liveness alias (does not touch the database).
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now().Format(time.RFC3339)})
		})
		api.GET("/health/live", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now().Format(time.RFC3339)})
		})
		api.GET("/health/ready", func(c *gin.Context) {
			ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
			defer cancel()
			version, err := database.Ready(ctx, db)
			if err != nil {
				metrics.RecordReadinessFailure()
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"status":            "not_ready",
					"migration_version": version,
					"error":             err.Error(),
				})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"status":            "ready",
				"migration_version": version,
				"time":              time.Now().Format(time.RFC3339),
			})
		})

		// Public Stats: Live places counter with auto-doubling capacity
		api.GET("/stats/places", statsHandler.GetPlacesStats)

		// Public Pricing: Dynamic tier prices (Accelerator & VIP)
		api.GET("/pricing", pricingHandler.GetPricing)

		// Public Receipts: Anti-fraud receipt submission & deduplication check
		receiptLimiter := limit(30, 1*time.Minute, "receipt", "Слишком много запросов проверки чеков. Пожалуйста, подождите.")
		api.POST("/receipts/verify", receiptLimiter, receiptHandler.VerifyOrSubmit)

		// Public Auth with brute-force protection
		authGroup := api.Group("/auth")
		authLimiter := limit(20, 1*time.Minute, "auth", "Слишком много попыток авторизации. Пожалуйста, подождите 60 секунд.")
		{
			authGroup.POST("/telegram-widget", authLimiter, authHandler.TelegramWidgetLogin)
			authGroup.GET("/me", auth.AuthMiddleware(cfg.JWTSecret), authHandler.GetMe)
		}

		// Affiliate API
		affGroup := api.Group("/affiliate")
		affGroup.Use(auth.AuthMiddleware(cfg.JWTSecret))
		payoutLimiter := limit(10, 5*time.Minute, "payout", "Слишком много запросов на вывод средств. Пожалуйста, подождите перед повторной отправкой.")
		{
			affGroup.GET("/stats", affiliateHandler.GetStats)
			affGroup.POST("/payout", payoutLimiter, affiliateHandler.SubmitPayout)
		}

		// Promo Codes Public API
		api.POST("/promo/validate", promoHandler.Validate)
		api.POST("/promo/redeem", promoHandler.Redeem)

		// Full-funnel analytics ingestion: whitelisted steps, bounded
		// metadata, no PII and an explicit error when persistence fails.
		analyticsLimiter := limit(60, 1*time.Minute, "analytics", "Слишком много событий аналитики.")
		api.POST("/analytics/track", analyticsLimiter, handlers.TrackAnalytics(db))
		api.POST("/diagnostic", analyticsLimiter, diagnosticHandler.Create)
		api.GET("/diagnostic/:token", diagnosticHandler.Get)
		api.POST("/client-errors", limit(20, 1*time.Minute, "client_errors", "Слишком много сообщений об ошибках."), handlers.ClientError)
		api.POST("/analytics/experiments", analyticsLimiter, handlers.AssignExperiments(db))

		// Telegram Bot Webhook endpoint. Authentication, a persistent
		// claim/lease and idempotent business operations protect against
		// duplicate delivery across any number of backend instances.
		updateClaimStore := bot.NewUpdateClaimStore(db, 5*time.Minute)

		api.POST("/bot/webhook", security.WebhookMetricsMiddleware(metrics), func(c *gin.Context) {
			if cfg.TelegramWebhookSecret == "" {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "telegram webhook secret is not configured"})
				return
			}
			secret := c.GetHeader("X-Telegram-Bot-Api-Secret-Token")
			if secret != cfg.TelegramWebhookSecret {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: invalid secret token"})
				return
			}

			var update telego.Update
			if err := c.ShouldBindJSON(&update); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
				return
			}

			ctx := c.Request.Context()
			claimed, err := updateClaimStore.Claim(ctx, update.UpdateID)
			if err != nil {
				log.Printf("[Bot Webhook] claim failed for update_id=%d: %v", update.UpdateID, err)
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "update claim unavailable"})
				return
			}
			if !claimed {
				status, _ := updateClaimStore.Status(ctx, update.UpdateID)
				c.JSON(http.StatusOK, gin.H{"ok": true, "duplicate": true, "status": status})
				return
			}

			if err := botService.HandleUpdate(ctx, &update); err != nil {
				log.Printf("[Bot Webhook] update_id=%d failed: %v", update.UpdateID, err)
				if failErr := updateClaimStore.Fail(ctx, update.UpdateID, err); failErr != nil {
					log.Printf("[Bot Webhook] failed to persist failure for update_id=%d: %v", update.UpdateID, failErr)
				}
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error processing update"})
				return
			}

			if err := updateClaimStore.Complete(ctx, update.UpdateID); err != nil {
				log.Printf("[Bot Webhook] completion failed for update_id=%d: %v", update.UpdateID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist update completion"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		// Payment Webhook endpoints (handles verification pings, callbacks, HMAC validation & access grant)
		api.Any("/payments/webhook", security.WebhookMetricsMiddleware(metrics), paymentWebhookHandler.HandleAntilopayWebhook)
		api.Any("/payments/tribute", security.WebhookMetricsMiddleware(metrics), paymentWebhookHandler.HandleTributeWebhook)

		// YooKassa Payment Endpoints with strict per-IP rate limiting
		yooKassaCreateLimiter := limit(20, 1*time.Minute, "yookassa_create", "Слишком много запросов на создание платежа. Пожалуйста, подождите 60 секунд.")
		yooKassaWebhookLimiter := limit(60, 1*time.Minute, "yookassa_webhook", "Превышен лимит запросов к вебхуку.")
		api.POST("/payments/yookassa/create", yooKassaCreateLimiter, yooKassaHandler.CreateCheckout)
		api.Any("/payments/yookassa", security.WebhookMetricsMiddleware(metrics), yooKassaWebhookLimiter, yooKassaHandler.HandleWebhook)

		// Student API for a future Telegram Mini App. The same JWT is used by
		// the web API, so the Mini App is an API client rather than a separate
		// backend.
		studentGroup := api.Group("/student")
		studentGroup.POST("/auth", authLimiter, studentHandler.TelegramAuth)
		studentAuth := studentGroup.Group("")
		studentAuth.Use(auth.AuthMiddleware(cfg.JWTSecret))
		{
			studentAuth.GET("/me", studentHandler.Me)
			studentAuth.GET("/orders", studentHandler.Orders)
		}

		// Admin Public Login (Username + Password -> JWT) with strict brute-force rate limiting (5 req/min)
		adminLoginLimiter := security.DatabaseRateLimitMiddleware(db, 5, 1*time.Minute, "admin_login", "Слишком много попыток входа (максимум 5 в минуту). Пожалуйста, подождите 60 секунд.")
		api.POST("/admin/login", adminLoginLimiter, adminHandler.Login)

		// Admin API (Protected by verified JWT role=admin)
		adminGroup := api.Group("/admin")
		adminGroup.Use(auth.AdminMiddlewareWithRevocation(cfg.JWTSecret, db))
		adminGroup.Use(handlers.AdminAuditMiddleware(db))
		{
			adminGroup.POST("/logout", adminHandler.Logout)
			adminGroup.GET("/audit-log", adminHandler.ListAuditLog)
			adminGroup.GET("/outbox/dead", adminHandler.ListDeadOutbox)
			adminGroup.POST("/outbox/:id/retry", adminHandler.RetryOutbox)
			adminGroup.GET("/stats", adminHandler.GetStats)
			adminGroup.GET("/analytics/funnel", handlers.GetFunnel(db))
			adminGroup.GET("/analytics/experiments", handlers.GetExperiments(db))
			adminGroup.GET("/users", adminHandler.ListUsers)
			adminGroup.POST("/users", adminHandler.CreateUser)
			adminGroup.PUT("/users/:id", adminHandler.UpdateUser)
			adminGroup.DELETE("/users/:id", adminHandler.DeleteUser)
			adminGroup.POST("/users/:id/erase", adminHandler.EraseUser)
			adminGroup.POST("/users/:id/access", adminHandler.ToggleAccess)
			adminGroup.POST("/users/:id/receipt-issued", adminHandler.ToggleReceiptIssued)
			adminGroup.POST("/users/:id/issue-receipt", adminHandler.IssueReceipt)
			adminGroup.GET("/payouts", adminHandler.ListPayouts)
			adminGroup.POST("/payouts/:id/approve", adminHandler.ApprovePayout)
			adminGroup.POST("/payouts/:id/reject", adminHandler.RejectPayout)
			adminGroup.GET("/promos", promoHandler.List)
			adminGroup.POST("/promos", promoHandler.Create)
			adminGroup.PUT("/promos/:id", promoHandler.Update)
			adminGroup.DELETE("/promos/:id", promoHandler.Delete)
			adminGroup.POST("/promos/:id/toggle", promoHandler.ToggleStatus)
			adminGroup.GET("/links", linkHandler.List)
			adminGroup.POST("/links", linkHandler.Create)
			adminGroup.PUT("/links/:id", linkHandler.Update)
			adminGroup.DELETE("/links/:id", linkHandler.Delete)
			adminGroup.GET("/links/report", linkHandler.Report)
			adminGroup.POST("/digest", adminHandler.TriggerDigest)
			adminGroup.GET("/pricing", pricingHandler.GetPricing)
			adminGroup.PUT("/pricing", pricingHandler.UpdatePricing)
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

	botService.Stop()
	log.Println("Server exiting successfully.")
}

// UpdateDeduplicator provides thread-safe LRU in-memory deduplication with persistent database backing and in-flight claim locking.
type UpdateDeduplicator struct {
	mu       sync.Mutex
	capacity int
	order    []int
	items    map[int]struct{}
	inFlight map[int]time.Time
	db       *gorm.DB
}

func NewUpdateDeduplicator(db *gorm.DB, capacity int) *UpdateDeduplicator {
	return &UpdateDeduplicator{
		capacity: capacity,
		order:    make([]int, 0, capacity),
		items:    make(map[int]struct{}, capacity),
		inFlight: make(map[int]time.Time),
		db:       db,
	}
}

// TryClaim atomically checks if updateID is already processed or currently in-flight.
// Returns claimed=true and a release callback if available.
// If claimed=false, release is nil and the caller must reject processing.
func (d *UpdateDeduplicator) TryClaim(updateID int) (bool, func(bool)) {
	if updateID == 0 {
		return true, func(bool) {}
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	// 1. Check in-memory processed items
	if _, found := d.items[updateID]; found {
		return false, nil
	}

	// 2. Check in-flight (with 5-minute timeout guard)
	if startTime, ok := d.inFlight[updateID]; ok {
		if time.Since(startTime) < 5*time.Minute {
			return false, nil
		}
		delete(d.inFlight, updateID)
	}

	// 3. Check persistent database
	if d.db != nil {
		var record models.ProcessedTelegramUpdate
		if err := d.db.Select("update_id").Where("update_id = ?", updateID).First(&record).Error; err == nil {
			d.addToMemory(updateID)
			return false, nil
		}
	}

	// 4. Atomically claim update in-flight
	d.inFlight[updateID] = time.Now()

	var once sync.Once
	release := func(success bool) {
		once.Do(func() {
			d.mu.Lock()
			defer d.mu.Unlock()
			delete(d.inFlight, updateID)
			if success {
				d.addToMemory(updateID)
				if d.db != nil {
					_ = d.db.Create(&models.ProcessedTelegramUpdate{
						UpdateID:  updateID,
						CreatedAt: time.Now(),
					}).Error
				}
			}
		})
	}

	return true, release
}

func (d *UpdateDeduplicator) IsProcessed(updateID int) bool {
	if updateID == 0 {
		return false
	}
	d.mu.Lock()
	_, found := d.items[updateID]
	_, inFlight := d.inFlight[updateID]
	d.mu.Unlock()
	if found || inFlight {
		return true
	}

	if d.db != nil {
		var record models.ProcessedTelegramUpdate
		if err := d.db.Select("update_id").Where("update_id = ?", updateID).First(&record).Error; err == nil {
			d.mu.Lock()
			d.addToMemory(updateID)
			d.mu.Unlock()
			return true
		}
	}
	return false
}

func (d *UpdateDeduplicator) addToMemory(updateID int) {
	if _, exists := d.items[updateID]; exists {
		return
	}
	if len(d.order) >= d.capacity {
		oldest := d.order[0]
		d.order = d.order[1:]
		delete(d.items, oldest)
	}
	d.order = append(d.order, updateID)
	d.items[updateID] = struct{}{}
}

func (d *UpdateDeduplicator) MarkProcessed(updateID int) {
	if updateID == 0 {
		return
	}
	d.mu.Lock()
	delete(d.inFlight, updateID)
	d.addToMemory(updateID)
	d.mu.Unlock()

	if d.db != nil {
		_ = d.db.Create(&models.ProcessedTelegramUpdate{
			UpdateID:  updateID,
			CreatedAt: time.Now(),
		}).Error
	}
}
