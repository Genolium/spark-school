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
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spark-school/backend/internal/config"
	"github.com/spark-school/backend/internal/models"
	"github.com/spark-school/backend/internal/services"
	"gorm.io/gorm"
)

type PaymentWebhookHandler struct {
	db               *gorm.DB
	cfg              *config.Config
	affiliateService *services.AffiliateService
	notifyFunc       func(telegramID int64, message string) error
	inviteLinkFunc   func(ctx context.Context, channelID int64) (string, error)
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
	log.Printf("[ANTILOPAY] Callback received (%d bytes): %s", len(bodyBytes), rawBodyStr)

	// If empty body (e.g. initial health check ping), respond with 200 OK immediately
	if len(bodyBytes) == 0 {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "empty ping acknowledged"})
		return
	}

	// 2. Strict Signature verification if public key is configured
	signHeader := c.GetHeader("X-Apay-Callback")
	if h.cfg != nil && h.cfg.AntilopayPublicKey != "" {
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

	// 4. Idempotency Check: prevent duplicate processing & double rewards
	if h.db != nil {
		var existing models.PaymentTransaction
		if err := h.db.Where("payment_id = ?", paymentID).First(&existing).Error; err == nil {
			log.Printf("[ANTILOPAY-IDEMPOTENCY] Payment %s already processed. Skipping duplicate.", paymentID)
			c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "already_processed"})
			return
		}
	}

	// 5. Extract metadata (telegram_id, tier, promo_code) from order_id or merchant_extra
	// Format conventions:
	// OrderID: "spark_{telegramId}_{tier}_{promo}_{timestamp}" or "pay_{tier}_{telegramId}"
	// MerchantExtra: "telegram_id=12345&tier=accelerator&promo=SPARK26"
	var telegramID int64
	tier := "accelerator"
	promoCode := ""
	username := ""

	// Check order_id parts
	orderParts := strings.Split(payload.OrderID, "_")
	for i, part := range orderParts {
		if id, err := strconv.ParseInt(part, 10, 64); err == nil && id > 10000 {
			telegramID = id
		}
		lower := strings.ToLower(part)
		if lower == "vip" {
			tier = "vip"
		} else if lower == "accelerator" {
			tier = "accelerator"
		}
		if (lower == "promo" || lower == "code") && i+1 < len(orderParts) {
			promoCode = strings.ToUpper(orderParts[i+1])
		}
	}

	// Check merchant_extra if present (query-string or JSON format)
	if payload.MerchantExtra != "" {
		extra := payload.MerchantExtra
		if strings.Contains(extra, "telegram_id=") {
			for _, param := range strings.Split(extra, "&") {
				kv := strings.SplitN(param, "=", 2)
				if len(kv) == 2 {
					k := strings.ToLower(strings.TrimSpace(kv[0]))
					v := strings.TrimSpace(kv[1])
					switch k {
					case "telegram_id", "tg_id", "user_id":
						if id, err := strconv.ParseInt(v, 10, 64); err == nil && id > 0 {
							telegramID = id
						}
					case "tier":
						if strings.ToLower(v) == "basic" {
							tier = "basic"
						} else if strings.ToLower(v) == "vip" {
							tier = "vip"
						}
					case "promo", "promocode", "promo_code":
						promoCode = strings.ToUpper(v)
					case "username":
						username = strings.TrimPrefix(v, "@")
					}
				}
			}
		}
	}

	// Determine tier based on amount if not explicit:
	// VIP without promo is 14900, with 5% discount is 14155
	// Accelerator is 6900 / 6555
	// Basic is 3990 / 3790
	if payload.Amount >= 13000 {
		tier = "vip"
	} else if payload.Amount < 5000 {
		tier = "basic"
	}

	// 6. Anti-Underpayment & Tampering Protection
	if strings.ToUpper(payload.Status) == "SUCCESS" && payload.Amount < MinCoursePaymentThreshold {
		log.Printf("[SECURITY-ALERT] Antilopay payment %s underpayment detected: %.2f %s. Minimum required: %.2f.",
			paymentID, payload.Amount, payload.Currency, MinCoursePaymentThreshold)

		transaction := models.PaymentTransaction{
			PaymentID:      paymentID,
			OrderID:        payload.OrderID,
			TelegramID:     telegramID,
			Username:       username,
			Tier:           tier,
			Amount:         payload.Amount,
			OriginalAmount: payload.OriginalAmount,
			Fee:            payload.Fee,
			Status:         "FLAGGED_UNDERPAID",
			Currency:       payload.Currency,
			PayMethod:      payload.PayMethod,
			PromoCode:      promoCode,
			CustomerEmail:  payload.Customer.Email,
			CustomerPhone:  payload.Customer.Phone,
			RawCallback:    rawBodyStr,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		if h.db != nil {
			_ = h.db.Create(&transaction)
		}

		adminTG := int64(123456789)
		if h.cfg != nil && h.cfg.AdminTelegramID != 0 {
			adminTG = h.cfg.AdminTelegramID
		}
		if h.notifyFunc != nil && adminTG != 0 {
			alertMsg := fmt.Sprintf(
				"⚠️ <b>ВНИМАНИЕ: ПОДОЗРИТЕЛЬНАЯ ОПЛАТА ANTILOPAY!</b> ⚠️\n\n"+
					"👤 <b>Студент:</b> ID %d (@%s)\n"+
					"💰 <b>Сумма:</b> %.2f %s <i>(МЕНЬШЕ МИНИМУМА 6 000 ₽)</i>\n"+
					"🆔 <b>ID платежа:</b> <code>%s</code>\n\n"+
					"⛔️ <b>Доступ НЕ выдан.</b> Проверьте транзакцию вручную!",
				telegramID, username, payload.Amount, payload.Currency, paymentID,
			)
			_ = h.notifyFunc(adminTG, alertMsg)
		}

		c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "payment_flagged_underpaid"})
		return
	}

	// Record PaymentTransaction in DB
	transaction := models.PaymentTransaction{
		PaymentID:      paymentID,
		OrderID:        payload.OrderID,
		TelegramID:     telegramID,
		Username:       username,
		Tier:           tier,
		Amount:         payload.Amount,
		OriginalAmount: payload.OriginalAmount,
		Fee:            payload.Fee,
		Status:         payload.Status,
		Currency:       payload.Currency,
		PayMethod:      payload.PayMethod,
		PromoCode:      promoCode,
		CustomerEmail:  payload.Customer.Email,
		CustomerPhone:  payload.Customer.Phone,
		RawCallback:    rawBodyStr,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if h.db != nil {
		_ = h.db.Create(&transaction)
	}

	// 7. If status is SUCCESS: Grant Access & Process Notifications
	if strings.ToUpper(payload.Status) == "SUCCESS" {
		tierNameRU := "Акселератор"
		if tier == "vip" {
			tierNameRU = "VIP-сопровождение"
		}

		// Update or Create user in database
		if h.db != nil && telegramID > 0 {
			var user models.User
			now := time.Now()
			err := h.db.Where("telegram_id = ?", telegramID).First(&user).Error
			if err == nil {
				user.HasAccess = true
				user.AccessGrantedAt = &now
				user.Tier = tier
				user.PaidAmount = payload.Amount
				if promoCode != "" {
					user.PromoCode = promoCode
				}
				h.db.Save(&user)
				if user.Username != "" && username == "" {
					username = user.Username
				}
			} else {
				// Create new student
				newUser := models.User{
					TelegramID:      telegramID,
					Username:        username,
					FirstName:       payload.Customer.Fullname,
					Role:            "student",
					HasAccess:       true,
					AccessGrantedAt: &now,
					Tier:            tier,
					PaidAmount:      payload.Amount,
					PromoCode:       promoCode,
				}
				if newUser.FirstName == "" {
					newUser.FirstName = "Студент"
				}
				h.db.Create(&newUser)
			}

			// Atomic promo attribution
			if promoCode != "" {
				_, _ = RedeemPromoCode(h.db, promoCode)
			}
		}

		// Generate dynamic single-use invite link for student
		inviteLink := ""
		if h.cfg != nil {
			inviteLink = h.cfg.TelegramInviteLink
		}
		if h.inviteLinkFunc != nil && h.cfg != nil && h.cfg.TelegramChannelID != 0 {
			if dynLink, err := h.inviteLinkFunc(context.Background(), h.cfg.TelegramChannelID); err == nil && dynLink != "" {
				inviteLink = dynLink
			}
		}

		// Deliver invite link to student via bot
		if h.notifyFunc != nil && telegramID > 0 {
			studentMsg := fmt.Sprintf(
				"🎉 <b>Оплата подтверждена!</b>\n\n"+
					"Добро пожаловать в проект «так называемый SPARK» 2027!\n\n"+
					"📦 <b>Твой тариф:</b> %s\n"+
					"💳 <b>Оплачено:</b> %.2f ₽\n\n"+
					"🔒 <b>Твоя персональная ссылка в закрытый канал и комьюнити:</b>\n%s\n\n"+
					"<i>Ссылка одноразовая и действует 24 часа. Сохрани её.</i>",
				tierNameRU, payload.Amount, inviteLink,
			)
			_ = h.notifyFunc(telegramID, studentMsg)
		}

		// Instant Push Notification to Ilya (Admin) in Telegram
		adminTG := int64(123456789)
		if h.cfg != nil && h.cfg.AdminTelegramID != 0 {
			adminTG = h.cfg.AdminTelegramID
		}

		if h.notifyFunc != nil && adminTG != 0 {
			studentRef := fmt.Sprintf("ID: %d", telegramID)
			if username != "" {
				studentRef = fmt.Sprintf("@%s (%s)", username, studentRef)
			}

			promoBadge := "— (без промокода)"
			if promoCode != "" {
				promoBadge = fmt.Sprintf("🏷 %s (-5%%)", promoCode)
			}

			customerContact := ""
			if payload.Customer.Phone != "" {
				customerContact += "\n📱 Телефон: " + payload.Customer.Phone
			}
			if payload.Customer.Email != "" {
				customerContact += "\n✉️ Email: " + payload.Customer.Email
			}

			adminAlert := fmt.Sprintf(
				"🔥 <b>Новая оплата через Antilopay!</b>\n\n"+
					"💳 <b>ID платежа:</b> <code>%s</code>\n"+
					"👤 <b>Студент:</b> %s\n"+
					"📦 <b>Тариф:</b> %s\n"+
					"💰 <b>Сумма:</b> %.2f ₽\n"+
					"🏷 <b>Промокод:</b> %s%s\n\n"+
					"✅ <i>Доступ студенту открыт автоматически, одноразовая ссылка отправлена.</i>",
				paymentID, studentRef, tierNameRU, payload.Amount, promoBadge, customerContact,
			)

			_ = h.notifyFunc(adminTG, adminAlert)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":     "ok",
		"payment_id": paymentID,
		"message":    "callback processed successfully",
	})
}
