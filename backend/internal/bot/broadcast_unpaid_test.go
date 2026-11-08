package bot

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/spark-school/backend/internal/config"
	"github.com/spark-school/backend/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func broadcastTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:broadcast_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.PaymentTransaction{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestUnpaidBroadcastRecipients(t *testing.T) {
	db := broadcastTestDB(t)
	const adminID int64 = 900
	now := time.Now()

	users := []models.User{
		{TelegramID: 101, FirstName: "unpaid", Role: "student"},
		{TelegramID: 102, FirstName: "paid", Role: "student", HasAccess: true},
		{TelegramID: 103, FirstName: "paid, access revoked", Role: "student"},
		{TelegramID: 104, FirstName: "manual access", Role: "student", HasAccess: true},
		{TelegramID: 105, FirstName: "only failed and pending payments", Role: "student"},
		{TelegramID: 106, FirstName: "opted out", Role: "student", RemindersOptOut: true},
		{TelegramID: 107, FirstName: "anonymized", Role: "student", AnonymizedAt: &now},
		{TelegramID: adminID, FirstName: "admin by id", Role: "student"},
		{TelegramID: 109, FirstName: "admin by role", Role: "admin"},
		{TelegramID: 0, FirstName: "no telegram", Role: "student"},
	}
	for i := range users {
		if err := db.Create(&users[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	txs := []models.PaymentTransaction{
		{PaymentID: "p102", TelegramID: 102, Amount: 10990, Status: "SUCCESS"},
		{PaymentID: "p103", TelegramID: 103, Amount: 3333, Status: "SUCCESS"},
		{PaymentID: "p105a", TelegramID: 105, Amount: 10990, Status: "FAIL"},
		{PaymentID: "p105b", TelegramID: 105, Amount: 10990, Status: "PENDING"},
	}
	for i := range txs {
		if err := db.Create(&txs[i]).Error; err != nil {
			t.Fatal(err)
		}
	}

	service := &BotService{db: db, cfg: &config.Config{AdminTelegramID: adminID}}
	got, err := service.unpaidBroadcastRecipients()
	if err != nil {
		t.Fatal(err)
	}
	ids := map[int64]bool{}
	for _, u := range got {
		ids[u.TelegramID] = true
	}
	want := map[int64]bool{101: true, 105: true}
	if len(ids) != len(want) {
		t.Fatalf("recipients = %v, want %v", ids, want)
	}
	for id := range want {
		if !ids[id] {
			t.Fatalf("recipient %d missing, got %v", id, ids)
		}
	}
}

func TestPendingBroadcastIsConsumedOnceAndExpires(t *testing.T) {
	service := &BotService{}
	now := time.Now()

	service.stageBroadcast(7, "hello", now)
	if text, ok := service.takeBroadcast(7, now.Add(time.Minute)); !ok || text != "hello" {
		t.Fatalf("take = %q, %v; want hello, true", text, ok)
	}
	if _, ok := service.takeBroadcast(7, now.Add(time.Minute)); ok {
		t.Fatal("a staged broadcast must be sent at most once")
	}

	service.stageBroadcast(7, "stale", now)
	if _, ok := service.takeBroadcast(7, now.Add(broadcastConfirmTTL+time.Second)); ok {
		t.Fatal("an expired preview must not be sent")
	}

	service.stageBroadcast(7, "first", now)
	service.stageBroadcast(7, "second", now)
	if text, _ := service.takeBroadcast(7, now); text != "second" {
		t.Fatalf("restaging must replace the pending text, got %q", text)
	}
	service.stageBroadcast(7, "mine", now)
	if _, ok := service.takeBroadcast(8, now); ok {
		t.Fatal("another chat must not take this admin's pending broadcast")
	}
}

func TestDeliverBroadcastCountsResults(t *testing.T) {
	users := []models.User{{TelegramID: 1}, {TelegramID: 2}, {TelegramID: 3}}
	var seen []int64
	sent, failed := deliverBroadcast(users, "text", 0, func(chatID int64, text string) error {
		seen = append(seen, chatID)
		if text != "text" {
			t.Fatalf("text = %q", text)
		}
		if chatID == 2 {
			return errors.New("bot was blocked")
		}
		return nil
	})
	if sent != 2 || failed != 1 {
		t.Fatalf("sent=%d failed=%d, want 2 and 1", sent, failed)
	}
	if len(seen) != 3 {
		t.Fatalf("one failure must not stop the broadcast, delivered to %v", seen)
	}
}
