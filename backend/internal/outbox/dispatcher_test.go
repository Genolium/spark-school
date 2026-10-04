package outbox

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/spark-school/backend/internal/models"
)

func TestOutbox_ProcessSingleEvent_NotificationSuccess(t *testing.T) {
	var sentID int64
	var sentMsg string

	dispatcher := NewDispatcher(nil, func(tgID int64, msg string) error {
		sentID = tgID
		sentMsg = msg
		return nil
	}, nil)

	payloadBytes, _ := json.Marshal(TelegramNotificationPayload{
		TelegramID: 123456789,
		Message:    "Hello Student!",
	})

	event := models.OutboxEvent{
		ID:         1,
		EventType:  "telegram_notification",
		Payload:    string(payloadBytes),
		Status:     "pending",
		MaxRetries: 5,
		CreatedAt:  time.Now(),
	}

	dispatcher.processSingleEvent(event)

	if sentID != 123456789 {
		t.Errorf("expected sentID 123456789, got %d", sentID)
	}
	if sentMsg != "Hello Student!" {
		t.Errorf("expected 'Hello Student!', got %q", sentMsg)
	}
}

func TestOutbox_ProcessSingleEvent_AdminCardSuccess(t *testing.T) {
	var sentAdminID int64
	var sentAlert AdminAlertPayload

	dispatcher := NewDispatcher(nil, nil, func(adminID int64, payload AdminAlertPayload) error {
		sentAdminID = adminID
		sentAlert = payload
		return nil
	})

	alert := AdminAlertPayload{
		AdminTelegramID: 987654321,
		ReceiptID:       42,
		StudentName:     "Ivan",
		Username:        "ivan_tg",
		Amount:          6900.0,
		Tier:            "accelerator",
	}
	payloadBytes, _ := json.Marshal(alert)

	event := models.OutboxEvent{
		ID:         2,
		EventType:  "admin_receipt_card",
		Payload:    string(payloadBytes),
		Status:     "pending",
		MaxRetries: 5,
		CreatedAt:  time.Now(),
	}

	dispatcher.processSingleEvent(event)

	if sentAdminID != 987654321 {
		t.Errorf("expected adminID 987654321, got %d", sentAdminID)
	}
	if sentAlert.ReceiptID != 42 || sentAlert.Amount != 6900.0 {
		t.Errorf("unexpected payload: %+v", sentAlert)
	}
}

func TestOutbox_ProcessSingleEvent_HandlesError(t *testing.T) {
	callCount := 0
	dispatcher := NewDispatcher(nil, func(tgID int64, msg string) error {
		callCount++
		return errors.New("simulated network timeout")
	}, nil)

	payloadBytes, _ := json.Marshal(TelegramNotificationPayload{
		TelegramID: 555,
		Message:    "Retry me",
	})

	event := models.OutboxEvent{
		ID:         3,
		EventType:  "telegram_notification",
		Payload:    string(payloadBytes),
		Status:     "pending",
		RetryCount: 0,
		MaxRetries: 5,
		CreatedAt:  time.Now(),
	}

	dispatcher.processSingleEvent(event)

	if callCount != 1 {
		t.Errorf("expected notification function to be called once, got %d", callCount)
	}
}
