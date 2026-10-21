package handlers

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spark-school/backend/internal/auth"
	"github.com/spark-school/backend/internal/config"
	"github.com/spark-school/backend/internal/models"
	"github.com/spark-school/backend/internal/money"
	"github.com/spark-school/backend/internal/privacy"
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
	receiptSender    func(telegramID int64, tier string, amount float64, receiptURL string, receiptNumber string, serviceName string) error
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

// SetReceiptSender sets the handler function for dispatching fiscal receipts to students via Telegram.
func (h *AdminHandler) SetReceiptSender(fn func(telegramID int64, tier string, amount float64, receiptURL string, receiptNumber string, serviceName string) error) *AdminHandler {
	h.receiptSender = fn
	return h
}

// Login validates admin credentials and issues a secure JWT with timing attack and brute-force protection.
func (h *AdminHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		TOTPCode string `json:"totp_code"`
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

	if h.cfg.AdminTOTPSecret != "" && !auth.VerifyTOTP(h.cfg.AdminTOTPSecret, req.TOTPCode, time.Now()) {
		time.Sleep(500 * time.Millisecond)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный или отсутствующий код 2FA", "totp_required": true})
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

	token, err := auth.GenerateAdminJWT(&adminUser, h.cfg.JWTSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка генерации токена"})
		return
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("spark_session", token, 12*3600, "/", "", h.cfg.IsProduction(), true)

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user":  adminUser,
	})
}

