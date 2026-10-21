package bot

import (
	"encoding/json"
	"time"

	"github.com/spark-school/backend/internal/models"
	"github.com/spark-school/backend/internal/outbox"
	"gorm.io/gorm"
)

// enqueueAdminAlert is used by background jobs that need a durable alert.
func enqueueAdminAlert(tx *gorm.DB, adminID int64, payload outbox.AdminAlertPayload) error {
	if adminID == 0 {
		return nil
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return tx.Create(&models.OutboxEvent{
		EventType:  "admin_receipt_card",
		Payload:    string(raw),
		Status:     "pending",
		MaxRetries: 5,
		CreatedAt:  time.Now(),
	}).Error
}
