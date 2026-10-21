package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/spark-school/backend/internal/models"
)

// ListDeadOutbox returns events that exhausted retries or are marked failed.
func (h *AdminHandler) ListDeadOutbox(c *gin.Context) {
	if h.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database is unavailable"})
		return
	}
	var events []models.OutboxEvent
	if err := h.db.Where("status = 'failed' OR retry_count >= max_retries").
		Order("created_at ASC").Limit(200).Find(&events).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read outbox"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"events": events})
}

// RetryOutbox resets a dead event so the dispatcher can try again.
func (h *AdminHandler) RetryOutbox(c *gin.Context) {
	if h.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database is unavailable"})
		return
	}
	res := h.db.Model(&models.OutboxEvent{}).
		Where("id = ?", c.Param("id")).
		Updates(map[string]interface{}{
			"status":       "pending",
			"retry_count":  0,
			"last_error":   "",
			"available_at": nil,
			"locked_until": nil,
			"processed_at": nil,
		})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retry event"})
		return
	}
	if res.RowsAffected != 1 {
		c.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
