package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/spark-school/backend/internal/models"
	"gorm.io/gorm"
)

type PromoHandler struct {
	db         *gorm.DB
	redeemFunc func(code string) (*models.PromoCode, error)
}

func NewPromoHandler(db *gorm.DB) *PromoHandler {
	return &PromoHandler{db: db}
}

// WithCustomRedeem sets a custom redemption function (useful for unit testing without a live db).
func (h *PromoHandler) WithCustomRedeem(fn func(code string) (*models.PromoCode, error)) *PromoHandler {
	h.redeemFunc = fn
	return h
}

type ValidatePromoRequest struct {
	Code  string  `json:"code" binding:"required"`
	Tier  string  `json:"tier"`
	Price float64 `json:"price"`
}

type ValidatePromoResponse struct {
	Valid           bool    `json:"valid"`
	Code            string  `json:"code"`
	DiscountPercent int     `json:"discount_percent"`
	OriginalPrice   float64 `json:"original_price"`
	DiscountAmount  float64 `json:"discount_amount"`
	FinalPrice      float64 `json:"final_price"`
	OwnerUsername   string  `json:"owner_username,omitempty"`
	Message         string  `json:"message,omitempty"`
}

const BaseCoursePrice = 6900.00

// GetTierPrice returns standard price for the selected tier
func GetTierPrice(tier string, fallbackPrice float64) float64 {
	if fallbackPrice > 0 {
		return fallbackPrice
	}
	switch strings.ToLower(strings.TrimSpace(tier)) {
	case "vip":
		return 14900.00
	case "accelerator", "акселератор":
		return 6900.00
	default:
		return BaseCoursePrice
	}
}

// Validate checks whether a promo code is valid and returns the discounted price.
func (h *PromoHandler) Validate(c *gin.Context) {
	var req ValidatePromoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Поле code обязательно для заполнения"})
		return
	}

	targetPrice := GetTierPrice(req.Tier, req.Price)

	cleanCode := strings.ToUpper(strings.TrimSpace(req.Code))
	if cleanCode == "" {
		c.JSON(http.StatusOK, ValidatePromoResponse{
			Valid:         false,
			OriginalPrice: targetPrice,
			FinalPrice:    targetPrice,
			Message:       "Промокод не указан",
		})
		return
	}

	var promo models.PromoCode
	err := h.db.Where("UPPER(code) = ? AND is_active = ?", cleanCode, true).First(&promo).Error
	if err != nil {
		c.JSON(http.StatusOK, ValidatePromoResponse{
			Valid:         false,
			Code:          cleanCode,
			OriginalPrice: targetPrice,
			FinalPrice:    targetPrice,
			Message:       "Промокод не найден или срок его действия истек",
		})
		return
	}

	discountPercent := promo.DiscountPercent
	if discountPercent <= 0 {
		discountPercent = 5
	}

	discountAmount := (targetPrice * float64(discountPercent)) / 100.0
	finalPrice := targetPrice - discountAmount

	c.JSON(http.StatusOK, ValidatePromoResponse{
		Valid:           true,
		Code:            promo.Code,
		DiscountPercent: discountPercent,
		OriginalPrice:   targetPrice,
		DiscountAmount:  discountAmount,
		FinalPrice:      finalPrice,
		OwnerUsername:   promo.OwnerUsername,
		Message:         "Промокод успешно применён!",
	})
}

type CreatePromoRequest struct {
	Code            string  `json:"code" binding:"required"`
	OwnerTelegramID int64   `json:"owner_telegram_id"`
	OwnerUsername   string  `json:"owner_username"`
	DiscountPercent int     `json:"discount_percent"`
	RewardAmount    float64 `json:"reward_amount"`
}

// Create generates a new promo code (Admin or Bot partner hook).
func (h *PromoHandler) Create(c *gin.Context) {
	var req CreatePromoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cleanCode := strings.ToUpper(strings.TrimSpace(req.Code))
	if cleanCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Код промокода не может быть пустым"})
		return
	}

	// Check if already exists
	var existing models.PromoCode
	if err := h.db.Where("UPPER(code) = ?", cleanCode).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Такой промокод уже существует"})
		return
	}

	discount := req.DiscountPercent
	if discount <= 0 {
		discount = 5
	}
	reward := req.RewardAmount
	if reward <= 0 {
		reward = 1000.00
	}

	promo := models.PromoCode{
		Code:            cleanCode,
		DiscountPercent: discount,
		OwnerTelegramID: req.OwnerTelegramID,
		OwnerUsername:   strings.TrimPrefix(req.OwnerUsername, "@"),
		RewardAmount:    reward,
		IsActive:        true,
	}

	if err := h.db.Create(&promo).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось создать промокод: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, promo)
}

// List returns all registered promo codes for the admin panel.
func (h *PromoHandler) List(c *gin.Context) {
	var promos []models.PromoCode
	if err := h.db.Order("created_at desc").Find(&promos).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения промокодов"})
		return
	}
	c.JSON(http.StatusOK, promos)
}

