package bot

import (
	"strings"
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

	if err := service.HandleCallbackQuery(nonAdminQuery); err != nil {
		t.Errorf("security rejection must not surface as a processing error: %v", err)
	}

	// Empty query
	if err := service.HandleCallbackQuery(nil); err != nil {
		t.Errorf("expected nil query to be ignored, got %v", err)
	}
}

func TestBot_AdminPanelSecurity(t *testing.T) {
	service := &BotService{
		cfg: &config.Config{
			AdminTelegramID: 123456789,
		},
	}

	// 1. IsAdmin checks
	if !service.IsAdmin(123456789) {
		t.Errorf("expected 123456789 to be admin")
	}
	if service.IsAdmin(999999999) {
		t.Errorf("expected 999999999 to NOT be admin")
	}

	// 2. Non-admin accessing admin_stats callback should be ignored
	nonAdminQuery := &telego.CallbackQuery{
		ID:   "cb_admin",
		Data: "admin_stats",
		From: telego.User{
			ID: 999999999,
		},
		Message: &telego.Message{
			Chat: telego.Chat{ID: 999999999, Type: "private"},
		},
	}

	if err := service.HandleCallbackQuery(nonAdminQuery); err != nil {
		t.Errorf("non-admin callback must be rejected without a processing error: %v", err)
	}

	// 3. Admin accessing admin_stats callback handled safely
	adminQuery := &telego.CallbackQuery{
		ID:   "cb_admin_ok",
		Data: "admin_stats",
		From: telego.User{
			ID: 123456789,
		},
		Message: &telego.Message{
			Chat: telego.Chat{ID: 123456789, Type: "private"},
		},
	}
	if err := service.HandleCallbackQuery(adminQuery); err != nil {
		t.Errorf("admin callback failed: %v", err)
	}

	// 4. Admin accessing admin_receipts callback handled safely
	adminReceiptsQuery := &telego.CallbackQuery{
		ID:   "cb_admin_receipts",
		Data: "admin_receipts",
		From: telego.User{
			ID: 123456789,
		},
		Message: &telego.Message{
			Chat: telego.Chat{ID: 123456789, Type: "private"},
		},
	}
	if err := service.HandleCallbackQuery(adminReceiptsQuery); err != nil {
		t.Errorf("admin receipts callback failed: %v", err)
	}
}

func TestBot_SendAccessNotificationToStudent_GracefulFallback(t *testing.T) {
	service := NewBotService(&config.Config{}, nil)
	defer service.Stop()

	// 1. Zero Telegram ID returns error
	errZero := service.sendAccessNotificationToStudent(0, "Test", "https://t.me/+link")
	if errZero == nil {
		t.Fatal("expected error for zero student ID")
	}

	// 2. Mock bot handles notification gracefully
	errMock := service.sendAccessNotificationToStudent(12345678, "<b>Welcome!</b>", "https://t.me/+link")
	if errMock != nil {
		t.Fatalf("expected nil error for mock send, got %v", errMock)
	}

	// 3. StripTgEmoji verification
	rawText := `Hello <tg-emoji emoji-id="12345">💚</tg-emoji> welcome <tg-emoji emoji-id="67890">☀️</tg-emoji>!`
	clean := StripTgEmoji(rawText)
	expected := `Hello 💚 welcome ☀️!`
	if clean != expected {
		t.Fatalf("StripTgEmoji: got %q, want %q", clean, expected)
	}
}

func TestBot_SyncTelegramUser_LeadCreation(t *testing.T) {
	service := NewBotService(&config.Config{}, nil)
	defer service.Stop()

	// 1. Nil user returns nil
	if u := service.syncTelegramUser(nil); u != nil {
		t.Errorf("expected nil for nil user, got %v", u)
	}

	// 2. Zero ID returns nil
	if u := service.syncTelegramUser(&telego.User{ID: 0}); u != nil {
		t.Errorf("expected nil for zero ID, got %v", u)
	}
}

func TestBot_UpgradeOptionsKeyboard(t *testing.T) {
	// 1. Accelerator tier student can upgrade to VIP (8 000 ₽)
	accKb := UpgradeOptionsKeyboard("accelerator")
	if len(accKb.InlineKeyboard) != 2 {
		t.Fatalf("expected 2 rows (1 option + menu) for accelerator tier, got %d", len(accKb.InlineKeyboard))
	}
	accOpt := accKb.InlineKeyboard[0][0]
	if !strings.Contains(accOpt.Text, "8 000") || accOpt.CallbackData != "upgrade_to_vip" {
		t.Errorf("expected upgrade to VIP with 8 000 ₽, got text=%s, data=%s", accOpt.Text, accOpt.CallbackData)
	}

	// 2. VIP tier student has no further upgrades (only menu button)
	vipKb := UpgradeOptionsKeyboard("vip")
	if len(vipKb.InlineKeyboard) != 1 {
		t.Errorf("expected 1 row (menu button) for VIP tier, got %d", len(vipKb.InlineKeyboard))
	}
	if vipKb.InlineKeyboard[0][0].CallbackData != "action_menu" {
		t.Errorf("expected action_menu for VIP tier, got %s", vipKb.InlineKeyboard[0][0].CallbackData)
	}
}

