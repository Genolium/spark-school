package handlers

import (
	"math"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/spark-school/backend/internal/models"
	"gorm.io/gorm"
)

type PricingHandler struct {
	db *gorm.DB
}

func NewPricingHandler(db *gorm.DB) *PricingHandler {
	return &PricingHandler{db: db}
}

// GetPricing returns the current course pricing (public & admin).
// GET /api/v1/pricing
// GET /api/v1/admin/pricing
func (h *PricingHandler) GetPricing(c *gin.Context) {
	pricing := models.GetCoursePricing(h.db)
	c.JSON(http.StatusOK, pricing)
}

// UpdatePricingRequest validates payload for changing course pricing.
type UpdatePricingRequest struct {
	BasicPrice          float64 `json:"basic_price"`
	BasicOldPrice       float64 `json:"basic_old_price"`
	AcceleratorPrice    float64 `json:"accelerator_price" binding:"required"`
	AcceleratorOldPrice float64 `json:"accelerator_old_price"`
	VIPPrice            float64 `json:"vip_price" binding:"required"`
	VIPOldPrice         float64 `json:"vip_old_price"`
	BankName            string  `json:"bank_name"`
}

// UpdatePricing updates course pricing in DB (admin only).
// PUT /api/v1/admin/pricing
func (h *PricingHandler) UpdatePricing(c *gin.Context) {
	var req UpdatePricingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат данных: " + err.Error()})
		return
	}

	if req.BasicPrice <= 0 {
		req.BasicPrice = 3990.00
	}

	if req.AcceleratorPrice <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Цена тарифа «Акселератор» должна быть положительным числом"})
		return
	}
	if req.VIPPrice <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Цена тарифа «VIP» должна быть положительным числом"})
		return
	}
	if req.AcceleratorPrice < req.BasicPrice {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Цена тарифа «Акселератор» не может быть ниже тарифа «Базовый»"})
		return
	}
	if req.VIPPrice < req.AcceleratorPrice {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Цена тарифа «VIP» не может быть ниже тарифа «Акселератор»"})
		return
	}

	// Ensure old strike-through prices are always dynamically higher than current price
	if req.BasicOldPrice <= req.BasicPrice {
		req.BasicOldPrice = math.Round((req.BasicPrice*1.4)/100) * 100
	}
	if req.AcceleratorOldPrice <= req.AcceleratorPrice {
		req.AcceleratorOldPrice = math.Round((req.AcceleratorPrice*1.4)/100) * 100
	}
	if req.VIPOldPrice <= req.VIPPrice {
		req.VIPOldPrice = math.Round((req.VIPPrice*1.4)/100) * 100
	}

	bankName := strings.TrimSpace(req.BankName)
	if bankName == "" {
		bankName = "БСПБ (Банк «Санкт-Петербург»)"
	}

	newPricing := models.CoursePricing{
		BasicPrice:          req.BasicPrice,
		BasicOldPrice:       req.BasicOldPrice,
		AcceleratorPrice:    req.AcceleratorPrice,
		AcceleratorOldPrice: req.AcceleratorOldPrice,
		VIPPrice:            req.VIPPrice,
		VIPOldPrice:         req.VIPOldPrice,
		BankName:            bankName,
	}

	updated, err := models.UpdateCoursePricing(h.db, newPricing)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось обновить цены: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, updated)
}