// Logout revokes the current administrative token and clears the cookie.
func (h *AdminHandler) Logout(c *gin.Context) {
	value, ok := c.Get("jwt_claims")
	if ok {
		if claims, ok := value.(*auth.JWTClaims); ok {
			_ = auth.RevokeJWT(h.db, claims)
		}
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("spark_session", "", -1, "/", "", h.cfg.IsProduction(), true)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// DailyStat represents daily revenue and order metrics for admin charts.
type DailyStat struct {
	Date              string  `json:"date"`
	DateLabel         string  `json:"date_label"`
	Revenue           float64 `json:"revenue"`
	Purchases         int     `json:"purchases"`
	CumulativeRevenue float64 `json:"cumulative_revenue"`
	BasicCount        int     `json:"basic_count"`
	AcceleratorCount  int     `json:"accelerator_count"`
	VIPCount          int     `json:"vip_count"`
}

// GetStats returns revenue, enrollments, and conversion metrics for the admin dashboard.
func (h *AdminHandler) GetStats(c *gin.Context) {
	if h.db == nil {
		c.JSON(http.StatusOK, gin.H{
			"gross_volume":          0.0,
			"active_students":       0,
			"total_registered":      0,
			"conversion_rate":       0.0,
			"pending_payouts_count": 0,
			"pending_payouts_sum":   0.0,
			"today_revenue":         0.0,
			"today_purchases":       0,
			"week_revenue":          0.0,
			"month_revenue":         0.0,
			"average_order_value":   0.0,
			"promo_usage_count":     0,
			"promo_saved_amount":    0.0,
			"tier_breakdown": gin.H{
				"basic":       gin.H{"count": 0, "revenue": 0.0, "percentage": 0.0},
				"accelerator": gin.H{"count": 0, "revenue": 0.0, "percentage": 0.0},
				"vip":         gin.H{"count": 0, "revenue": 0.0, "percentage": 0.0},
			},
			"daily_stats": []DailyStat{},
		})
		return
	}

	var totalUsers int64
	h.db.Model(&models.User{}).Where("role != ?", "admin").Count(&totalUsers)

	var activeStudents int64
	h.db.Model(&models.User{}).Where("has_access = true AND role != ?", "admin").Count(&activeStudents)

	var pendingPayoutsCount int64
	h.db.Model(&models.PayoutRequest{}).Where("status = 'pending'").Count(&pendingPayoutsCount)

	var pendingPayoutsSum float64
	h.db.Model(&models.PayoutRequest{}).Where("status = 'pending'").Select("COALESCE(SUM(amount), 0)").Scan(&pendingPayoutsSum)

	// Calculate accurate gross revenue from actual financial transactions (payments and receipts)
	var grossVolume float64
	pricing := models.GetCoursePricing(h.db)

	msk := time.FixedZone("MSK", 3*3600)
	now := time.Now().In(msk)
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, msk)
	weekStart := todayStart.AddDate(0, 0, -6)
	monthStart := todayStart.AddDate(0, 0, -29)

	var todayRevenue float64
	var todayPurchases int
	var weekRevenue float64
	var monthRevenue float64
	var promoUsageCount int
	var promoSavedAmount float64

	basicCount := 0
	basicRevenue := 0.0
	acceleratorCount := 0
	acceleratorRevenue := 0.0
	vipCount := 0
	vipRevenue := 0.0

	// 30-day timeline map
	ruMonths := []string{"", "янв", "фев", "мар", "апр", "май", "июн", "июл", "авг", "сен", "окт", "ноя", "дек"}
	dailyMap := make(map[string]*DailyStat)
	var orderedDates []string

	for d := monthStart; !d.After(todayStart); d = d.AddDate(0, 0, 1) {
		dateKey := d.Format("2006-01-02")
		mIdx := int(d.Month())
		label := fmt.Sprintf("%02d %s", d.Day(), ruMonths[mIdx])
		dailyMap[dateKey] = &DailyStat{
			Date:      dateKey,
			DateLabel: label,
		}
		orderedDates = append(orderedDates, dateKey)
	}

	var preMonthRevenue float64
	var grossKopecks money.Kopecks

	// 1. Gather all actual confirmed financial operations
	type TransactionRecord struct {
		Amount        float64
		AmountKopecks int64
		Tier          string
		CreatedAt     time.Time
		PromoCode     string
	}
	var txRecords []TransactionRecord

	var pts []models.PaymentTransaction
	h.db.Where("status = 'SUCCESS'").Order("created_at ASC").Find(&pts)
	for _, pt := range pts {
		t := strings.ToLower(strings.TrimSpace(pt.Tier))
		if t == "" {
			t = "accelerator"
		}
		kopecks := pt.AmountKopecks
		if kopecks == 0 {
			kopecks = int64(money.FromRubles(pt.Amount))
		}
		txRecords = append(txRecords, TransactionRecord{
			Amount:        money.Kopecks(kopecks).ToRubles(),
			AmountKopecks: kopecks,
			Tier:          t,
			CreatedAt:     pt.CreatedAt,
			PromoCode:     pt.PromoCode,
		})
	}

	var activeUsers []models.User
	h.db.Model(&models.User{}).Where("has_access = true AND role != ?", "admin").Find(&activeUsers)

	var freeAccessCount int64
	for _, u := range activeUsers {
		if u.PaidAmount <= 0 {
			freeAccessCount++
		}
	}

	for _, tx := range txRecords {
		grossKopecks += money.Kopecks(tx.AmountKopecks)
		grossVolume += tx.Amount
		switch tx.Tier {
		case "vip":
			vipCount++
			vipRevenue += tx.Amount
		case "basic":
			basicCount++
			basicRevenue += tx.Amount
		default:
			acceleratorCount++
			acceleratorRevenue += tx.Amount
		}

		if tx.PromoCode != "" {
			promoUsageCount++
			if tx.Tier == "vip" {
				promoSavedAmount += pricing.VIPPrice * 0.05
			} else if tx.Tier == "basic" {
				promoSavedAmount += pricing.BasicPrice * 0.05
			} else {
				promoSavedAmount += pricing.AcceleratorPrice * 0.05
			}
		}

		opTime := tx.CreatedAt.In(msk)
		if opTime.After(todayStart) || opTime.Equal(todayStart) {
			todayRevenue += tx.Amount
			todayPurchases++
		}
		if opTime.After(weekStart) || opTime.Equal(weekStart) {
			weekRevenue += tx.Amount
		}
		if opTime.After(monthStart) || opTime.Equal(monthStart) {
			monthRevenue += tx.Amount
		}

		dateKey := opTime.Format("2006-01-02")
		if stat, ok := dailyMap[dateKey]; ok {
			stat.Revenue += tx.Amount
			stat.Purchases++
			switch tx.Tier {
			case "vip":
				stat.VIPCount++
			case "basic":
				stat.BasicCount++
			default:
				stat.AcceleratorCount++
			}
		} else if opTime.Before(monthStart) {
			preMonthRevenue += tx.Amount
		}
	}

	// Calculate cumulative revenue running total
	runningCumulative := preMonthRevenue
	var dailyStats []DailyStat
	for _, dk := range orderedDates {
		stat := dailyMap[dk]
		runningCumulative += stat.Revenue
		stat.CumulativeRevenue = runningCumulative
		dailyStats = append(dailyStats, *stat)
	}

	averageOrderValue := 0.0
	if len(txRecords) > 0 {
		averageOrderValue = grossVolume / float64(len(txRecords))
	}

	conversionRate := 0.0
	if totalUsers > 0 {
		conversionRate = (float64(activeStudents) / float64(totalUsers)) * 100.0
	}

	basicPct := 0.0
	accPct := 0.0
	vipPct := 0.0
	if grossVolume > 0 {
		basicPct = (basicRevenue / grossVolume) * 100.0
		accPct = (acceleratorRevenue / grossVolume) * 100.0
		vipPct = (vipRevenue / grossVolume) * 100.0
	}

	c.JSON(http.StatusOK, gin.H{
		"gross_volume":          grossVolume,
		"gross_volume_kopecks":  int64(grossKopecks),
		"active_students":       activeStudents,
		"free_access_count":     freeAccessCount,
		"total_registered":      totalUsers,
		"conversion_rate":       conversionRate,
		"pending_payouts_count": pendingPayoutsCount,
		"pending_payouts_sum":   pendingPayoutsSum,
		"today_revenue":         todayRevenue,
		"today_purchases":       todayPurchases,
		"week_revenue":          weekRevenue,
		"month_revenue":         monthRevenue,
		"average_order_value":   averageOrderValue,
		"promo_usage_count":     promoUsageCount,
		"promo_saved_amount":    promoSavedAmount,
		"tier_breakdown": gin.H{
			"basic": gin.H{
				"count":      basicCount,
				"revenue":    basicRevenue,
				"percentage": basicPct,
			},
			"accelerator": gin.H{
				"count":      acceleratorCount,
				"revenue":    acceleratorRevenue,
				"percentage": accPct,
			},
			"vip": gin.H{
				"count":      vipCount,
				"revenue":    vipRevenue,
				"percentage": vipPct,
			},
		},
		"daily_stats": dailyStats,
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

	// 5. Filter by Self-Employed Receipt Status (чек самозанятого: выписан / не выписан)
	receiptIssued := strings.ToLower(strings.TrimSpace(c.Query("receipt_issued")))
	if receiptIssued == "issued" || receiptIssued == "true" {
		dbQuery = dbQuery.Where("receipt_issued = true")
	} else if receiptIssued == "not_issued" || receiptIssued == "false" || receiptIssued == "pending" {
		dbQuery = dbQuery.Where("receipt_issued = false")
	}

	// 6. Dynamic Sorting (+ purchase date, amount, name, created_at, receipt_issued)
	sortBy := strings.ToLower(strings.TrimSpace(c.DefaultQuery("sort_by", "purchase_date")))
	sortDir := strings.ToLower(strings.TrimSpace(c.DefaultQuery("sort_dir", "desc")))
	if sortDir != "asc" && sortDir != "desc" {
		sortDir = "desc"
	}

	var orderClause string
	switch sortBy {
	case "purchase_date", "access_granted_at", "access_date":
		orderClause = fmt.Sprintf("COALESCE(access_granted_at, created_at) %s, id %s", strings.ToUpper(sortDir), strings.ToUpper(sortDir))
	case "paid_amount", "amount", "revenue":
		orderClause = fmt.Sprintf("paid_amount %s, id %s", strings.ToUpper(sortDir), strings.ToUpper(sortDir))
	case "name", "first_name":
		orderClause = fmt.Sprintf("LOWER(first_name) %s, LOWER(last_name) %s", strings.ToUpper(sortDir), strings.ToUpper(sortDir))
	case "username":
		orderClause = fmt.Sprintf("LOWER(username) %s", strings.ToUpper(sortDir))
	case "tier":
		orderClause = fmt.Sprintf("tier %s, paid_amount DESC", strings.ToUpper(sortDir))
	case "receipt_issued", "fiscal":
		orderClause = fmt.Sprintf("receipt_issued %s, created_at DESC", strings.ToUpper(sortDir))
	case "has_access", "access":
		orderClause = fmt.Sprintf("has_access %s, created_at DESC", strings.ToUpper(sortDir))
	case "created_at", "date":
		orderClause = fmt.Sprintf("created_at %s, id %s", strings.ToUpper(sortDir), strings.ToUpper(sortDir))
	default:
		orderClause = fmt.Sprintf("COALESCE(access_granted_at, created_at) %s, id %s", strings.ToUpper(sortDir), strings.ToUpper(sortDir))
	}

	var total int64
	dbQuery.Count(&total)

	users := make([]models.User, 0)
	dbQuery.Order(orderClause).Offset(offset).Limit(pageSize).Find(&users)

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
	Username      string   `json:"username"`
	FirstName     string   `json:"first_name"`
	LastName      string   `json:"last_name"`
	Role          string   `json:"role"`
	HasAccess     *bool    `json:"has_access"`
	Tier          string   `json:"tier"`
	PaidAmount    *float64 `json:"paid_amount"`
	PromoCode     *string  `json:"promo_code"`
	ReceiptIssued *bool    `json:"receipt_issued"`
	ReceiptURL    *string  `json:"receipt_url"`
	ReceiptNumber *string  `json:"receipt_number"`
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
	if req.ReceiptIssued != nil {
		user.ReceiptIssued = *req.ReceiptIssued
		if *req.ReceiptIssued {
			if user.ReceiptIssuedAt == nil {
				now := time.Now()
				user.ReceiptIssuedAt = &now
			}
		} else {
			user.ReceiptIssuedAt = nil
		}
	}
	if req.ReceiptURL != nil {
		user.ReceiptURL = strings.TrimSpace(*req.ReceiptURL)
	}
	if req.ReceiptNumber != nil {
		user.ReceiptNumber = strings.TrimSpace(*req.ReceiptNumber)
	}
	if req.HasAccess != nil {
		user.HasAccess = *req.HasAccess
		if *req.HasAccess && user.AccessGrantedAt == nil {
			now := time.Now()
			user.AccessGrantedAt = &now
			// Manual access is not a sale. Revenue must only come from
			// payment_transactions, never from an inferred list price.
		}
	}

	if err := h.db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сохранения: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

// ToggleReceiptIssued toggles or explicitly sets whether self-employed receipt was issued (Мой налог)
func (h *AdminHandler) ToggleReceiptIssued(c *gin.Context) {
	var req struct {
		ReceiptIssued *bool `json:"receipt_issued"`
	}
	_ = c.ShouldBindJSON(&req)

	if h.db == nil {
		issued := true
		if req.ReceiptIssued != nil {
			issued = *req.ReceiptIssued
		}
		var now *time.Time
		if issued {
			t := time.Now()
			now = &t
		}
		c.JSON(http.StatusOK, gin.H{
			"success":           true,
			"receipt_issued":    issued,
			"receipt_issued_at": now,
		})
		return
	}

	id := c.Param("id")
	var user models.User
	if err := h.db.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		return
	}

	if req.ReceiptIssued != nil {
		user.ReceiptIssued = *req.ReceiptIssued
	} else {
		user.ReceiptIssued = !user.ReceiptIssued
	}

	if user.ReceiptIssued {
		now := time.Now()
		user.ReceiptIssuedAt = &now
	} else {
		user.ReceiptIssuedAt = nil
	}

	if err := h.db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сохранения: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":           true,
		"receipt_issued":    user.ReceiptIssued,
		"receipt_issued_at": user.ReceiptIssuedAt,
		"user":              user,
	})
}

// IssueReceiptRequest payload for issuing a tax/self-employed receipt to a student
type IssueReceiptRequest struct {
	ReceiptURL    string   `json:"receipt_url"`
	ReceiptNumber string   `json:"receipt_number"`
	Amount        *float64 `json:"amount"`
	ServiceName   string   `json:"service_name"`
	SendTelegram  *bool    `json:"send_telegram"`
}

// IssueReceipt issues an official fiscal receipt to a student and optionally delivers it via Telegram bot
func (h *AdminHandler) IssueReceipt(c *gin.Context) {
	id := c.Param("id")
	var req IssueReceiptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат запроса"})
		return
	}

	cleanURL := strings.TrimSpace(req.ReceiptURL)
	cleanNum := strings.TrimSpace(req.ReceiptNumber)
	serviceName := strings.TrimSpace(req.ServiceName)
	sendTelegram := true
	if req.SendTelegram != nil {
		sendTelegram = *req.SendTelegram
	}

	now := time.Now()

	if h.db == nil {
		amount := 6900.0
		if req.Amount != nil && *req.Amount > 0 {
			amount = *req.Amount
		}
		c.JSON(http.StatusOK, gin.H{
			"success":           true,
			"message":           "Чек успешно выписан и отправлен в Telegram (тестовый режим)",
			"telegram_sent":     sendTelegram,
			"receipt_issued":    true,
			"receipt_issued_at": now,
			"user": gin.H{
				"id":                1,
				"receipt_issued":    true,
				"receipt_issued_at": now,
				"receipt_url":       cleanURL,
				"receipt_number":    cleanNum,
				"paid_amount":       amount,
			},
		})
		return
	}

	var user models.User
	if err := h.db.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		return
	}

	amount := user.PaidAmount
	if req.Amount != nil && *req.Amount > 0 {
		amount = *req.Amount
		user.PaidAmount = amount
	} else if amount <= 0 {
		pricing := models.GetCoursePricing(h.db)
		switch strings.ToLower(user.Tier) {
		case "basic":
			amount = pricing.BasicPrice
		case "vip":
			amount = pricing.VIPPrice
		default:
			amount = pricing.AcceleratorPrice
		}
		user.PaidAmount = amount
	}

	user.ReceiptIssued = true
	user.ReceiptIssuedAt = &now
	user.ReceiptURL = cleanURL
	user.ReceiptNumber = cleanNum

	if err := h.db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сохранения данных пользователя: " + err.Error()})
		return
	}

	telegramSent := false
	var botErr error

	if sendTelegram && user.TelegramID != 0 {
		if h.receiptSender != nil {
			botErr = h.receiptSender(user.TelegramID, user.Tier, amount, cleanURL, cleanNum, serviceName)
			if botErr == nil {
				telegramSent = true
			} else {
				log.Printf("[Admin IssueReceipt] Warning: failed to send receipt via bot to %d: %v", user.TelegramID, botErr)
			}
		} else if h.notifyFunc != nil {
			tierDisplay := "«Акселератор»"
			switch strings.ToLower(user.Tier) {
			case "basic":
				tierDisplay = "«Базовый»"
			case "vip":
				tierDisplay = "«VIP»"
			}
			msg := fmt.Sprintf("🧾 <b>Электронный чек об оплате</b>\n\n"+
				"Поздравляем! Ваш платёж за курс в проекте «так называемый SPARK» (Тариф %s) на сумму %.2f ₽ успешно зарегистрирован.\n", tierDisplay, amount)
			if cleanNum != "" {
				msg += fmt.Sprintf("Номер чека: <code>%s</code>\n", cleanNum)
			}
			if cleanURL != "" {
				msg += fmt.Sprintf("Ссылка на чек: %s\n", cleanURL)
			}
			botErr = h.notifyFunc(user.TelegramID, msg)
			if botErr == nil {
				telegramSent = true
			}
		}
	}

	// Record in fiscal_receipts audit log
	fiscal := models.FiscalReceipt{
		UserID:        user.ID,
		TelegramID:    user.TelegramID,
		Amount:        amount,
		Tier:          user.Tier,
		ReceiptURL:    cleanURL,
		ReceiptNumber: cleanNum,
		ServiceName:   serviceName,
		SentToBot:     telegramSent,
		CreatedAt:     now,
	}
	if telegramSent {
		fiscal.SentAt = &now
	}
	_ = h.db.Create(&fiscal)

	respMsg := "Чек успешно выписан"
	if telegramSent {
		respMsg = "Чек успешно выписан и отправлен ученику в Telegram"
	} else if sendTelegram && botErr != nil {
		respMsg = fmt.Sprintf("Чек сохранён в системе, но бот не смог доставить сообщение (%v)", botErr)
	}

	c.JSON(http.StatusOK, gin.H{
		"success":           true,
		"message":           respMsg,
		"telegram_sent":     telegramSent,
		"receipt_issued":    user.ReceiptIssued,
		"receipt_issued_at": user.ReceiptIssuedAt,
		"receipt_url":       user.ReceiptURL,
		"receipt_number":    user.ReceiptNumber,
		"user":              user,
		"fiscal_record":     fiscal,
	})
}

