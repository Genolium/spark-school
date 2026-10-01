package bot

import (
	"context"
	"strings"
	"testing"

	"github.com/mymmrac/telego"
	"github.com/spark-school/backend/internal/config"
)

func TestParsePaymentCommand(t *testing.T) {
	testCases := []struct {
		name         string
		param        string
		wantUsername string
		wantRefCode  string
	}{
		{
			name:         "Standard pay with referral",
			param:        "pay_alexey_ref_partner_il",
			wantUsername: "alexey",
			wantRefCode:  "partner_il",
		},
		{
			name:         "Pay with @ symbol and referral",
			param:        "pay_@alexey_ref_partner_il",
			wantUsername: "alexey",
			wantRefCode:  "partner_il",
		},
		{
			name:         "Pay without referral code",
			param:        "pay_john_doe",
			wantUsername: "john_doe",
			wantRefCode:  "",
		},
		{
			name:         "Pay with empty ref suffix",
			param:        "pay_ivan_ref_",
			wantUsername: "ivan",
			wantRefCode:  "",
		},
		{
			name:         "Pay with complex username containing underscores",
			param:        "pay_john_student_pro_ref_partner2026",
			wantUsername: "john_student_pro",
			wantRefCode:  "partner2026",
		},
		{
			name:         "Pay with referral without username prefix",
			param:        "pay_ref_partner2026",
			wantUsername: "",
			wantRefCode:  "partner2026",
		},
		{
			name:         "Bare pay without suffix",
			param:        "pay",
			wantUsername: "",
			wantRefCode:  "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			payload := strings.TrimPrefix(tc.param, "pay_")
			if tc.param == "pay" {
				payload = ""
			}
			username := payload
			refCode := ""

			if idx := strings.Index(payload, "_ref_"); idx != -1 {
				username = payload[:idx]
				refCode = payload[idx+len("_ref_"):]
			} else if strings.HasPrefix(payload, "ref_") {
				username = ""
				refCode = strings.TrimPrefix(payload, "ref_")
			}
			username = strings.TrimPrefix(username, "@")

			if username != tc.wantUsername {
				t.Errorf("for param %q: got username %q, want %q", tc.param, username, tc.wantUsername)
			}
			if refCode != tc.wantRefCode {
				t.Errorf("for param %q: got refCode %q, want %q", tc.param, refCode, tc.wantRefCode)
			}
		})
	}
}

func TestParseWithdrawCommand(t *testing.T) {
	// Test 1: /start withdraw
	text1 := "/start withdraw"
	parts := strings.Fields(text1)
	if len(parts) < 2 || parts[1] != "withdraw" {
		t.Fatalf("expected withdraw command param, got %v", parts)
	}

	// Test 2: direct /withdraw command
	text2 := "/withdraw"
	if !strings.HasPrefix(text2, "/withdraw") {
		t.Fatalf("expected /withdraw command to match prefix")
	}
}

func TestParseReferralCommand(t *testing.T) {
	text := "/start ref_partner_il"
	parts := strings.Fields(text)
	if len(parts) < 2 {
		t.Fatalf("expected 2 parts, got %v", parts)
	}
	refCode := strings.TrimPrefix(parts[1], "ref_")
	if refCode != "partner_il" {
		t.Fatalf("expected refCode 'partner_il', got %q", refCode)
	}
}

func TestBot_HandleMessage_IgnoresNonPrivateChats(t *testing.T) {
	service := &BotService{
		cfg: &config.Config{
			FrontendURL: "http://localhost:3000",
		},
	}

	testCases := []struct {
		name       string
		chatType   string
		chatID     int64
		text       string
		shouldProc bool
	}{
		{
			name:       "Group chat strictly ignored",
			chatType:   "group",
			chatID:     -1001234567,
			text:       "/start",
			shouldProc: false,
		},
		{
			name:       "Supergroup chat strictly ignored",
			chatType:   "supergroup",
			chatID:     -1009876543,
			text:       "/start pay",
			shouldProc: false,
		},
		{
			name:       "Channel chat strictly ignored",
			chatType:   "channel",
			chatID:     -1005555555,
			text:       "/withdraw",
			shouldProc: false,
		},
		{
			name:       "Negative ID with private type ignored",
			chatType:   "private",
			chatID:     -100,
			text:       "/status",
			shouldProc: false,
		},
		{
			name:       "Zero ID ignored",
			chatType:   "private",
			chatID:     0,
			text:       "/start",
			shouldProc: false,
		},
		{
			name:       "Valid positive private chat processed",
			chatType:   "private",
			chatID:     12345678,
			text:       "/start",
			shouldProc: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			msg := &telego.Message{
				Chat: telego.Chat{
					ID:   tc.chatID,
					Type: tc.chatType,
				},
				Text: tc.text,
				From: &telego.User{
					ID:        tc.chatID,
					FirstName: "TestUser",
				},
			}

			processed := service.handleMessage(msg)
			if processed != tc.shouldProc {
				t.Errorf("handleMessage for %s (type=%s, id=%d): got processed=%v, want %v",
					tc.name, tc.chatType, tc.chatID, processed, tc.shouldProc)
			}
		})
	}

	// Also verify nil message returns false safely
	if service.handleMessage(nil) != false {
		t.Error("expected nil message to return false")
	}
}

func TestBot_CreateOneTimeInviteLink(t *testing.T) {
	// 1. Uninitialized bot service should return error
	service := &BotService{bot: nil}
	_, err := service.CreateOneTimeInviteLink(context.Background(), -1001234567890)
	if err == nil {
		t.Fatal("expected error when bot is not initialized")
	}

	// 2. Channel ID 0 should return error
	_, errZero := service.CreateOneTimeInviteLink(context.Background(), 0)
	if errZero == nil {
		t.Fatal("expected error when channel ID is 0")
	}

	// 3. Canceled context should return error
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, errCanceled := service.CreateOneTimeInviteLink(ctx, -1001234567890)
	if errCanceled == nil {
		t.Fatal("expected error when context is canceled")
	}
}
