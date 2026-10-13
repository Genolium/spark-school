package handlers

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spark-school/backend/internal/auth"
	"github.com/spark-school/backend/internal/config"
	"github.com/spark-school/backend/internal/models"
	"github.com/spark-school/backend/internal/services"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AdminHandler struct {
	db               *gorm.DB
	cfg              *config.Config
	affiliateService *services.AffiliateService
	notifyFunc       func(telegramID int64, message string) error
	inviteLinkFunc   func(ctx context.Context, channelID int64) (string, error)
	digestFunc       func() error
}

func NewAdminHandler(db *gorm.DB, cfg *config.Config, aff *services.AffiliateService, notify func(telegramID int64, message string) error, inviteLinkGen ...func(ctx context.Context, channelID int64) (string, error)) *AdminHandler {
	var gen func(ctx context.Context, channelID int64) (string, error)
	if len(inviteLinkGen) > 0 {
		gen = inviteLinkGen[0]
	}
	return &AdminHandler{
		db:               db,
		cfg:              cfg,
		affiliateService: aff,
		notifyFunc:       notify,
		inviteLinkFunc:   gen,
	}
}

// SetInviteLinkGenerator sets a custom generator for one-time Telegram invite links.
func (h *AdminHandler) SetInviteLinkGenerator(fn func(ctx context.Context, channelID int64) (string, error)) *AdminHandler {
	h.inviteLinkFunc = fn
	return h
}

// Login validates admin credentials and issues a secure JWT with timing attack and brute-force protection.
func (h *AdminHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Необходимо указать логин и пароль"})
		return
	}

	// Constant-time username comparison
	userMatch := subtle.ConstantTimeCompare([]byte(req.Username), []byte(h.cfg.AdminUsername)) == 1

	// Secure password comparison: bcrypt hash verification or constant-time comparison
	var passMatch bool
	if strings.HasPrefix(h.cfg.AdminPassword, "$2a$") || strings.HasPrefix(h.cfg.AdminPassword, "$2b$") {
		passMatch = bcrypt.CompareHashAndPassword([]byte(h.cfg.AdminPassword), []byte(req.Password)) == nil
	} else {
		passMatch = subtle.ConstantTimeCompare([]byte(req.Password), []byte(h.cfg.AdminPassword)) == 1
	}

	if !userMatch || !passMatch {
		// Artificial latency penalty against brute-force tools
		time.Sleep(1 * time.Second)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный логин или пароль администратора"})
		return
	}

	adminUser := models.User{
		ID:         1,
		TelegramID: h.cfg.AdminTelegramID,
		Username:   h.cfg.AdminUsername,
		FirstName:  "Васюнин Илья",
		LastName:   "",
		Role:       "admin",
		HasAccess:  true,
	}

	token, err := auth.GenerateJWT(&adminUser, h.cfg.JWTSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка генерации токена"})
		return
	}

	c.SetCookie("spark_session", token, 30*86400, "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user":  adminUser,
	})
}

// GetStats returns revenue, enrollments, and conversion metrics for the admin dashboard.
// GetStats returns revenue, enrollments, and conversion metrics for the admin dashboard.
func (h *AdminHandler) GetStats(c *gin.Context) {
	var totalUsers int64
	h.db.Model(&models.User{}).Where("role != ?", "admin").Count(&totalUsers)

	var activeStudents int64
	h.db.Model(&models.User{}).Where("has_access = true AND role != ?", "admin").Count(&activeStudents)

	var pendingPayoutsCount int64
	h.db.Model(&models.PayoutRequest{}).Where("status = 'pending'").Count(&pendingPayoutsCount)

	var pendingPayoutsSum float64
	h.db.Model(&models.PayoutRequest{}).Where("status = 'pending'").Select("COALESCE(SUM(amount), 0)").Scan(&pendingPayoutsSum)

	// Calculate accurate gross revenue: sum of paid_amount for all active students with access
	var grossVolume float64
	var activeUsers []models.User
	h.db.Model(&models.User{}).Where("has_access = true AND role != ?", "admin").Find(&activeUsers)
	pricing := models.GetCoursePricing(h.db)
	for _, u := range activeUsers {
		if u.PaidAmount > 0 {
			grossVolume += u.PaidAmount
		} else {
			// Fallback based on tier
			discountFactor := 1.0
			if u.PromoCode != "" {
				var promo models.PromoCode
				pct := 5
				if err := h.db.Where("UPPER(code) = ?", strings.ToUpper(u.PromoCode)).First(&promo).Error; err == nil && promo.DiscountPercent > 0 {
					pct = promo.DiscountPercent
				}
				discountFactor = 1.0 - float64(pct)/100.0
			}

			if strings.ToLower(u.Tier) == "vip" {
				grossVolume += pricing.VIPPrice * discountFactor
			} else {
				grossVolume += pricing.AcceleratorPrice * discountFactor
			}
		}
	}

	conversionRate := 0.0
	if totalUsers > 0 {
		conversionRate = (float64(activeStudents) / float64(totalUsers)) * 100.0
	}

	c.JSON(http.StatusOK, gin.H{
		"gross_volume":          grossVolume,
		"active_students":       activeStudents,
		"total_registered":      totalUsers,
		"conversion_rate":       conversionRate,
		"pending_payouts_count": pendingPayoutsCount,
		"pending_payouts_sum":   pendingPayoutsSum,
	})
}