// EraseUser anonymizes a user and deletes their analytics data while keeping
// financial transactions for accounting.
func (h *AdminHandler) EraseUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}
	if h.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database is unavailable"})
		return
	}
	if err := privacy.EraseUser(h.db, uint(id)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось удалить персональные данные"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Персональные данные удалены, финансовые операции сохранены"})
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
				msg := "🎉 Поздравляем! Ваш доступ к закрытому Telegram-каналу и комьюнити проекта «так называемый SPARK» успешно активирован.\n\nСсылка-приглашение в канал: " + inviteLink + "\n\nЧто дальше:\n1. Зайдите в канал по ссылке.\n2. Иль напишет вам в личку с несколькими вопросами — ответьте, пожалуйста.\n3. Все задания и материалы — в канале."
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
		// Access can be granted manually for free. Do not invent revenue from
		// the current list price; only explicit paid amounts are recorded.
		if req.HasAccess && user.AccessGrantedAt == nil {
			now := time.Now()
			user.AccessGrantedAt = &now
		}
		if err := tx.Save(&user).Error; err != nil {
			return err
		}

		// Record manual paid activation in payment_transactions for unified financial stats
		if req.HasAccess && !wasActive && user.PaidAmount > 0 {
			// Deterministic key prevents a repeated admin click from creating
			// a second financial operation for the same manual payment.
			paymentID := fmt.Sprintf("MANUAL_%d_%s_%d", user.ID, strings.ToLower(user.Tier), int64(user.PaidAmount*100))
			if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "payment_id"}}, DoNothing: true}).Create(&models.PaymentTransaction{
				PaymentID:     paymentID,
				OrderID:       paymentID,
				TelegramID:    user.TelegramID,
				Username:      user.Username,
				Amount:        user.PaidAmount,
				AmountKopecks: int64(money.FromRubles(user.PaidAmount)),
				Currency:      "RUB",
				Tier:          user.Tier,
				Status:        "SUCCESS",
				PayMethod:     "admin_manual",
				PromoCode:     user.PromoCode,
				CreatedAt:     time.Now(),
			}).Error; err != nil {
				return err
			}
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
			msg := "🎉 Поздравляем! Ваш доступ к закрытому Telegram-каналу и комьюнити проекта «так называемый SPARK» успешно активирован.\n\nСсылка-приглашение в канал: " + inviteLink + "\n\nЧто дальше:\n1. Зайдите в канал по ссылке.\n2. Иль напишет вам в личку с несколькими вопросами — ответьте, пожалуйста.\n3. Все задания и материалы — в канале."
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
