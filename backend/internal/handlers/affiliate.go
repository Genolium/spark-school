package handlers

import (
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spark-school/backend/internal/config"
	"github.com/spark-school/backend/internal/models"
	"github.com/spark-school/backend/internal/services"
	"gorm.io/gorm"
)

type AffiliateHandler struct {
	db               *gorm.DB
	cfg              *config.Config
	affiliateService *services.AffiliateService
}

func NewAffiliateHandler(db *gorm.DB, cfg *config.Config, aff *services.AffiliateService) *AffiliateHandler {
	return &AffiliateHandler{db: db, cfg: cfg, affiliateService: aff}
}

// GetStats returns partner profile, referral counts, balance, and transaction history.
func (h *AffiliateHandler) GetStats(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)

	var profile models.AffiliateProfile
	if err := h.db.Where("user_id = ?", userID).First(&profile).Error; err != nil {
		var user models.User
		if errUser := h.db.First(&user, userID).Error; errUser != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		cleanUsername := strings.ToLower(strings.TrimPrefix(user.Username, "@"))
		if cleanUsername == "" {
			cleanUsername = fmt.Sprintf("id%d", user.TelegramID)
		}
		finalCode := cleanUsername
		var existingCode models.AffiliateProfile
		if errCode := h.db.Where("referral_code = ?", finalCode).First(&existingCode).Error; errCode == nil && existingCode.UserID != user.ID {
			finalCode = fmt.Sprintf("%s_%d", cleanUsername, user.ID)
		}

		profile = models.AffiliateProfile{
			UserID:       user.ID,
			ReferralCode: finalCode,
			CreatedAt:    time.Now(),
		}
		if errCreate := h.db.Create(&profile).Error; errCreate != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create affiliate profile"})
			return
		}
	}

	var l1Count int64
	h.db.Model(&models.Referral{}).Where("referred_by_id = ?", userID).Count(&l1Count)

	var l2Count int64
	h.db.Model(&models.Referral{}).Where("parent_referrer_id = ?", userID).Count(&l2Count)

	var transactions []models.ReferralTransaction
	h.db.Where("affiliate_id = ?", userID).Order("created_at DESC").Limit(20).Find(&transactions)

	var payouts []models.PayoutRequest
	h.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&payouts)

	domain := h.cfg.Domain
	if domain == "" {
		domain = "sparkprep.ru"
	}
	refLink := fmt.Sprintf("https://%s/?ref=%s", domain, profile.ReferralCode)

	c.JSON(http.StatusOK, gin.H{
		"referral_code":    profile.ReferralCode,
		"referral_link":    refLink,
		"current_balance":  profile.CurrentBalance,
		"total_earned":     profile.TotalEarned,
		"total_withdrawn":  profile.TotalWithdrawn,
		"level1_referrals": l1Count,
		"level2_referrals": l2Count,
		"total_referrals":  l1Count + l2Count,
		"transactions":     transactions,
		"payouts":          payouts,
	})
}

// SubmitPayout handles withdrawal requests from partners.
func (h *AffiliateHandler) SubmitPayout(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)

	var req struct {
		Amount  float64 `json:"amount" binding:"required"`
		Details string  `json:"details" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Необходимо указать сумму и реквизиты для вывода"})
		return
	}

	cleanDetails := strings.TrimSpace(req.Details)
	cleanDetails = html.EscapeString(cleanDetails)

	if len(cleanDetails) < 4 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Укажите корректные реквизиты для перевода (номер карты или телефон СБП)"})
		return
	}
	if len(cleanDetails) > 255 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Реквизиты превышают допустимую длину (максимум 255 символов)"})
		return
	}
	if req.Amount <= 0 || req.Amount > 500000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Недопустимая сумма вывода (лимит: от 1 000 ₽ до 500 000 ₽)"})
		return
	}

	payout, err := h.affiliateService.RequestPayout(userID, req.Amount, cleanDetails)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Заявка на вывод успешно принята и отправлена на модерацию",
		"payout":  payout,
	})
}