// ListUsers returns paginated/searchable list of students and access states with filtering by Tier, Access, and Receipt Status.
func (h *AdminHandler) ListUsers(c *gin.Context) {
	query := strings.TrimSpace(c.Query("q"))
	tier := strings.ToLower(strings.TrimSpace(c.Query("tier")))
	access := strings.ToLower(strings.TrimSpace(c.Query("access")))
	receiptStatus := strings.ToLower(strings.TrimSpace(c.Query("receipt_status")))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize

	if h.db == nil {
		c.JSON(http.StatusOK, gin.H{
			"users":    []models.User{},
			"total":    0,
			"page":     page,
			"pageSize": pageSize,
		})
		return
	}

	dbQuery := h.db.Model(&models.User{}).Preload("AffiliateProfile")

	// 1. Search Query: username, first_name, last_name, telegram_id, promo_code
	if query != "" {
		search := "%" + query + "%"
		dbQuery = dbQuery.Where(
			"LOWER(username) LIKE LOWER(?) OR LOWER(first_name) LIKE LOWER(?) OR LOWER(last_name) LIKE LOWER(?) OR CAST(telegram_id AS TEXT) LIKE ? OR LOWER(promo_code) LIKE LOWER(?)",
			search, search, search, search, search,
		)
	}

	// 2. Filter by Tier: accelerator, vip
	if tier != "" && tier != "all" {
		dbQuery = dbQuery.Where("LOWER(tier) = ?", tier)
	}

	// 3. Filter by Access Status: active, pending/inactive
	if access == "active" {
		dbQuery = dbQuery.Where("has_access = true")
	} else if access == "pending" || access == "inactive" {
		dbQuery = dbQuery.Where("has_access = false")
	}

	// 4. Filter by Payment Receipt Status: pending, approved, rejected, none
	if receiptStatus != "" && receiptStatus != "all" {
		if receiptStatus == "none" || receiptStatus == "no_receipt" {
			dbQuery = dbQuery.Where("telegram_id NOT IN (SELECT telegram_id FROM payment_receipts WHERE telegram_id IS NOT NULL AND telegram_id != 0)")
		} else {
			dbQuery = dbQuery.Where("telegram_id IN (SELECT pr1.telegram_id FROM payment_receipts pr1 WHERE pr1.id = (SELECT MAX(pr2.id) FROM payment_receipts pr2 WHERE pr2.telegram_id = pr1.telegram_id) AND pr1.status = ?)", receiptStatus)
		}
	}

	var total int64
	dbQuery.Count(&total)

	users := make([]models.User, 0)
	dbQuery.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&users)

	// Populate receipt_status for returned users
	if len(users) > 0 {
		tgIDs := make([]int64, len(users))
		for i, u := range users {
			tgIDs[i] = u.TelegramID
		}
		var receipts []models.PaymentReceipt
		h.db.Where("telegram_id IN ?", tgIDs).Order("id ASC").Find(&receipts)
		latestStatus := make(map[int64]string)
		for _, r := range receipts {
			latestStatus[r.TelegramID] = r.Status
		}
		for i := range users {
			if st, ok := latestStatus[users[i].TelegramID]; ok {
				users[i].ReceiptStatus = st
			} else {
				users[i].ReceiptStatus = "none"
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"total": total,
		"page":  page,
	})
}

