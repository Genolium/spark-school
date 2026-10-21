package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spark-school/backend/internal/config"
	"github.com/spark-school/backend/internal/diagnostics"
	"github.com/spark-school/backend/internal/models"
	"gorm.io/gorm"
)

type DiagnosticHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewDiagnosticHandler(db *gorm.DB, cfg *config.Config) *DiagnosticHandler {
	return &DiagnosticHandler{db: db, cfg: cfg}
}

type createDiagnosticRequest struct {
	VisitorID        string `json:"visitor_id"`
	AttributionToken string `json:"attribution_token"`
	AgeCitizenship   bool   `json:"age_citizenship"`
	UniversityYear   string `json:"university_year"`
	EnglishLevel     string `json:"english_level"`
	Source           string `json:"source"`
}

// Create stores a completed diagnostic and returns a signed public token plus
// a personalized plan. The token is opaque and contains no personal data.
func (h *DiagnosticHandler) Create(c *gin.Context) {
	var req createDiagnosticRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid diagnostic payload"})
		return
	}
	level := strings.ToLower(strings.TrimSpace(req.EnglishLevel))
	if level != "b1" && level != "b2" && level != "c1" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "english_level must be b1, b2 or c1"})
		return
	}
	year := strings.ToLower(strings.TrimSpace(req.UniversityYear))
	if year != "1-2" && year != "3" && year != "senior" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown university_year"})
		return
	}
	if h.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database is unavailable"})
		return
	}

	tokenBytes := make([]byte, 12)
	if _, err := rand.Read(tokenBytes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot create plan token"})
		return
	}
	token := hex.EncodeToString(tokenBytes)
	record := models.Diagnostic{
		Token:            token,
		VisitorID:        sanitizeToken(req.VisitorID, 64),
		AttributionToken: sanitizeToken(req.AttributionToken, 64),
		AgeCitizenship:   req.AgeCitizenship,
		UniversityYear:   year,
		EnglishLevel:     level,
		Score:            diagnostics.ScoreFor(level),
		Source:           sanitizeToken(req.Source, 64),
		CreatedAt:        time.Now(),
	}
	if err := h.db.Create(&record).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot save diagnostic"})
		return
	}
	_ = h.db.Create(&models.AnalyticsEvent{
		VisitorID:        record.VisitorID,
		AttributionToken: record.AttributionToken,
		Step:             "diagnostic_plan_created",
		Source:           record.Source,
		Metadata:         `{"diagnostic":true}`,
		CreatedAt:        time.Now(),
	}).Error

	plan := diagnostics.Generate(req.AgeCitizenship, year, level)
	c.JSON(http.StatusOK, gin.H{
		"token":        token,
		"plan":         plan,
		"telegram_url": h.telegramURL(token),
		"plan_url":     "/plan/" + token,
	})
}

// Get returns a plan by its opaque token.
func (h *DiagnosticHandler) telegramURL(token string) string {
	bot := "spark_prep_bot"
	if h.cfg != nil && strings.TrimSpace(h.cfg.TelegramBotUsername) != "" {
		bot = strings.TrimPrefix(strings.TrimSpace(h.cfg.TelegramBotUsername), "@")
	}
	return "https://t.me/" + bot + "?start=plan_" + token
}

func (h *DiagnosticHandler) Get(c *gin.Context) {
	if h.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database is unavailable"})
		return
	}
	var record models.Diagnostic
	if err := h.db.Where("token = ?", c.Param("token")).First(&record).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
		return
	}
	plan := diagnostics.Generate(record.AgeCitizenship, record.UniversityYear, record.EnglishLevel)
	c.JSON(http.StatusOK, gin.H{
		"token":      record.Token,
		"plan":       plan,
		"linked":     record.TelegramID != 0,
		"created_at": record.CreatedAt,
	})
}
