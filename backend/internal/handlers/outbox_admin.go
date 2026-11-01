package handlers

import (
	"net/http"
	"time"

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
	// Ignored events keep retry_count = max_retries, so they must be excluded
	// explicitly, otherwise they stay in the queue forever.
	if err := h.db.Where("(status = 'failed' OR retry_count >= max_retries) AND status <> 'ignored'").
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

// IgnoreOutbox marks a dead event as ignored so it leaves the retry queue.
// Only events that already exhausted their retries can be ignored; a live
// event must still be delivered.
func (h *AdminHandler) IgnoreOutbox(c *gin.Context) {
	if h.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database is unavailable"})
		return
	}
	now := time.Now()
	res := h.db.Model(&models.OutboxEvent{}).
		Where("id = ? AND (status = 'failed' OR retry_count >= max_retries)", c.Param("id")).
		Updates(map[string]interface{}{
			"status":       "ignored",
			"locked_until": nil,
			"processed_at": &now,
		})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to ignore event"})
		return
	}
	if res.RowsAffected != 1 {
		c.JSON(http.StatusNotFound, gin.H{"error": "event not found or still deliverable"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