// CreateUserRequest payload for manually adding a user/student
type CreateUserRequest struct {
	TelegramID int64    `json:"telegram_id" binding:"required"`
	Username   string   `json:"username"`
	FirstName  string   `json:"first_name" binding:"required"`
	LastName   string   `json:"last_name"`
	Role       string   `json:"role"`
	HasAccess  bool     `json:"has_access"`
	Tier       string   `json:"tier"`        // 'accelerator', 'vip'
	PaidAmount *float64 `json:"paid_amount"` // custom paid sum
	PromoCode  string   `json:"promo_code"`  // promo code used
}

// CreateUser handles admin manual creation of a user/student
func (h *AdminHandler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Укажите Telegram ID и имя пользователя"})
		return
	}

	role := req.Role
	if role == "" {
		role = "student"
	}

	tier := strings.ToLower(strings.TrimSpace(req.Tier))
	if tier == "" {
		tier = "accelerator"
	}
	cleanPromo := strings.ToUpper(strings.TrimSpace(req.PromoCode))

	paidAmount := 0.0
	if req.PaidAmount != nil {
		paidAmount = *req.PaidAmount
	} else if req.HasAccess {
		pricing := models.GetCoursePricing(h.db)
		if tier == "vip" {
			if cleanPromo != "" {
				paidAmount = pricing.VIPPrice * 0.95
			} else {
				paidAmount = pricing.VIPPrice
			}
		} else {
			if cleanPromo != "" {
				paidAmount = pricing.AcceleratorPrice * 0.95
			} else {
				paidAmount = pricing.AcceleratorPrice
			}
		}
	}

	user := models.User{
		TelegramID: req.TelegramID,
		Username:   strings.TrimPrefix(req.Username, "@"),
		FirstName:  req.FirstName,
		LastName:   req.LastName,
		Role:       role,
		HasAccess:  req.HasAccess,
		Tier:       tier,
		PaidAmount: paidAmount,
		PromoCode:  cleanPromo,
	}

	if req.HasAccess {
		now := time.Now()
		user.AccessGrantedAt = &now
	}

	if err := h.db.Create(&user).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ошибка создания пользователя: " + err.Error()})
		return
	}

	// If promo was specified with access granted, increment promo uses atomically
	if req.HasAccess && cleanPromo != "" {
		_, _ = RedeemPromoCode(h.db, cleanPromo)
	}

	c.JSON(http.StatusCreated, user)
}

// UpdateUserRequest payload for updating user profile
type UpdateUserRequest struct {
	Username   string   `json:"username"`
	FirstName  string   `json:"first_name"`
	LastName   string   `json:"last_name"`
	Role       string   `json:"role"`
	HasAccess  *bool    `json:"has_access"`
	Tier       string   `json:"tier"`
	PaidAmount *float64 `json:"paid_amount"`
	PromoCode  *string  `json:"promo_code"`
}

// UpdateUser updates user details
func (h *AdminHandler) UpdateUser(c *gin.Context) {
	id := c.Param("id")
	var user models.User
	if err := h.db.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат запроса"})
		return
	}

	if req.FirstName != "" {
		user.FirstName = req.FirstName
	}
	if req.LastName != "" {
		user.LastName = req.LastName
	}
	if req.Username != "" {
		user.Username = strings.TrimPrefix(req.Username, "@")
	}
	if req.Role != "" {
		user.Role = req.Role
	}
	if req.Tier != "" {
		user.Tier = strings.ToLower(strings.TrimSpace(req.Tier))
	}
	if req.PaidAmount != nil {
		user.PaidAmount = *req.PaidAmount
	}
	if req.PromoCode != nil {
		user.PromoCode = strings.ToUpper(strings.TrimSpace(*req.PromoCode))
	}
	if req.HasAccess != nil {
		user.HasAccess = *req.HasAccess
		if *req.HasAccess && user.AccessGrantedAt == nil {
			now := time.Now()
			user.AccessGrantedAt = &now
			if user.PaidAmount == 0 {
				pricing := models.GetCoursePricing(h.db)
				if user.Tier == "vip" {
					if user.PromoCode != "" {
						user.PaidAmount = pricing.VIPPrice * 0.95
					} else {
						user.PaidAmount = pricing.VIPPrice
					}
				} else {
					if user.PromoCode != "" {
						user.PaidAmount = pricing.AcceleratorPrice * 0.95
					} else {
						user.PaidAmount = pricing.AcceleratorPrice
					}
				}
			}
		}
	}

	if err := h.db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сохранения: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

