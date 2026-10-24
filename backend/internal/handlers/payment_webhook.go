package handlers

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
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

type PaymentWebhookHandler struct {
	db               *gorm.DB
	cfg              *config.Config
	affiliateService *services.AffiliateService
	notifyFunc       func(telegramID int64, message string) error
	inviteLinkFunc   func(ctx context.Context, channelID int64) (string, error)
	payments         *payments.Service
}

func NewPaymentWebhookHandler(
	db *gorm.DB,
	cfg *config.Config,
	aff *services.AffiliateService,
	notify func(telegramID int64, message string) error,
	inviteLinkGen ...func(ctx context.Context, channelID int64) (string, error),
) *PaymentWebhookHandler {
	var gen func(ctx context.Context, channelID int64) (string, error)
	if len(inviteLinkGen) > 0 {
		gen = inviteLinkGen[0]
	}
	return &PaymentWebhookHandler{
		db:               db,
		cfg:              cfg,
		affiliateService: aff,
		notifyFunc:       notify,
		inviteLinkFunc:   gen,
		payments:         payments.NewService(db, aff),
	}
}

// AntilopayCallbackPayload matches the official Antilopay merchant callback JSON schema.
type AntilopayCallbackPayload struct {
	Type           string  `json:"type"`            // "payment", "withdraw", "refund"
	PaymentID      string  `json:"payment_id"`      // "APAY..."
	OrderID        string  `json:"order_id"`        // Merchant order ID
	CTime          string  `json:"ctime"`           // Creation timestamp
	Amount         float64 `json:"amount"`          // Settled amount
	OriginalAmount float64 `json:"original_amount"` // Expected amount
	Fee            float64 `json:"fee"`             // Payment gateway fee
	Status         string  `json:"status"`          // "SUCCESS", "FAIL", "PENDING"
	Currency       string  `json:"currency"`        // "RUB"
	ProductName    string  `json:"product_name"`
	MerchantExtra  string  `json:"merchant_extra"`
	Description    string  `json:"description"`
	PayMethod      string  `json:"pay_method"` // "CARD_RU", "SBP", etc.
	PayData        string  `json:"pay_data"`
	RecurrentID    string  `json:"recurrent_id"`
	Customer       struct {
		Email    string `json:"email"`
		Phone    string `json:"phone"`
		Fullname string `json:"fullname"`
		IP       string `json:"ip"`
		Address  string `json:"address"`
	} `json:"customer"`
}

// VerifyAntilopaySignature verifies RSA-SHA256 signature according to Antilopay Merchant API specification.
func VerifyAntilopaySignature(publicKeyBase64 string, body []byte, signBase64 string) error {
	cleanKey := strings.TrimSpace(publicKeyBase64)
	cleanKey = strings.TrimPrefix(cleanKey, "-----BEGIN PUBLIC KEY-----")
	cleanKey = strings.TrimSuffix(cleanKey, "-----END PUBLIC KEY-----")
	cleanKey = strings.ReplaceAll(cleanKey, "\n", "")
	cleanKey = strings.ReplaceAll(cleanKey, "\r", "")

	pk, err := base64.StdEncoding.DecodeString(cleanKey)
	if err != nil {
		return fmt.Errorf("decode public key: %w", err)
	}

	publicKey, err := x509.ParsePKIXPublicKey(pk)
	if err != nil {
		return fmt.Errorf("parse public key: %w", err)
	}

	rsaPub, ok := publicKey.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("not an rsa public key")
	}

	signRaw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(signBase64))
	if err != nil {
		return fmt.Errorf("decode signature base64: %w", err)
	}

	hash := sha256.New()
	hash.Write(body)
	d := hash.Sum(nil)

	return rsa.VerifyPKCS1v15(rsaPub, crypto.SHA256, d, signRaw)
}

