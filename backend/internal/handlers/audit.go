package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spark-school/backend/internal/models"
	"gorm.io/gorm"
)

// AdminAuditMiddleware records every privileged mutation. GET requests are
// skipped to keep the log focused on state changes.
func AdminAuditMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if db == nil {
			return
		}
		switch c.Request.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		default:
			return
		}
		requestID := make([]byte, 8)
		_, _ = rand.Read(requestID)
		entry := models.AdminAuditLog{
			AdminUsername:   c.GetString("username"),
			AdminTelegramID: c.GetInt64("telegram_id"),
			Method:          c.Request.Method,
			Path:            c.FullPath(),
			Status:          c.Writer.Status(),
			IP:              c.ClientIP(),
			RequestID:       hex.EncodeToString(requestID),
			CreatedAt:       time.Now(),
		}
		_ = db.Create(&entry).Error
	}
}

// ListAuditLog returns the latest privileged actions.
func (h *AdminHandler) ListAuditLog(c *gin.Context) {
	if h.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database is unavailable"})
		return
	}
	var entries []models.AdminAuditLog
	if err := h.db.Order("created_at DESC").Limit(200).Find(&entries).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read audit log"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"entries": entries})
}
