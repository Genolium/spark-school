package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
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

type YooKassaHandler struct {
	db               *gorm.DB
	cfg              *config.Config
	yooKassaService  *services.YooKassaService
	affiliateService *services.AffiliateService
	notifyFunc       func(telegramID int64, message string) error
	inviteLinkFunc   func(ctx context.Context, channelID int64) (string, error)
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
	}
}

type CreateYooKassaCheckoutRequest struct {
	Tier             string `json:"tier"` // "accelerator" or "vip"
	TelegramUsername string `json:"telegram_username"`
	TelegramID       int64  `json:"telegram_id"`
	PromoCode        string `json:"promo_code"`
	Email            string `json:"email"`
}

// CreateCheckout creates a payment session via official YooKassa API and returns the redirect URL.
// POST /api/v1/payments/yookassa/create
func (h *YooKassaHandler) CreateCheckout(c *gin.Context) {
	var req CreateYooKassaCheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат запроса"})
		return
	}

	tier := strings.ToLower(strings.TrimSpace(req.Tier))
	if tier != "vip" {
		tier = "accelerator"
	}

	cleanPromo := strings.ToUpper(strings.TrimSpace(req.PromoCode))
	basePrice := 6900.00
	if tier == "vip" {
		basePrice = 14900.00
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
	if tier == "vip" {
		tierNameRU = "VIP"
	}

	domain := "so-called-spark.ru"
	if h.cfg != nil && h.cfg.Domain != "" {
		domain = h.cfg.Domain
	}

	description := fmt.Sprintf("Оплата участия в проекте «так называемый SPARK» (Тариф %s)", tierNameRU)
	returnURL := fmt.Sprintf("https://%s/#payment-success", domain)

	metadata := map[string]string{
		"tier":       tier,
		"username":   strings.TrimPrefix(req.TelegramUsername, "@"),
		"promo_code": cleanPromo,
	}
	if req.TelegramID > 0 {
		metadata["telegram_id"] = strconv.FormatInt(req.TelegramID, 10)
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

	if h.yooKassaService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Платёжный сервис ЮKassa временно недоступен"})
		return
	}

	paymentResp, err := h.yooKassaService.CreatePayment(c.Request.Context(), ykReq)
	if err != nil {
		log.Printf("[YOOKASSA-ERROR] CreatePayment failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Не удалось сформировать счёт в ЮKassa. Пожалуйста, попробуйте позже.",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"payment_id":       paymentResp.ID,
		"status":           paymentResp.Status,
		"confirmation_url": paymentResp.Confirmation.ConfirmationURL,
		"amount":           finalPrice,
		"tier":             tier,
	})
}

