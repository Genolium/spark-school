// Package monitoring checks operational invariants and alerts the admin in
// Telegram. Alerts have a cooldown so a persistent failure does not spam.
package monitoring

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/spark-school/backend/internal/database"
	"github.com/spark-school/backend/internal/models"
	"github.com/spark-school/backend/internal/security"
	"gorm.io/gorm"
)

type Monitor struct {
	db       *gorm.DB
	adminID  int64
	notify   func(telegramID int64, message string) error
	metrics  *security.Metrics
	ready    func(ctx context.Context) error
	interval time.Duration
	cooldown time.Duration
	stop     chan struct{}
	wg       sync.WaitGroup

	mu   sync.Mutex
	last map[string]time.Time
}

func New(
	db *gorm.DB,
	adminID int64,
	notify func(int64, string) error,
	metrics *security.Metrics,
	readyFunc ...func(ctx context.Context) error,
) *Monitor {
	ready := func(ctx context.Context) error {
		_, err := database.Ready(ctx, db)
		return err
	}
	if len(readyFunc) > 0 && readyFunc[0] != nil {
		ready = readyFunc[0]
	}
	return &Monitor{
		db:       db,
		adminID:  adminID,
		notify:   notify,
		metrics:  metrics,
		ready:    ready,
		interval: time.Minute,
		cooldown: 30 * time.Minute,
		stop:     make(chan struct{}),
		last:     make(map[string]time.Time),
	}
}

func (m *Monitor) Start() {
	if m == nil || m.db == nil {
		return
	}
	m.wg.Add(1)
	go m.loop()
}

func (m *Monitor) Stop() {
	if m == nil {
		return
	}
	select {
	case <-m.stop:
	default:
		close(m.stop)
	}
	m.wg.Wait()
}

func (m *Monitor) loop() {
	defer m.wg.Done()
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()
	m.CheckOnce(context.Background())
	for {
		select {
		case <-m.stop:
			return
		case <-ticker.C:
			m.CheckOnce(context.Background())
		}
	}
}

// CheckOnce runs every check and returns the alerts that were actually sent
// (after applying the cooldown). Exposed for tests.
func (m *Monitor) CheckOnce(ctx context.Context) []string {
	if m == nil || m.db == nil {
		return nil
	}
	var alerts []string
	now := time.Now()

	var failedOutbox int64
	if err := m.db.Model(&models.OutboxEvent{}).Where("status = 'failed'").Count(&failedOutbox).Error; err != nil {
		alerts = append(alerts, m.alert("db_outbox_query", fmt.Sprintf("⚠️ Мониторинг: не удалось проверить outbox: %v", err)))
	} else if failedOutbox > 0 {
		alerts = append(alerts, m.alert("outbox_failed", fmt.Sprintf("⚠️ <b>Outbox:</b> %d событий в статусе failed. Проверьте доставку Telegram.", failedOutbox)))
	}

	var missingConfirmations int64
	err := m.db.Model(&models.PaymentTransaction{}).
		Where("status = 'SUCCESS' AND created_at >= ? AND created_at < ?", now.AddDate(0, 0, -7), now.Add(-15*time.Minute)).
		Where("COALESCE(pay_method, '') NOT IN ?", []string{"admin_manual", "historical_access"}).
		Where("COALESCE(source, '') NOT IN ?", []string{"manual", "historical"}).
		Where("NOT EXISTS (SELECT 1 FROM analytics_events ae WHERE ae.payment_id = payment_transactions.payment_id AND ae.step = 'payment_confirmed')").
		Count(&missingConfirmations).Error
	if err != nil {
		alerts = append(alerts, m.alert("db_payment_query", fmt.Sprintf("⚠️ Мониторинг: не удалось проверить платежи: %v", err)))
	} else if missingConfirmations > 0 {
		alerts = append(alerts, m.alert("payment_without_event", fmt.Sprintf("⚠️ <b>Платежи без payment_confirmed:</b> %d за 7 дней. Проверьте финансовую воронку.", missingConfirmations)))
	}

	var heartbeat models.BackupHeartbeat
	err = m.db.Where("id = 1").First(&heartbeat).Error
	switch {
	case err == gorm.ErrRecordNotFound:
		alerts = append(alerts, m.alert("backup_missing", "⚠️ <b>Backup:</b> нет ни одной успешной резервной копии."))
	case err != nil:
		alerts = append(alerts, m.alert("backup_query", fmt.Sprintf("⚠️ Мониторинг: не удалось проверить backup: %v", err)))
	case now.Sub(heartbeat.LastSuccessAt) > 24*time.Hour:
		alerts = append(alerts, m.alert("backup_stale", fmt.Sprintf("⚠️ <b>Backup устарел:</b> последняя успешная копия %s назад.", humanDuration(now.Sub(heartbeat.LastSuccessAt)))))
	}

	readyCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := m.ready(readyCtx); err != nil {
		alerts = append(alerts, m.alert("readiness", fmt.Sprintf("🚨 <b>Readiness failed:</b> %v", err)))
	}

	if m.metrics != nil {
		if failures := m.metrics.SwapWebhook5xx(); failures > 0 {
			alerts = append(alerts, m.alert("webhook_5xx", fmt.Sprintf("🚨 <b>Webhook 5xx:</b> %d ответов с ошибкой с прошлой проверки.", failures)))
		}
	}
	return nonEmpty(alerts)
}

func nonEmpty(values []string) []string {
	out := values[:0]
	for _, value := range values {
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func (m *Monitor) alert(key, message string) string {
	now := time.Now()
	m.mu.Lock()
	if last, ok := m.last[key]; ok && now.Sub(last) < m.cooldown {
		m.mu.Unlock()
		return ""
	}
	m.last[key] = now
	m.mu.Unlock()

	log.Printf("[Monitoring] %s: %s", key, message)
	if m.notify != nil && m.adminID != 0 {
		if err := m.notify(m.adminID, message); err != nil {
			log.Printf("[Monitoring] failed to deliver %s alert: %v", key, err)
		}
	}
	return message
}

func humanDuration(d time.Duration) string {
	if d < time.Hour {
		return fmt.Sprintf("%d мин", int(d.Minutes()))
	}
	if d < 48*time.Hour {
		return fmt.Sprintf("%d ч", int(d.Hours()))
	}
	return fmt.Sprintf("%d дн", int(d.Hours()/24))
}