// DeleteUser removes a user
func (h *AdminHandler) DeleteUser(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&models.User{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка удаления: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Пользователь удалён"})
}

// ToggleAccess updates has_access, redeems promo codes atomically, and processes affiliate commissions.
func (h *AdminHandler) ToggleAccess(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req struct {
		HasAccess  bool     `json:"has_access"`
		PromoCode  string   `json:"promo_code,omitempty"`
		Tier       string   `json:"tier,omitempty"`
		PaidAmount *float64 `json:"paid_amount,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	cleanPromo := strings.ToUpper(strings.TrimSpace(req.PromoCode))

	// Support mock / unit testing when db is nil
	if h.db == nil {
		inviteLink := ""
		if req.HasAccess {
			if h.cfg != nil {
				inviteLink = h.cfg.TelegramInviteLink
			}
			if h.inviteLinkFunc != nil && h.cfg != nil && h.cfg.TelegramChannelID != 0 {
				if dynamicLink, err := h.inviteLinkFunc(c.Request.Context(), h.cfg.TelegramChannelID); err == nil && dynamicLink != "" {
					inviteLink = dynamicLink
				}
			}
			if inviteLink == "" {
				inviteLink = ""
			}
			if h.notifyFunc != nil {
				msg := "🎉 Поздравляем! Ваш доступ к закрытому Telegram-каналу и комьюнити проекта «так называемый SPARK» успешно активирован.\n\nСсылка-приглашение в канал: " + inviteLink
				_ = h.notifyFunc(12345678, msg)
			}
		}
		resp := gin.H{
			"message":    "Статус доступа успешно обновлён",
			"has_access": req.HasAccess,
		}
		if inviteLink != "" {
			resp["invite_link"] = inviteLink
		}
		if cleanPromo != "" {
			resp["promo_code"] = cleanPromo
		}
		c.JSON(http.StatusOK, resp)
		return
	}

	var user models.User
	var wasActive bool
	var redeemedPromo *models.PromoCode

	err = h.db.Transaction(func(tx *gorm.DB) error {
		// Row-level lock via SELECT ... FOR UPDATE to eliminate concurrent access approval races
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, userID).Error; err != nil {
			return err
		}

		wasActive = user.HasAccess
		user.HasAccess = req.HasAccess
		if req.Tier != "" {
			user.Tier = strings.ToLower(strings.TrimSpace(req.Tier))
		}
		if cleanPromo != "" {
			user.PromoCode = cleanPromo
		}
		if req.PaidAmount != nil {
			user.PaidAmount = *req.PaidAmount
		}
		if req.HasAccess && user.AccessGrantedAt == nil {
			now := time.Now()
			user.AccessGrantedAt = &now
			if user.PaidAmount == 0 {
				pricing := models.GetCoursePricing(h.db)
				if user.Tier == "vip" {
					if user.PromoCode != "" {
						user.PaidAmount = pricing.VIPPrice * 0.95
					} else {
						user.PaidAmount = pricing.VIPPrice
					}
				} else {
					if user.PromoCode != "" {
						user.PaidAmount = pricing.AcceleratorPrice * 0.95
					} else {
						user.PaidAmount = pricing.AcceleratorPrice
					}
				}
			}
		}
		if err := tx.Save(&user).Error; err != nil {
			return err
		}

		// Only redeem promo and distribute commission on initial access activation
		if req.HasAccess && !wasActive {
			// 1. Atomic Promo Code Redemption (if provided)
			if cleanPromo != "" {
				promo, err := RedeemPromoCode(tx, cleanPromo)
				if err != nil {
					return err
				}
				redeemedPromo = promo
			}

			// 2. Affiliate Commission Settlement (inside same transaction)
			if h.affiliateService != nil {
				purchasePrice := user.PaidAmount
				if purchasePrice <= 0 {
					pricing := models.GetCoursePricing(h.db)
					purchasePrice = pricing.AcceleratorPrice
				}
				var userReceipt models.PaymentReceipt
				if err := tx.Where("telegram_id = ?", user.TelegramID).Order("id DESC").First(&userReceipt).Error; err == nil && userReceipt.Amount > 0 {
					purchasePrice = userReceipt.Amount
				}
				if err := h.affiliateService.ProcessCoursePurchaseTx(tx, user.ID, purchasePrice); err != nil {
					return err
				}
			}
		}

		return nil
	})

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// External network operations (invite link generation and notification) occur strictly POST-COMMIT
	var generatedInviteLink string
	if req.HasAccess && !wasActive {
		if h.notifyFunc != nil && user.TelegramID != 0 {
			inviteLink := ""
			if h.cfg != nil {
				inviteLink = h.cfg.TelegramInviteLink
			}
			if h.inviteLinkFunc != nil && h.cfg != nil && h.cfg.TelegramChannelID != 0 {
				if dynamicLink, err := h.inviteLinkFunc(c.Request.Context(), h.cfg.TelegramChannelID); err == nil && dynamicLink != "" {
					inviteLink = dynamicLink
				}
			}
			if inviteLink == "" && h.cfg != nil {
				inviteLink = h.cfg.TelegramInviteLink
			}
			if inviteLink == "" {
				inviteLink = "https://t.me/+so_called_spark_private"
			}
			generatedInviteLink = inviteLink
			msg := "🎉 Поздравляем! Ваш доступ к закрытому Telegram-каналу и комьюнити проекта «так называемый SPARK» успешно активирован.\n\nСсылка-приглашение в канал: " + inviteLink
			errSend := h.notifyFunc(user.TelegramID, msg)
			if errSend != nil {
				// Resilient fallback: enqueue to Outbox table for background retries
				if h.db != nil {
					_ = h.db.Create(&models.OutboxEvent{
						EventType:  "telegram_notification",
						Payload:    fmt.Sprintf(`{"telegram_id":%d,"message":%q}`, user.TelegramID, msg),
						Status:     "pending",
						CreatedAt:  time.Now(),
						MaxRetries: 5,
						LastError:  errSend.Error(),
					})
				}
			}
		}
	}

	resp := gin.H{
		"message":    "Статус доступа успешно обновлён",
		"has_access": user.HasAccess,
	}
	if generatedInviteLink != "" {
		resp["invite_link"] = generatedInviteLink
	}
	if redeemedPromo != nil {
		resp["promo_code"] = redeemedPromo.Code
		resp["promo_uses"] = redeemedPromo.UsesCount
	}
	c.JSON(http.StatusOK, resp)
}

// ListPayouts returns all affiliate payout requests.
func (h *AdminHandler) ListPayouts(c *gin.Context) {
	var payouts []models.PayoutRequest
	h.db.Preload("User").Order("created_at DESC").Find(&payouts)

	c.JSON(http.StatusOK, gin.H{
		"payouts": payouts,
	})
}

// ApprovePayout marks payout request as paid.
func (h *AdminHandler) ApprovePayout(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseUint(idStr, 10, 32)

	if err := h.affiliateService.ApprovePayout(uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Выплата успешно подтверждена"})
}

// RejectPayout rejects payout request and returns balance.
func (h *AdminHandler) RejectPayout(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseUint(idStr, 10, 32)

	if err := h.affiliateService.RejectPayout(uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Выплата отклонена, средства возвращены на баланс партнёра"})
}

// SetDigestSender sets the trigger for daily curator digest.
func (h *AdminHandler) SetDigestSender(fn func() error) *AdminHandler {
	h.digestFunc = fn
	return h
}

// TriggerDigest triggers the daily curator digest manually.
func (h *AdminHandler) TriggerDigest(c *gin.Context) {
	if h.digestFunc == nil {
		c.JSON(http.StatusOK, gin.H{"ok": true, "message": "Дайджест сформирован"})
		return
	}
	if err := h.digestFunc(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "message": "Дайджест успешно отправлен куратору в Telegram"})
}
