package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spark-school/backend/internal/config"
	"github.com/spark-school/backend/internal/models"
	"github.com/spark-school/backend/internal/payments"
	"github.com/spark-school/backend/internal/services"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type YooKassaHandler struct {
	db               *gorm.DB
	cfg              *config.Config
	yooKassaService  *services.YooKassaService
	affiliateService *services.AffiliateService
	notifyFunc       func(telegramID int64, message string) error
	inviteLinkFunc   func(ctx context.Context, channelID int64) (string, error)
	payments         *payments.Service
}

func NewYooKassaHandler(
	db *gorm.DB,
	cfg *config.Config,
	ykService *services.YooKassaService,
	affService *services.AffiliateService,
	notify func(telegramID int64, message string) error,
	inviteLinkGen ...func(ctx context.Context, channelID int64) (string, error),
) *YooKassaHandler {
	var gen func(ctx context.Context, channelID int64) (string, error)
	if len(inviteLinkGen) > 0 {
		gen = inviteLinkGen[0]
	}
	return &YooKassaHandler{
		db:               db,
		cfg:              cfg,
		yooKassaService:  ykService,
		affiliateService: affService,
		notifyFunc:       notify,
		inviteLinkFunc:   gen,
		payments:         payments.NewService(db, affService),
	}
}

type CreateYooKassaCheckoutRequest struct {
	Tier             string `json:"tier"` // "basic", "accelerator", "vip" or "audit_essay"
	TelegramUsername string `json:"telegram_username"`
	TelegramID       int64  `json:"telegram_id"`
	PromoCode        string `json:"promo_code"`
	Email            string `json:"email"`
	VisitorID        string `json:"visitor_id"`
	AttributionToken string `json:"attribution_token"`
}

