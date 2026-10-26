// Package privacy implements analytics retention and personal-data erasure.
package privacy

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/spark-school/backend/internal/database"
	"github.com/spark-school/backend/internal/models"
	"gorm.io/gorm"
)

// Cleaner removes analytics data older than the configured retention period.
type Cleaner struct {
	db       *gorm.DB
	days     int
	interval time.Duration
	stop     chan struct{}
	wg       sync.WaitGroup
}

func NewCleaner(db *gorm.DB, days int) *Cleaner {
	if days <= 0 {
		days = 180
	}
	return &Cleaner{db: db, days: days, interval: 24 * time.Hour, stop: make(chan struct{})}
}

func (c *Cleaner) Start() {
	if c == nil || c.db == nil {
		return
	}
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		// Run once at startup, then daily.
		c.RunOnce(context.Background())
		ticker := time.NewTicker(c.interval)
		defer ticker.Stop()
		for {
			select {
			case <-c.stop:
				return
			case <-ticker.C:
				c.RunOnce(context.Background())
			}
		}
	}()
}

func (c *Cleaner) Stop() {
	if c == nil {
		return
	}
	select {
	case <-c.stop:
	default:
		close(c.stop)
	}
	c.wg.Wait()
}

// RunOnce deletes analytics events, experiment assignments and campaign link
// clicks older than the retention window. Financial tables are never touched.
func (c *Cleaner) RunOnce(ctx context.Context) (int64, error) {
	if c == nil || c.db == nil {
		return 0, nil
	}
	cutoff := time.Now().AddDate(0, 0, -c.days)
	var deleted int64
	res := c.db.WithContext(ctx).Where("created_at < ?", cutoff).Delete(&models.AnalyticsEvent{})
	if res.Error != nil {
		return 0, res.Error
	}
	deleted += res.RowsAffected
	res = c.db.WithContext(ctx).Where("created_at < ?", cutoff).Delete(&models.ExperimentAssignment{})
	if res.Error != nil {
		return deleted, res.Error
	}
	deleted += res.RowsAffected
	res = c.db.WithContext(ctx).Where("created_at < ?", cutoff).Delete(&models.CampaignLinkClick{})
	if res.Error != nil {
		return deleted, res.Error
	}
	deleted += res.RowsAffected
	// Operational tables: expired leases/tokens/sessions and old rate-limit rows.
	cutoff24 := time.Now().Add(-24 * time.Hour)
	_ = c.db.WithContext(ctx).Where("window_start < ?", cutoff24).Delete(&models.RateLimit{}).Error
	_ = c.db.WithContext(ctx).Where("expires_at < ?", time.Now()).Delete(&models.RevokedToken{}).Error
	_ = c.db.WithContext(ctx).Where("expires_at < ?", time.Now()).Delete(&models.BotSession{}).Error
	_ = c.db.WithContext(ctx).Where("expires_at < ?", time.Now()).Delete(&models.SchedulerLease{}).Error
	if err := database.EnsureAnalyticsPartitions(c.db, time.Now()); err != nil {
		log.Printf("[Privacy] failed to ensure analytics partitions: %v", err)
	}
	if deleted > 0 {
		log.Printf("[Privacy] removed %d analytics/click rows older than %d days", deleted, c.days)
	}
	return deleted, nil
}

// EraseUser removes analytics, experiment assignments and campaign link clicks
// linked to a user and anonymizes personal profile fields. Financial
// transactions and their amounts are intentionally preserved for accounting;
// only the identity is removed.
func EraseUser(db *gorm.DB, userID uint) error {
	if db == nil {
		return gorm.ErrInvalidDB
	}
	return db.Transaction(func(tx *gorm.DB) error {
		var user models.User
		if err := tx.First(&user, userID).Error; err != nil {
			return err
		}
		if user.VisitorID != "" {
			if err := tx.Where("visitor_id = ?", user.VisitorID).Delete(&models.AnalyticsEvent{}).Error; err != nil {
				return err
			}
			if err := tx.Where("visitor_id = ?", user.VisitorID).Delete(&models.ExperimentAssignment{}).Error; err != nil {
				return err
			}
			if err := tx.Where("visitor_id = ?", user.VisitorID).Delete(&models.CampaignLinkClick{}).Error; err != nil {
				return err
			}
		}
		if user.AttributionToken != "" {
			if err := tx.Where("attribution_token = ?", user.AttributionToken).Delete(&models.AnalyticsEvent{}).Error; err != nil {
				return err
			}
			if err := tx.Where("attribution_token = ?", user.AttributionToken).Delete(&models.ExperimentAssignment{}).Error; err != nil {
				return err
			}
			if err := tx.Where("attribution_token = ?", user.AttributionToken).Delete(&models.CampaignLinkClick{}).Error; err != nil {
				return err
			}
		}
		now := time.Now()
		if err := tx.Model(&models.User{}).Where("id = ?", user.ID).Updates(map[string]interface{}{
			"username":          "",
			"first_name":        "Удалённый пользователь",
			"last_name":         "",
			"photo_url":         "",
			"attribution_token": "",
			"visitor_id":        "",
			"reminders_opt_out": true,
			"anonymized_at":     &now,
		}).Error; err != nil {
			return err
		}
		_ = tx.Model(&models.DripLead{}).Where("telegram_id = ?", user.TelegramID).
			Updates(map[string]interface{}{"stage": -1, "is_completed": true}).Error
		return nil
	})
}
