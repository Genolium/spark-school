package security

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spark-school/backend/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// DatabaseRateLimitMiddleware stores counters in PostgreSQL so the limit is
// shared by every backend instance. Use it for security-sensitive endpoints
// (admin login, auth) where per-instance limits are not acceptable.
func DatabaseRateLimitMiddleware(db *gorm.DB, limit int, window time.Duration, keyPrefix, customMsg string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if ip == "" {
			ip = "unknown"
		}
		allowed, retry := allowDatabase(db, keyPrefix+":"+ip, limit, window)
		if !allowed {
			seconds := int(retry.Seconds())
			if seconds < 1 {
				seconds = 1
			}
			c.Header("Retry-After", strconv.Itoa(seconds))
			msg := customMsg
			if msg == "" {
				msg = "Превышен лимит запросов. Пожалуйста, подождите."
			}
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":               msg,
				"retry_after_seconds": seconds,
			})
			return
		}
		c.Next()
	}
}

func allowDatabase(db *gorm.DB, key string, limit int, window time.Duration) (bool, time.Duration) {
	if db == nil {
		return true, 0
	}
	now := time.Now()
	allowed := true
	var retry time.Duration

	err := db.Transaction(func(tx *gorm.DB) error {
		var record models.RateLimit
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("key = ?", key).First(&record).Error
		if err == gorm.ErrRecordNotFound {
			record = models.RateLimit{Key: key, WindowStart: now, Count: 1}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&record).Error; err != nil {
				return err
			}
			return nil
		}
		if err != nil {
			return err
		}
		if now.Sub(record.WindowStart) >= window {
			record.WindowStart = now
			record.Count = 1
		} else if record.Count >= limit {
			allowed = false
			retry = window - now.Sub(record.WindowStart)
			return nil
		} else {
			record.Count++
		}
		return tx.Model(&models.RateLimit{}).Where("key = ?", key).Updates(map[string]interface{}{
			"window_start": record.WindowStart,
			"count":        record.Count,
		}).Error
	})
	if err != nil {
		// Fail open on a database outage would defeat the limiter; fail closed
		// for the sensitive endpoints that use it.
		return false, window
	}
	return allowed, retry
}
