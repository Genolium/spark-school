package outbox

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/spark-school/backend/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// TelegramNotificationPayload structure serialized into outbox payload.
type TelegramNotificationPayload struct {
	TelegramID int64  `json:"telegram_id"`
	Message    string `json:"message"`
}

// AdminAlertPayload structure for receipt cards and system alerts.
type AdminAlertPayload struct {
	AdminTelegramID int64  `json:"admin_telegram_id"`
	ReceiptID       uint   `json:"receipt_id"`
	StudentName     string `json:"student_name"`
	Username        string `json:"username"`
	Amount          float64 `json:"amount"`
	Tier            string `json:"tier"`
	FileID          string `json:"file_id,omitempty"`
}

// Dispatcher processes queued Outbox events with retries and exponential backoff.
type Dispatcher struct {
	db                  *gorm.DB
	notifyFunc          func(telegramID int64, message string) error
	sendReceiptCardFunc func(adminID int64, payload AdminAlertPayload) error
	pollInterval        time.Duration
	stopChan            chan struct{}
	wg                  sync.WaitGroup
}

// NewDispatcher creates a new Outbox Dispatcher.
func NewDispatcher(
	db *gorm.DB,
	notifyFunc func(telegramID int64, message string) error,
	sendReceiptCardFunc func(adminID int64, payload AdminAlertPayload) error,
) *Dispatcher {
	return &Dispatcher{
		db:                  db,
		notifyFunc:          notifyFunc,
		sendReceiptCardFunc: sendReceiptCardFunc,
		pollInterval:        2 * time.Second,
		stopChan:            make(chan struct{}),
	}
}

// EnqueueNotificationTx enqueues a telegram notification inside an active database transaction.
func EnqueueNotificationTx(tx *gorm.DB, telegramID int64, message string) error {
	if tx == nil {
		return fmt.Errorf("transaction is nil")
	}
	payloadBytes, err := json.Marshal(TelegramNotificationPayload{
		TelegramID: telegramID,
		Message:    message,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal notification payload: %w", err)
	}

	event := models.OutboxEvent{
		EventType:  "telegram_notification",
		Payload:    string(payloadBytes),
		Status:     "pending",
		CreatedAt:  time.Now(),
		MaxRetries: 5,
	}

	return tx.Create(&event).Error
}

// EnqueueAdminReceiptAlertTx enqueues an admin receipt review card alert inside an active database transaction.
func EnqueueAdminReceiptAlertTx(tx *gorm.DB, payload AdminAlertPayload) error {
	if tx == nil {
		return fmt.Errorf("transaction is nil")
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal admin alert payload: %w", err)
	}

	event := models.OutboxEvent{
		EventType:  "admin_receipt_card",
		Payload:    string(payloadBytes),
		Status:     "pending",
		CreatedAt:  time.Now(),
		MaxRetries: 5,
	}

	return tx.Create(&event).Error
}

// Start launches the background worker loop.
func (d *Dispatcher) Start() {
	d.wg.Add(1)
	go d.workerLoop()
}

// Stop gracefully shuts down the worker.
func (d *Dispatcher) Stop() {
	close(d.stopChan)
	d.wg.Wait()
}

func (d *Dispatcher) workerLoop() {
	defer d.wg.Done()
	ticker := time.NewTicker(d.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-d.stopChan:
			return
		case <-ticker.C:
			d.processPendingEvents()
		}
	}
}

// ProcessPendingEvents manually executes a processing cycle (public for tests).
func (d *Dispatcher) ProcessPendingEvents() {
	d.processPendingEvents()
}

func (d *Dispatcher) processPendingEvents() {
	if d.db == nil {
		return
	}

	var events []models.OutboxEvent
	// Fetch pending or retryable events with row locking
	err := d.db.Transaction(func(tx *gorm.DB) error {
		return tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("status IN ('pending', 'processing') AND retry_count < max_retries").
			Order("created_at ASC").
			Limit(10).
			Find(&events).Error
	})

	if err != nil || len(events) == 0 {
		return
	}

	for _, event := range events {
		d.processSingleEvent(event)
	}
}

func (d *Dispatcher) processSingleEvent(event models.OutboxEvent) {
	var procErr error

	switch event.EventType {
	case "telegram_notification":
		var p TelegramNotificationPayload
		if err := json.Unmarshal([]byte(event.Payload), &p); err != nil {
			procErr = err
		} else if d.notifyFunc != nil {
			procErr = d.notifyFunc(p.TelegramID, p.Message)
		}
	case "admin_receipt_card":
		var p AdminAlertPayload
		if err := json.Unmarshal([]byte(event.Payload), &p); err != nil {
			procErr = err
		} else if d.sendReceiptCardFunc != nil {
			procErr = d.sendReceiptCardFunc(p.AdminTelegramID, p)
		}
	default:
		log.Printf("[Outbox] Unknown event type: %s", event.EventType)
	}

	now := time.Now()
	if d.db != nil {
		if procErr == nil {
			_ = d.db.Model(&models.OutboxEvent{}).Where("id = ?", event.ID).Updates(map[string]interface{}{
				"status":       "completed",
				"processed_at": &now,
				"last_error":   "",
			})
		} else {
			newCount := event.RetryCount + 1
			newStatus := "processing"
			if newCount >= event.MaxRetries {
				newStatus = "failed"
			}
			_ = d.db.Model(&models.OutboxEvent{}).Where("id = ?", event.ID).Updates(map[string]interface{}{
				"status":      newStatus,
				"retry_count": newCount,
				"last_error":  procErr.Error(),
			})
			log.Printf("[Outbox] Failed delivering event #%d (%s): %v. Retry %d/%d",
				event.ID, event.EventType, procErr, newCount, event.MaxRetries)
		}
	}
}