// HandleAntilopayWebhook processes incoming payment callbacks from Antilopay with idempotency & instant push alerts.
// POST /api/payments/webhook
func (h *PaymentWebhookHandler) HandleAntilopayWebhook(c *gin.Context) {
	// 1. Read raw body
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("[ANTILOPAY] Error reading request body: %v", err)
		c.JSON(http.StatusOK, gin.H{"status": "ok", "error": "cannot read body"})
		return
	}

	rawBodyStr := string(bodyBytes)
	log.Printf("[ANTILOPAY] Callback received (%d bytes)", len(bodyBytes))

	// If empty body (e.g. initial health check ping), respond with 200 OK immediately
	if len(bodyBytes) == 0 {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "empty ping acknowledged"})
		return
	}

	// 2. Signature is required whenever the webhook is reachable.
	signHeader := c.GetHeader("X-Apay-Callback")
	isTestMode := gin.Mode() == gin.TestMode || os.Getenv("ENV") == "test"
	if h.cfg == nil || strings.TrimSpace(h.cfg.AntilopayPublicKey) == "" {
		if !isTestMode {
			log.Printf("[ANTILOPAY-SECURITY] Public key is not configured; rejecting unsigned callback")
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "payment webhook is not configured"})
			return
		}
	} else {
		if signHeader == "" {
			log.Printf("[ANTILOPAY-SECURITY] Missing required signature header (X-Apay-Callback)")
			c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "missing signature header"})
			return
		}
		if err := VerifyAntilopaySignature(h.cfg.AntilopayPublicKey, bodyBytes, signHeader); err != nil {
			log.Printf("[ANTILOPAY-SECURITY] Signature verification failed: %v", err)
			c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "invalid signature"})
			return
		}
		log.Printf("[ANTILOPAY-SECURITY] Signature verified successfully with RSA-SHA256")
	}

	// 3. Parse JSON payload
	var payload AntilopayCallbackPayload
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		log.Printf("[ANTILOPAY] JSON parse error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "unmarshal error"})
		return
	}

	// Only process payment callbacks
	if payload.Type != "" && payload.Type != "payment" {
		log.Printf("[ANTILOPAY] Non-payment callback type: %s", payload.Type)
		c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "unsupported type"})
		return
	}

	paymentID := strings.TrimSpace(payload.PaymentID)
	if paymentID == "" {
		// Deterministic fallback from body hash to guarantee replay-attack prevention
		paymentID = "APAY_HASH_" + ComputeHash(bodyBytes)[:20]
	}

	// 4. Extract metadata (telegram_id, tier, promo_code) from order_id or merchant_extra.
	var telegramID int64
	tier := "accelerator"
	promoCode := ""
	username := ""
	orderID := strings.TrimSpace(payload.OrderID)

	orderParts := strings.Split(payload.OrderID, "_")
	for i, part := range orderParts {
		if id, err := strconv.ParseInt(part, 10, 64); err == nil && id > 10000 {
			telegramID = id
		}
		lower := strings.ToLower(part)
		if lower == "vip" {
			tier = "vip"
		} else if lower == "basic" {
			tier = "basic"
		} else if lower == "accelerator" {
			tier = "accelerator"
		} else if lower == "audit" && i+1 < len(orderParts) && strings.EqualFold(orderParts[i+1], "essay") {
			tier = "audit_essay"
		}
		if (lower == "promo" || lower == "code") && i+1 < len(orderParts) {
			promoCode = strings.ToUpper(orderParts[i+1])
		}
	}

	if payload.MerchantExtra != "" {
		extra := payload.MerchantExtra
		if strings.Contains(extra, "telegram_id=") {
			for _, param := range strings.Split(extra, "&") {
				kv := strings.SplitN(param, "=", 2)
				if len(kv) != 2 {
					continue
				}
				k := strings.ToLower(strings.TrimSpace(kv[0]))
				v := strings.TrimSpace(kv[1])
				switch k {
				case "telegram_id", "tg_id", "user_id":
					if id, err := strconv.ParseInt(v, 10, 64); err == nil && id > 0 {
						telegramID = id
					}
				case "tier":
					if normalized := normalizePaidTier(v); normalized != "" {
						tier = normalized
					}
				case "promo", "promocode", "promo_code":
					promoCode = strings.ToUpper(v)
				case "username":
					username = strings.TrimPrefix(v, "@")
				}
			}
		}
	}

	amount := roundAmount(payload.Amount)
	currency := strings.ToUpper(payload.Currency)
	if currency == "" {
		currency = "RUB"
	}

	// Amount-based fallback is only used for legacy orders without metadata.
	if telegramID == 0 || (promoCode == "" && payload.MerchantExtra == "") {
		if amount >= 13000 {
			tier = "vip"
		} else if amount < 5000 {
			tier = "basic"
		}
	}

	// If this is a server-side checkout order, it is authoritative.
	var order models.PaymentOrder
	hasOrder := false
	if orderID != "" && h.db != nil {
		if err := h.db.Where("order_id = ?", orderID).First(&order).Error; err == nil {
			hasOrder = true
			if order.Provider != "" && order.Provider != "antilopay" {
				c.JSON(http.StatusConflict, gin.H{"status": "error", "message": "order provider mismatch"})
				return
			}
			if order.Currency != "" && currency != strings.ToUpper(order.Currency) {
				c.JSON(http.StatusConflict, gin.H{"status": "error", "message": "currency mismatch"})
				return
			}
			if diff := amount - order.Amount; diff > 0.01 || diff < -0.01 {
				c.JSON(http.StatusConflict, gin.H{"status": "error", "message": "amount mismatch"})
				return
			}
			telegramID = order.TelegramID
			tier = order.Tier
			promoCode = firstNonEmptyString(promoCode, order.PromoCode)
			username = firstNonEmptyString(username, order.Username)
		}
	}

	// 5. Anti-underpayment protection by product.
	minAmount := minimumForTier(tier)
	if strings.ToUpper(payload.Status) == "SUCCESS" && amount < minAmount {
		log.Printf("[SECURITY-ALERT] Antilopay payment %s underpayment detected: %.2f %s. Minimum for %s: %.2f.",
			paymentID, amount, currency, tier, minAmount)

		if h.db != nil {
			transaction := models.PaymentTransaction{
				PaymentID:      paymentID,
				OrderID:        orderID,
				TelegramID:     telegramID,
				Username:       username,
				Tier:           tier,
				Amount:         amount,
				OriginalAmount: payload.OriginalAmount,
				Fee:            payload.Fee,
				Status:         "FLAGGED_UNDERPAID",
				Currency:       currency,
				PayMethod:      payload.PayMethod,
				PromoCode:      promoCode,
				CustomerEmail:  payload.Customer.Email,
				CustomerPhone:  payload.Customer.Phone,
				RawCallback:    rawBodyStr,
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			}
			_ = h.db.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "payment_id"}}, DoNothing: true}).Create(&transaction).Error
		}
		h.alertAdmin(fmt.Sprintf(
			"⚠️ <b>Antilopay: недоплата</b>\n\nСтудент: ID %d (@%s)\nСумма: %.2f %s\nМинимум для %s: %.2f\nПлатёж: <code>%s</code>\n\nДоступ НЕ выдан.",
			telegramID, html.EscapeString(username), amount, currency, html.EscapeString(tier), minAmount, html.EscapeString(paymentID),
		))
		c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "payment_flagged_underpaid"})
		return
	}

	// 6. Only settled payments change state.
	if strings.ToUpper(payload.Status) != "SUCCESS" {
		if h.db != nil {
			pending := models.PaymentTransaction{
				PaymentID:      paymentID,
				OrderID:        orderID,
				TelegramID:     telegramID,
				Username:       username,
				Tier:           tier,
				Amount:         amount,
				OriginalAmount: payload.OriginalAmount,
				Fee:            payload.Fee,
				Status:         strings.ToUpper(payload.Status),
				Currency:       currency,
				PayMethod:      payload.PayMethod,
				PromoCode:      promoCode,
				RawCallback:    rawBodyStr,
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			}
			_ = h.db.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "payment_id"}}, DoNothing: true}).Create(&pending).Error
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "non-success status recorded"})
		return
	}

	visitorID := ""
	attributionToken := ""
	if hasOrder {
		visitorID = order.VisitorID
		attributionToken = order.AttributionToken
	}

	result, err := h.payments.Confirm(payments.ConfirmInput{
		Source:           "antilopay",
		PaymentID:        paymentID,
		OrderID:          orderID,
		TelegramID:       telegramID,
		Username:         username,
		Tier:             tier,
		Amount:           amount,
		Currency:         currency,
		PayMethod:        payload.PayMethod,
		PromoCode:        promoCode,
		ProductName:      payload.ProductName,
		RawCallback:      rawBodyStr,
		VisitorID:        visitorID,
		AttributionToken: attributionToken,
		CreatedAt:        time.Now(),
	})
	if err != nil {
		log.Printf("[ANTILOPAY-ERROR] Confirmation failed for %s: %v", paymentID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "payment confirmation failed"})
		return
	}
	if result.AlreadyProcessed {
		log.Printf("[ANTILOPAY-IDEMPOTENCY] Payment %s already processed.", paymentID)
		c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "already_processed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":     "ok",
		"payment_id": paymentID,
		"message":    "callback processed successfully",
	})
}