// ToggleStatus activates or deactivates a promo code.
func (h *PromoHandler) ToggleStatus(c *gin.Context) {
	id := c.Param("id")
	var promo models.PromoCode
	if err := h.db.First(&promo, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Промокод не найден"})
		return
	}

	promo.IsActive = !promo.IsActive
	if err := h.db.Save(&promo).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка обновления статуса"})
		return
	}

	c.JSON(http.StatusOK, promo)
}

type UpdatePromoRequest struct {
	Code            string   `json:"code"`
	OwnerTelegramID *int64   `json:"owner_telegram_id"`
	OwnerUsername   *string  `json:"owner_username"`
	DiscountPercent *int     `json:"discount_percent"`
	RewardAmount    *float64 `json:"reward_amount"`
	IsActive        *bool    `json:"is_active"`
}

// Update modifies an existing promo code.
func (h *PromoHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var promo models.PromoCode
	if err := h.db.First(&promo, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Промокод не найден"})
		return
	}

	var req UpdatePromoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат запроса: " + err.Error()})
		return
	}

	if req.Code != "" {
		cleanCode := strings.ToUpper(strings.TrimSpace(req.Code))
		var existing models.PromoCode
		if err := h.db.Where("UPPER(code) = ? AND id != ?", cleanCode, promo.ID).First(&existing).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Промокод с таким названием уже существует"})
			return
		}
		promo.Code = cleanCode
	}

	if req.OwnerTelegramID != nil {
		promo.OwnerTelegramID = *req.OwnerTelegramID
	}
	if req.OwnerUsername != nil {
		promo.OwnerUsername = strings.TrimPrefix(*req.OwnerUsername, "@")
	}
	if req.DiscountPercent != nil && *req.DiscountPercent > 0 {
		promo.DiscountPercent = *req.DiscountPercent
	}
	if req.RewardAmount != nil && *req.RewardAmount >= 0 {
		promo.RewardAmount = *req.RewardAmount
	}
	if req.IsActive != nil {
		promo.IsActive = *req.IsActive
	}

	if err := h.db.Save(&promo).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сохранения промокода: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, promo)
}

// Delete removes a promo code.
func (h *PromoHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&models.PromoCode{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка удаления: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Промокод удалён"})
}

// RedeemPromoCode atomically redeems a promo code and increments uses_count in a single SQL statement.
// It eliminates lost updates under concurrent transactions and verifies code existence and active status.
func RedeemPromoCode(tx *gorm.DB, code string) (*models.PromoCode, error) {
	cleanCode := strings.ToUpper(strings.TrimSpace(code))
	if cleanCode == "" {
		return nil, errors.New("empty promo code")
	}
	if tx == nil {
		return nil, errors.New("database transaction is not initialized")
	}

	// Atomic SQL increment: eliminates lost updates under concurrent transactions
	res := tx.Model(&models.PromoCode{}).
		Where("UPPER(code) = ? AND is_active = ?", cleanCode, true).
		Update("uses_count", gorm.Expr("uses_count + 1"))

	if res.Error != nil {
		return nil, fmt.Errorf("failed to increment promo code uses: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return nil, errors.New("promo code not found or inactive")
	}

	var updated models.PromoCode
	if err := tx.Where("UPPER(code) = ?", cleanCode).First(&updated).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch updated promo code: %w", err)
	}
	return &updated, nil
}

type RedeemPromoRequest struct {
	Code string `json:"code" binding:"required"`
}

type RedeemPromoResponse struct {
	Success         bool    `json:"success"`
	Code            string  `json:"code"`
	UsesCount       int     `json:"uses_count"`
	DiscountPercent int     `json:"discount_percent"`
	OwnerUsername   string  `json:"owner_username,omitempty"`
	Message         string  `json:"message,omitempty"`
}

// Redeem handles public promo code redemption requests atomically.
// POST /api/v1/promo/redeem
func (h *PromoHandler) Redeem(c *gin.Context) {
	var req RedeemPromoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Поле code обязательно для заполнения"})
		return
	}

	cleanCode := strings.ToUpper(strings.TrimSpace(req.Code))
	if cleanCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Поле code обязательно для заполнения"})
		return
	}

	var promo *models.PromoCode
	var err error
	if h.redeemFunc != nil {
		promo, err = h.redeemFunc(cleanCode)
	} else {
		if h.db == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "База данных не инициализирована"})
			return
		}
		promo, err = RedeemPromoCode(h.db, cleanCode)
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, RedeemPromoResponse{
		Success:         true,
		Code:            promo.Code,
		UsesCount:       promo.UsesCount,
		DiscountPercent: promo.DiscountPercent,
		OwnerUsername:   promo.OwnerUsername,
		Message:         "Промокод успешно применён!",
	})
}