// YooKassaWebhookPayload represents YooKassa notification webhook schema.
type YooKassaWebhookPayload struct {
	Type   string `json:"type"`  // "notification"
	Event  string `json:"event"` // "payment.succeeded", "payment.canceled", etc.
	Object struct {
		ID            string                 `json:"id"`
		Status        string                 `json:"status"` // "succeeded"
		Paid          bool                   `json:"paid"`
		Amount        services.YooKassaAmount `json:"amount"`
		Description   string                 `json:"description"`
		Metadata      map[string]string      `json:"metadata"`
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

	// Gold Standard Security Verification:
	// Verify payment directly against official YooKassa API to eliminate spoofing risks
	if h.yooKassaService != nil && h.cfg != nil && h.cfg.YooKassaSecretKey != "" {
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
		log.Printf("[YOOKASSA-SECURITY] Payment %s verified directly with YooKassa API: 100%% authentic.", paymentID)
	}

	// Extract amount
	amount, _ := strconv.ParseFloat(payload.Object.Amount.Value, 64)
	currency := payload.Object.Amount.Currency
	if currency == "" {
		currency = "RUB"
	}

	// Anti-underpayment protection
	if amount < MinCoursePaymentThreshold {
		log.Printf("[SECURITY-ALERT] YooKassa payment %s underpayment detected: %.2f %s. Minimum: %.2f.",
			paymentID, amount, currency, MinCoursePaymentThreshold)

		adminTG := int64(123456789)
		if h.cfg != nil && h.cfg.AdminTelegramID != 0 {
			adminTG = h.cfg.AdminTelegramID
		}
		if h.notifyFunc != nil && adminTG != 0 {
			alertMsg := fmt.Sprintf(
				"⚠️ <b>ВНИМАНИЕ: ПОДОЗРИТЕЛЬНАЯ ОПЛАТА ЮKASSA!</b> ⚠️\n\n"+
					"💰 <b>Сумма:</b> %.2f %s <i>(МЕНЬШЕ МИНИМУМА 6 000 ₽)</i>\n"+
					"🆔 <b>ID платежа:</b> <code>%s</code>\n\n"+
					"⛔️ <b>Доступ НЕ выдан.</b> Проверьте транзакцию в кабинете ЮKassa вручную!",
				amount, currency, paymentID,
			)
			_ = h.notifyFunc(adminTG, alertMsg)
		}

		c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "payment_flagged_underpaid"})
		return
	}

	// Extract metadata
	tier := "accelerator"
	promoCode := ""
	username := ""
	var telegramID int64

	if payload.Object.Metadata != nil {
		m := payload.Object.Metadata
		if t, ok := m["tier"]; ok && strings.ToLower(t) == "vip" {
			tier = "vip"
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
	}

	if amount >= MinVipPaymentThreshold {
		tier = "vip"
	}

	// Concurrency & Idempotency Execution in DB Transaction
	var isAlreadyProcessed bool
	var user models.User

	if h.db != nil {
		err = h.db.Transaction(func(tx *gorm.DB) error {
			var existing models.PaymentTransaction
			if err := tx.Where("payment_id = ?", paymentID).First(&existing).Error; err == nil {
				isAlreadyProcessed = true
				return nil
			}

			payMethod := "YOOKASSA"
			if payload.Object.PaymentMethod.Type != "" {
				payMethod = "YOOKASSA_" + strings.ToUpper(payload.Object.PaymentMethod.Type)
			}

			transaction := models.PaymentTransaction{
				PaymentID:      paymentID,
				OrderID:        paymentID,
				TelegramID:     telegramID,
				Username:       username,
				Tier:           tier,
				Amount:         amount,
				OriginalAmount: amount,
				Status:         "SUCCESS",
				Currency:       currency,
				PayMethod:      payMethod,
				PromoCode:      promoCode,
				ProductName:    payload.Object.Description,
				RawCallback:    string(bodyBytes),
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			}

			if err := tx.Create(&transaction).Error; err != nil {
				if strings.Contains(strings.ToLower(err.Error()), "duplicate") || strings.Contains(strings.ToLower(err.Error()), "unique") {
					isAlreadyProcessed = true
					return nil
				}
				return err
			}

			// Update or Create Student Record
			if telegramID > 0 {
				now := time.Now()
				if err := tx.Where("telegram_id = ?", telegramID).First(&user).Error; err == nil {
					user.HasAccess = true
					user.AccessGrantedAt = &now
					user.Tier = tier
					user.PaidAmount = amount
					if promoCode != "" {
						user.PromoCode = promoCode
					}
					if username != "" && user.Username == "" {
						user.Username = username
					}
					if err := tx.Save(&user).Error; err != nil {
						return err
					}
				} else {
					user = models.User{
						TelegramID:      telegramID,
						Username:        username,
						FirstName:       "Студент",
						Role:            "student",
						HasAccess:       true,
						AccessGrantedAt: &now,
						Tier:            tier,
						PaidAmount:      amount,
						PromoCode:       promoCode,
					}
					if err := tx.Create(&user).Error; err != nil {
						return err
					}
				}

				// Redeem promo code atomically
				if promoCode != "" {
					_, _ = RedeemPromoCode(tx, promoCode)
				}

				// Affiliate Attribution
				if h.affiliateService != nil && user.ID != 0 {
					_ = h.affiliateService.ProcessCoursePurchaseTx(tx, user.ID, amount)
				}
			}

			return nil
		})

		if err != nil {
			log.Printf("[YOOKASSA-ERROR] DB transaction failed: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database transaction error"})
			return
		}

		if isAlreadyProcessed {
			log.Printf("[YOOKASSA-IDEMPOTENCY] Payment %s already processed. Skipping duplicate safely.", paymentID)
			c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "already_processed"})
			return
		}
	}

	// Generate single-use channel invite link
	inviteLink := ""
	if h.cfg != nil {
		inviteLink = h.cfg.TelegramInviteLink
	}
	if h.inviteLinkFunc != nil {
		var chanID int64
		if h.cfg != nil {
			chanID = h.cfg.TelegramChannelID
		}
		if dynLink, err := h.inviteLinkFunc(context.Background(), chanID); err == nil && dynLink != "" {
			inviteLink = dynLink
		}
	}

	tierNameRU := "Акселератор (6 900 ₽)"
	if tier == "vip" {
		tierNameRU = "VIP-сопровождение (14 900 ₽)"
	}

	// Deliver to student in Telegram
	if h.notifyFunc != nil && telegramID > 0 {
		studentMsg := fmt.Sprintf(
			"🎉 <b>Оплата через ЮKassa успешно подтверждена!</b>\n\n"+
				"Добро пожаловать в проект «так называемый SPARK» 2027!\n\n"+
				"📦 <b>Твой тариф:</b> %s\n"+
				"💳 <b>Оплачено:</b> %.2f ₽\n\n"+
				"🔒 <b>Твоя персональная ссылка в закрытый канал и комьюнити:</b>\n%s\n\n"+
				"<i>Ссылка одноразовая и действует 24 часа. Сохрани её.</i>",
			tierNameRU, amount, inviteLink,
		)
		_ = h.notifyFunc(telegramID, studentMsg)
	}

	// Instant push alert to Ilya
	adminTG := int64(123456789)
	if h.cfg != nil && h.cfg.AdminTelegramID != 0 {
		adminTG = h.cfg.AdminTelegramID
	}

	if h.notifyFunc != nil && adminTG != 0 {
		studentRef := fmt.Sprintf("ID: %d", telegramID)
		if username != "" {
			studentRef = fmt.Sprintf("@%s (%s)", html.EscapeString(username), studentRef)
		}

		promoBadge := "— (без промокода)"
		if promoCode != "" {
			promoBadge = fmt.Sprintf("🏷 %s (-5%%)", html.EscapeString(promoCode))
		}

		adminAlert := fmt.Sprintf(
			"⚡️ <b>НОВАЯ ОПЛАТА ЧЕРЕЗ ЮKASSA!</b> ⚡️\n\n"+
				"👤 <b>Студент:</b> %s\n"+
				"📦 <b>Тариф:</b> %s\n"+
				"💰 <b>Сумма:</b> %.2f %s\n"+
				"🎟 <b>Промокод:</b> %s\n"+
				"🆔 <b>ID платежа:</b> <code>%s</code>\n"+
				"🏪 <b>Магазин:</b> ShopID <code>1491540</code> (so-called-spark.ru)\n"+
				"💳 <b>Шлюз:</b> ЮKassa (ООО НКО «ЮМани» / Сбер)\n"+
				"⏱ <b>Время:</b> %s\n\n"+
				"✅ <i>Официальный платёж подтверждён, доступ выдан автоматически!</i>",
			studentRef,
			tierNameRU,
			amount,
			currency,
			promoBadge,
			paymentID,
			time.Now().Format("15:04:05 02.01.2006"),
		)
		_ = h.notifyFunc(adminTG, adminAlert)
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "payment processed successfully"})
}