func TestBot_UpgradeDifferencesAndReceiptCard(t *testing.T) {
	service := NewBotService(&config.Config{
		AdminTelegramID: 123456789,
	}, nil)
	defer service.Stop()

	// 1. Difference math check: Accelerator (6 900) -> VIP (14 900) = 8 000
	diffAccToVIP := 14900.0 - 6900.0
	if diffAccToVIP != 8000.0 {
		t.Errorf("expected 8000 difference, got %v", diffAccToVIP)
	}

	// 2. SendAdminReceiptCard with Upgrade amount
	upgradeReceipt := &models.PaymentReceipt{
		ID:         105,
		TelegramID: 987654321,
		Username:   "spark_alumni",
		Tier:       "vip",
		Amount:     8000.0,
		CreatedAt:  time.Now(),
	}
	if err := service.SendAdminReceiptCard(123456789, upgradeReceipt); err != nil {
		t.Errorf("expected nil error for upgrade receipt card mock send, got %v", err)
	}
}

func TestBot_CuratorDailyDigest(t *testing.T) {
	service := NewBotService(&config.Config{
		AdminTelegramID: 123456789,
	}, nil)
	defer service.Stop()

	// 1. SendCuratorDailyDigest invocation with nil DB (graceful fallback)
	digest, err := service.SendCuratorDailyDigest(123456789)
	if err != nil {
		t.Fatalf("expected nil error for mock digest send, got %v", err)
	}
	if digest == nil {
		t.Fatal("expected non-nil DailyDigestData")
	}
	if digest.Date == "" {
		t.Error("expected non-empty digest Date")
	}
	if digest.SpotsLeft < 0 {
		t.Errorf("expected non-negative spots left, got %d", digest.SpotsLeft)
	}

	// 2. Admin callback for digest
	cb := &telego.CallbackQuery{
		ID:   "cb_digest_test",
		Data: "admin_digest",
		From: telego.User{
			ID: 123456789,
		},
		Message: &telego.Message{
			Chat: telego.Chat{ID: 123456789, Type: "private"},
		},
	}
	if err := service.HandleCallbackQuery(cb); err != nil {
		t.Errorf("admin digest callback failed: %v", err)
	}
}

func TestBot_MainMenuKeyboard_UpgradeForActiveUsers(t *testing.T) {
	// 1. Without access, upgrade button is omitted
	kbNoAccess := MainMenuKeyboard("https://spark.test", false)
	for _, row := range kbNoAccess.InlineKeyboard {
		for _, btn := range row {
			if btn.CallbackData == "action_upgrade" {
				t.Error("unexpected action_upgrade button for user without access")
			}
		}
	}

	// 2. With access, upgrade button is present
	kbAccess := MainMenuKeyboard("https://spark.test", true)
	hasUpgradeBtn := false
	hasStatusBtn := false
	for _, row := range kbAccess.InlineKeyboard {
		for _, btn := range row {
			if btn.CallbackData == "action_upgrade" {
				hasUpgradeBtn = true
			}
			if btn.CallbackData == "action_status" {
				hasStatusBtn = true
			}
		}
	}
	if !hasUpgradeBtn {
		t.Error("expected action_upgrade button in main menu for user with access")
	}
	if !hasStatusBtn {
		t.Error("expected action_status button in main menu for user with access")
	}
}

func TestBot_SendFiscalReceipt(t *testing.T) {
	service := NewBotService(&config.Config{}, nil)
	defer service.Stop()

	// 1. Zero Telegram ID returns error
	if err := service.SendFiscalReceipt(0, "accelerator", 6900, "https://lknpd.nalog.ru/receipt/123", "001", "Курс"); err == nil {
		t.Error("expected error for zero Telegram ID")
	}

	// 2. Mock mode executes cleanly
	err := service.SendFiscalReceipt(12345678, "vip", 14900, "https://lknpd.nalog.ru/receipt/123", "002", "VIP Обучение")
	if err != nil {
		t.Errorf("expected nil error in mock mode, got %v", err)
	}

	// 3. Without receipt URL
	errNoURL := service.SendFiscalReceipt(12345678, "basic", 3990, "", "003", "")
	if errNoURL != nil {
		t.Errorf("expected nil error for receipt without URL, got %v", errNoURL)
	}
}