// CreateCheckout creates a payment session via official YooKassa API and returns the redirect URL.
// POST /api/payments/yookassa/create
func (h *YooKassaHandler) CreateCheckout(c *gin.Context) {
	var req CreateYooKassaCheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат запроса"})
		return
	}

	tier := normalizePaidTier(req.Tier)
	if tier == "" {
		tier = "accelerator"
	}

	// Security: Sanitize Telegram ID (user IDs are strictly positive)
	telegramID := req.TelegramID
	if telegramID < 0 || telegramID > 1000000000000 {
		telegramID = 0
	}
	// A payment must always have an identifiable recipient; otherwise access
	// cannot be granted automatically after the provider confirms it.
	if telegramID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "telegram_id is required for checkout"})
		return
	}

	// Security: Sanitize Username (strip @, restrict length, allow only valid Telegram username chars)
	rawUsername := strings.TrimPrefix(strings.TrimSpace(req.TelegramUsername), "@")
	if len(rawUsername) > 64 {
		rawUsername = rawUsername[:64]
	}
	var cleanUserBuilder strings.Builder
	for _, r := range rawUsername {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			cleanUserBuilder.WriteRune(r)
		}
	}
	cleanUsername := cleanUserBuilder.String()

	// Security: Sanitize Promo code
	cleanPromo := strings.ToUpper(strings.TrimSpace(req.PromoCode))
	if len(cleanPromo) > 32 {
		cleanPromo = cleanPromo[:32]
	}

	pricing := models.GetCoursePricing(h.db)
	basePrice := pricing.AcceleratorPrice
	switch tier {
	case "basic":
		basePrice = pricing.BasicPrice
	case "vip":
		basePrice = pricing.VIPPrice
	case "audit_essay":
		basePrice = 1900
	}

	finalPrice := basePrice
	discountPct := 0

	// Check promo code if supplied
	if cleanPromo != "" && h.db != nil {
		var promo models.PromoCode
		if err := h.db.Where("UPPER(code) = ? AND is_active = true", cleanPromo).First(&promo).Error; err == nil {
			discountPct = promo.DiscountPercent
			if discountPct <= 0 {
				discountPct = 5
			}
			discountAmount := (basePrice * float64(discountPct)) / 100.0
			finalPrice = basePrice - discountAmount
		} else {
			cleanPromo = "" // Reset invalid promo
		}
	}

	tierNameRU := "Акселератор"
	switch tier {
	case "vip":
		tierNameRU = "VIP"
	case "basic":
		tierNameRU = "Базовый"
	case "audit_essay":
		tierNameRU = "Аудит эссе"
	}

	domain := "so-called-spark.ru"
	if h.cfg != nil && h.cfg.Domain != "" {
		domain = h.cfg.Domain
	}

	description := fmt.Sprintf("Оплата участия в проекте «так называемый SPARK» (Тариф %s)", tierNameRU)
	returnURL := fmt.Sprintf("https://%s/#payment-success", domain)

	orderID := fmt.Sprintf("YK_%d_%d", telegramID, time.Now().UnixNano())
	visitorID := sanitizeToken(req.VisitorID, 64)
	attributionToken := sanitizeToken(req.AttributionToken, 64)

	if h.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "checkout is temporarily unavailable"})
		return
	}
	order := models.PaymentOrder{
		OrderID:          orderID,
		TelegramID:       telegramID,
		Username:         cleanUsername,
		Tier:             tier,
		Amount:           roundAmount(finalPrice),
		Currency:         "RUB",
		PromoCode:        cleanPromo,
		Provider:         "yookassa",
		Status:           "pending",
		VisitorID:        visitorID,
		AttributionToken: attributionToken,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	if err := h.db.Create(&order).Error; err != nil {
		log.Printf("[YOOKASSA-ERROR] Failed to persist checkout order: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "checkout could not be created"})
		return
	}
	_ = h.db.Create(&models.AnalyticsEvent{
		VisitorID:        visitorID,
		AttributionToken: attributionToken,
		Step:             "order_created",
		Source:           "web",
		Tier:             tier,
		PromoCode:        cleanPromo,
		Metadata:         fmt.Sprintf(`{"order_id":%q,"telegram_id":%d,"amount":%f}`, orderID, telegramID, order.Amount),
		CreatedAt:        time.Now(),
	}).Error

	metadata := map[string]string{
		"tier":              tier,
		"username":          cleanUsername,
		"promo_code":        cleanPromo,
		"telegram_id":       strconv.FormatInt(telegramID, 10),
		"order_id":          orderID,
		"visitor_id":        visitorID,
		"attribution_token": attributionToken,
	}

	ykReq := services.YooKassaPaymentRequest{
		Amount: services.YooKassaAmount{
			Value:    fmt.Sprintf("%.2f", finalPrice),
			Currency: "RUB",
		},
		Capture: true,
		Confirmation: services.YooKassaConfirmation{
			Type:      "redirect",
			ReturnURL: returnURL,
		},
		Description: description,
		Metadata:    metadata,
	}

	if h.yooKassaService == nil || h.cfg == nil || strings.TrimSpace(h.cfg.YooKassaSecretKey) == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Платёжный сервис ЮKassa настраивается. Пожалуйста, оплатите через Telegram-бота или Tribute.",
		})
		return
	}

	paymentResp, err := h.yooKassaService.CreatePayment(c.Request.Context(), ykReq)
	if err != nil {
		log.Printf("[YOOKASSA-ERROR] CreatePayment failed: %v", err)
		_ = h.db.Model(&models.PaymentOrder{}).Where("order_id = ?", orderID).Updates(map[string]interface{}{
			"status":     "failed",
			"updated_at": time.Now(),
		}).Error
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Платёжный сервис ЮKassa временно недоступен. Пожалуйста, оплатите через Telegram-бота или Tribute.",
		})
		return
	}
	_ = h.db.Model(&models.PaymentOrder{}).Where("order_id = ?", orderID).Updates(map[string]interface{}{
		"provider_payment_id": paymentResp.ID,
		"updated_at":          time.Now(),
	}).Error

	c.JSON(http.StatusOK, gin.H{
		"payment_id":       paymentResp.ID,
		"order_id":         orderID,
		"status":           paymentResp.Status,
		"confirmation_url": paymentResp.Confirmation.ConfirmationURL,
		"amount":           roundAmount(finalPrice),
		"tier":             tier,
	})
}

// YooKassaWebhookPayload represents YooKassa notification webhook schema.
type YooKassaWebhookPayload struct {
	Type   string `json:"type"`  // "notification"
	Event  string `json:"event"` // "payment.succeeded", "payment.canceled", etc.
	Object struct {
		ID            string                  `json:"id"`
		Status        string                  `json:"status"` // "succeeded"
		Paid          bool                    `json:"paid"`
		Amount        services.YooKassaAmount `json:"amount"`
		Description   string                  `json:"description"`
		Metadata      map[string]string       `json:"metadata"`
		PaymentMethod struct {
			Type  string `json:"type"`
			Title string `json:"title"`
		} `json:"payment_method"`
	} `json:"object"`
}

