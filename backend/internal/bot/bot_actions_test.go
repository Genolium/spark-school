package bot

import (
	"testing"
	"time"

	"github.com/mymmrac/telego"
	"github.com/spark-school/backend/internal/config"
	"github.com/spark-school/backend/internal/models"
)

func TestBot_SendAdminReceiptCard_GracefulFallback(t *testing.T) {
	service := NewBotService(&config.Config{
		AdminTelegramID: 123456789,
	}, nil)
	defer service.Stop()

	// 1. Nil receipt should return nil
	if err := service.SendAdminReceiptCard(123456789, nil); err != nil {
		t.Errorf("expected nil error for nil receipt, got %v", err)
	}

	// 2. Zero admin ID should return nil
	if err := service.SendAdminReceiptCard(0, &models.PaymentReceipt{}); err != nil {
		t.Errorf("expected nil error for zero admin ID, got %v", err)
	}

	// 3. Mock bot should handle receipt card gracefully
	receipt := &models.PaymentReceipt{
		ID:         99,
		TelegramID: 777888999,
		Username:   "student_alex",
		Tier:       "accelerator",
		Amount:     6900.0,
		CreatedAt:  time.Now(),
	}
	if err := service.SendAdminReceiptCard(123456789, receipt); err != nil {
		t.Errorf("expected nil error for mock send, got %v", err)
	}
}

func TestBot_HandleCallbackQuery_SecurityCheck(t *testing.T) {
	service := &BotService{
		cfg: &config.Config{
			AdminTelegramID: 123456789,
		},
	}

	// Non-admin query should be rejected or handled without grant
	nonAdminQuery := &telego.CallbackQuery{
		ID:   "cb_1",
		Data: "receipt_approve_10",
		From: telego.User{
			ID: 999999999, // Intruder
		},
	}

	handled := service.HandleCallbackQuery(nonAdminQuery)
	if !handled {
		t.Errorf("expected callback query to be intercepted by security check")
	}

	// Empty query
	if service.HandleCallbackQuery(nil) {
		t.Errorf("expected nil query to return false")
	}
}
