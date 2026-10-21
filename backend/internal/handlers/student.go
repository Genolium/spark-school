package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spark-school/backend/internal/auth"
	"github.com/spark-school/backend/internal/config"
	"github.com/spark-school/backend/internal/models"
	"gorm.io/gorm"
)

type StudentHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewStudentHandler(db *gorm.DB, cfg *config.Config) *StudentHandler {
	return &StudentHandler{db: db, cfg: cfg}
}

type miniAppUser struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	PhotoURL  string `json:"photo_url"`
}

// TelegramAuth verifies Mini App initData and issues the same student JWT used
// by the web API. This is the entry point for a future student Mini App.
func (h *StudentHandler) TelegramAuth(c *gin.Context) {
	var req struct {
		InitData string `json:"init_data" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "init_data is required"})
		return
	}
	if h.cfg == nil || strings.TrimSpace(h.cfg.TelegramBotToken) == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "telegram auth is not configured"})
		return
	}
	values, ok := auth.VerifyTelegramWebApp(req.InitData, h.cfg.TelegramBotToken)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid telegram init data"})
		return
	}
	var tgUser miniAppUser
	if raw := values["user"]; raw != "" {
		if err := json.Unmarshal([]byte(raw), &tgUser); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid telegram user"})
			return
		}
	}
	if tgUser.ID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "telegram user is missing"})
		return
	}
	if h.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database is unavailable"})
		return
	}

	var user models.User
	err := h.db.Where("telegram_id = ?", tgUser.ID).First(&user).Error
	if err == gorm.ErrRecordNotFound {
		user = models.User{
			TelegramID: tgUser.ID,
			Username:   strings.TrimPrefix(tgUser.Username, "@"),
			FirstName:  strings.TrimSpace(tgUser.FirstName),
			LastName:   strings.TrimSpace(tgUser.LastName),
			PhotoURL:   strings.TrimSpace(tgUser.PhotoURL),
			Role:       "student",
			Tier:       "accelerator",
			CreatedAt:  time.Now(),
		}
		if user.FirstName == "" {
			user.FirstName = "Студент"
		}
		if err := h.db.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create student"})
			return
		}
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	token, err := auth.GenerateJWT(&user, h.cfg.JWTSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token, "user": studentProfile(user)})
}

// Me returns the authenticated student's current access state.
func (h *StudentHandler) Me(c *gin.Context) {
	user, ok := h.currentUser(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, studentProfile(*user))
}

// Orders returns the student's payment history without internal gateway data.
func (h *StudentHandler) Orders(c *gin.Context) {
	user, ok := h.currentUser(c)
	if !ok {
		return
	}
	var payments []models.PaymentTransaction
	if err := h.db.Where("telegram_id = ?", user.TelegramID).
		Order("created_at DESC").Limit(100).Find(&payments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load orders"})
		return
	}
	type order struct {
		PaymentID string    `json:"payment_id"`
		Tier      string    `json:"tier"`
		Amount    float64   `json:"amount"`
		Status    string    `json:"status"`
		CreatedAt time.Time `json:"created_at"`
	}
	result := make([]order, 0, len(payments))
	for _, p := range payments {
		result = append(result, order{
			PaymentID: p.PaymentID,
			Tier:      p.Tier,
			Amount:    p.Amount,
			Status:    p.Status,
			CreatedAt: p.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"orders": result})
}

func (h *StudentHandler) currentUser(c *gin.Context) (*models.User, bool) {
	userID := c.GetUint("user_id")
	if userID == 0 || h.db == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return nil, false
	}
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return nil, false
	}
	return &user, true
}

func studentProfile(user models.User) gin.H {
	return gin.H{
		"id":                user.ID,
		"telegram_id":       user.TelegramID,
		"username":          user.Username,
		"first_name":        user.FirstName,
		"last_name":         user.LastName,
		"photo_url":         user.PhotoURL,
		"has_access":        user.HasAccess,
		"tier":              user.Tier,
		"paid_amount":       user.PaidAmount,
		"audit_credits":     user.AuditCredits,
		"created_at":        user.CreatedAt,
		"access_granted_at": user.AccessGrantedAt,
	}
}