// HandleWebhook processes payment.succeeded notifications from YooKassa with end-to-end API verification.
// POST /api/payments/yookassa and /payments/yookassa
func (h *YooKassaHandler) HandleWebhook(c *gin.Context) {
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("[YOOKASSA-SECURITY] Error reading webhook body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot read body"})
		return
	}

	if len(bodyBytes) == 0 {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "empty ping acknowledged"})
		return
	}

	var payload YooKassaWebhookPayload
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		log.Printf("[YOOKASSA] JSON parse error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}

	// We only process successful settlement events
	if payload.Event != "payment.succeeded" {
		log.Printf("[YOOKASSA] Ignored non-success event: %s", payload.Event)
		c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "event ignored"})
		return
	}

	paymentID := strings.TrimSpace(payload.Object.ID)
	if paymentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing payment id"})
		return
	}

	// Security: Gold Standard Server-to-Server Verification
	// Verify payment directly against official YooKassa API to eliminate spoofing risks.
	// Fail-closed: If secret key is missing in production, reject immediately.
	isTestMode := gin.Mode() == gin.TestMode || os.Getenv("ENV") == "test"
	if h.cfg == nil || strings.TrimSpace(h.cfg.YooKassaSecretKey) == "" {
		if !isTestMode {
			log.Printf("[CRITICAL-SECURITY] YooKassa webhook received but YOOKASSA_SECRET_KEY is not configured! Rejecting to prevent unverified access.")
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "payment gateway verification not configured"})
			return
		}
	} else if h.yooKassaService != nil {
		verifiedPayment, err := h.yooKassaService.GetPayment(c.Request.Context(), paymentID)
		if err != nil {
			log.Printf("[YOOKASSA-SECURITY] Failed to verify payment with YooKassa API: %v", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "failed to verify payment with yookassa api"})
			return
		}

		if verifiedPayment.Status != "succeeded" || !verifiedPayment.Paid {
			log.Printf("[YOOKASSA-SECURITY] Payment %s status mismatch (status=%s, paid=%v)",
				paymentID, verifiedPayment.Status, verifiedPayment.Paid)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "payment not confirmed by yookassa"})
			return
		}

		// ZERO-TRUST DEFENSE: Overwrite payload fields with authoritative data directly from YooKassa server
		payload.Object.Status = verifiedPayment.Status
		payload.Object.Paid = verifiedPayment.Paid
		if verifiedPayment.Amount.Value != "" {
			payload.Object.Amount = verifiedPayment.Amount
		}
		if len(verifiedPayment.Metadata) > 0 {
			payload.Object.Metadata = verifiedPayment.Metadata
		}
		if verifiedPayment.Description != "" {
			payload.Object.Description = verifiedPayment.Description
		}
		log.Printf("[YOOKASSA-SECURITY] Payment %s verified directly with YooKassa API: 100%% authentic.", paymentID)
	}

	// Extract amount
	amount, _ := strconv.ParseFloat(payload.Object.Amount.Value, 64)
	amount = roundAmount(amount)
	currency := strings.ToUpper(payload.Object.Amount.Currency)
	if currency == "" {
		currency = "RUB"
	}

	// Extract metadata.
	tier := "accelerator"
	promoCode := ""
	username := ""
	visitorID := ""
	attributionToken := ""
	orderID := ""
	var telegramID int64

	if payload.Object.Metadata != nil {
		m := payload.Object.Metadata
		if t, ok := m["tier"]; ok {
			if normalized := normalizePaidTier(t); normalized != "" {
				tier = normalized
			}
		}
		if p, ok := m["promo_code"]; ok {
			promoCode = strings.ToUpper(strings.TrimSpace(p))
		}
		if u, ok := m["username"]; ok {
			username = strings.TrimPrefix(u, "@")
		}
		if tgStr, ok := m["telegram_id"]; ok {
			if id, err := strconv.ParseInt(tgStr, 10, 64); err == nil {
				telegramID = id
			}
		}
		if v, ok := m["visitor_id"]; ok {
			visitorID = v
		}
		if a, ok := m["attribution_token"]; ok {
			attributionToken = a
		}
		if o, ok := m["order_id"]; ok {
			orderID = strings.TrimSpace(o)
		}
	}

	// If checkout was created through the web API, the server-side order is
	// authoritative: provider amount, currency and product must match it.
	if orderID != "" && h.db != nil {
		var order models.PaymentOrder
		if err := h.db.Where("order_id = ?", orderID).First(&order).Error; err != nil {
			log.Printf("[YOOKASSA-SECURITY] Unknown order %s for payment %s", orderID, paymentID)
			c.JSON(http.StatusBadRequest, gin.H{"error": "unknown order"})
			return
		}
		if order.Status != "pending" && order.Status != "paid" {
			c.JSON(http.StatusConflict, gin.H{"error": "order is not payable"})
			return
		}
		if order.Provider != "" && order.Provider != "yookassa" {
			c.JSON(http.StatusConflict, gin.H{"error": "order provider mismatch"})
			return
		}
		if order.Currency != "" && currency != strings.ToUpper(order.Currency) {
			c.JSON(http.StatusConflict, gin.H{"error": "currency mismatch"})
			return
		}
		if diff := amount - order.Amount; diff > 0.01 || diff < -0.01 {
			log.Printf("[YOOKASSA-SECURITY] Amount mismatch for order %s: got %.2f, expected %.2f", orderID, amount, order.Amount)
			c.JSON(http.StatusConflict, gin.H{"error": "amount mismatch"})
			return
		}
		telegramID = order.TelegramID
		tier = order.Tier
		promoCode = firstNonEmptyString(promoCode, order.PromoCode)
		visitorID = firstNonEmptyString(visitorID, order.VisitorID)
		attributionToken = firstNonEmptyString(attributionToken, order.AttributionToken)
		username = firstNonEmptyString(username, order.Username)
	}

	// Anti-underpayment protection by product.
	minAmount := minimumForTier(tier)
	if amount < minAmount {
		log.Printf("[SECURITY-ALERT] YooKassa payment %s underpayment detected: %.2f %s. Minimum for %s: %.2f.",
			paymentID, amount, currency, tier, minAmount)

		if h.db != nil {
			flagged := models.PaymentTransaction{
				PaymentID:      paymentID,
				OrderID:        orderID,
				TelegramID:     telegramID,
				Username:       username,
				Tier:           tier,
				Amount:         amount,
				OriginalAmount: amount,
				Status:         "FLAGGED_UNDERPAID",
				Currency:       currency,
				PayMethod:      "YOOKASSA",
				PromoCode:      promoCode,
				RawCallback:    string(bodyBytes),
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			}
			_ = h.db.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "payment_id"}}, DoNothing: true}).Create(&flagged).Error
		}
		h.alertAdmin(fmt.Sprintf("⚠️ <b>ЮKassa: недоплата</b>\n\nПлатёж <code>%s</code>\nСумма: %.2f %s\nМинимум для %s: %.2f\n\nДоступ НЕ выдан.", html.EscapeString(paymentID), amount, currency, html.EscapeString(tier), minAmount))
		c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "payment_flagged_underpaid"})
		return
	}

	payMethod := "YOOKASSA"
	if payload.Object.PaymentMethod.Type != "" {
		payMethod = "YOOKASSA_" + strings.ToUpper(payload.Object.PaymentMethod.Type)
	}

	result, err := h.payments.Confirm(payments.ConfirmInput{
		Source:           "yookassa",
		PaymentID:        paymentID,
		OrderID:          orderID,
		TelegramID:       telegramID,
		Username:         username,
		Tier:             tier,
		Amount:           amount,
		Currency:         currency,
		PayMethod:        payMethod,
		PromoCode:        promoCode,
		ProductName:      payload.Object.Description,
		RawCallback:      string(bodyBytes),
		VisitorID:        visitorID,
		AttributionToken: attributionToken,
		CreatedAt:        time.Now(),
	})
	if err != nil {
		log.Printf("[YOOKASSA-ERROR] Confirmation failed for %s: %v", paymentID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "payment confirmation failed"})
		return
	}
	if result.AlreadyProcessed {
		log.Printf("[YOOKASSA-IDEMPOTENCY] Payment %s already processed.", paymentID)
		c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "already_processed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "payment processed successfully"})
}
