// Package readmodels maintains pre-aggregated tables for the admin dashboard.
package readmodels

import (
	"context"
	"log"
	"sync"
	"time"

	"gorm.io/gorm"
)

type RevenueDaily struct {
	Day            time.Time `gorm:"column:day" json:"day"`
	Tier           string    `gorm:"column:tier" json:"tier"`
	Source         string    `gorm:"column:source" json:"source"`
	PromoCode      string    `gorm:"column:promo_code" json:"promo_code"`
	Purchases      int64     `gorm:"column:purchases" json:"purchases"`
	RevenueKopecks int64     `gorm:"column:revenue_kopecks" json:"revenue_kopecks"`
	UpdatedAt      time.Time `gorm:"column:updated_at" json:"updated_at"`
}

type RevenueWorker struct {
	db       *gorm.DB
	interval time.Duration
	stop     chan struct{}
	wg       sync.WaitGroup
}

func NewRevenueWorker(db *gorm.DB) *RevenueWorker {
	return &RevenueWorker{db: db, interval: time.Minute, stop: make(chan struct{})}
}

func (w *RevenueWorker) Start() {
	if w == nil || w.db == nil {
		return
	}
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		if err := w.Refresh(context.Background()); err != nil {
			log.Printf("[ReadModels] initial revenue refresh failed: %v", err)
		}
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()
		for {
			select {
			case <-w.stop:
				return
			case <-ticker.C:
				if err := w.Refresh(context.Background()); err != nil {
					log.Printf("[ReadModels] revenue refresh failed: %v", err)
				}
			}
		}
	}()
}

func (w *RevenueWorker) Stop() {
	if w == nil {
		return
	}
	select {
	case <-w.stop:
	default:
		close(w.stop)
	}
	w.wg.Wait()
}

// Refresh rebuilds the daily revenue rollup used by admin stats.
func (w *RevenueWorker) Refresh(ctx context.Context) error {
	if w == nil || w.db == nil {
		return nil
	}
	return w.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM admin_revenue_daily").Error; err != nil {
			return err
		}
		return tx.Exec(`
			INSERT INTO admin_revenue_daily (day, tier, source, promo_code, purchases, revenue_kopecks, updated_at)
			SELECT (created_at AT TIME ZONE 'Europe/Moscow')::date,
			       COALESCE(NULLIF(LOWER(tier), ''), 'accelerator'),
			       COALESCE(source, ''),
			       COALESCE(promo_code, ''),
			       COUNT(*),
			       COALESCE(SUM(amount_kopecks), 0),
			       NOW()
			FROM payment_transactions
			WHERE status = 'SUCCESS'
			GROUP BY 1, 2, 3, 4
		`).Error
	})
}
