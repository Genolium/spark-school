package bot

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mymmrac/telego"
	"github.com/spark-school/backend/internal/models"
	"gorm.io/gorm"
)

// confirmPaymentAfterFirstUserRead simulates a payment being confirmed right
// after the bot has read the user row: the first SELECT on users is followed
// by a concurrent write of has_access/tier/paid_amount.
func confirmPaymentAfterFirstUserRead(t *testing.T, db *gorm.DB, telegramID int64) {
	t.Helper()
	fired := false
	err := db.Callback().Query().After("gorm:query").Register("test:concurrent_payment", func(tx *gorm.DB) {
		if fired || tx.Statement.Table != "users" {
			return
		}
		fired = true
		if err := db.Session(&gorm.Session{NewDB: true}).Model(&models.User{}).
			Where("telegram_id = ?", telegramID).
			Updates(map[string]interface{}{"has_access": true, "tier": "vip", "paid_amount": 14900}).Error; err != nil {
			t.Errorf("concurrent payment update: %v", err)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Callback().Query().Remove("test:concurrent_payment") })
}

func assertPaidState(t *testing.T, db *gorm.DB, telegramID int64) models.User {
	t.Helper()
	var u models.User
	if err := db.Where("telegram_id = ?", telegramID).First(&u).Error; err != nil {
		t.Fatal(err)
	}
	if !u.HasAccess || u.Tier != "vip" || u.PaidAmount != 14900 {
		t.Fatalf("concurrent payment was clobbered: has_access=%v tier=%q paid_amount=%v", u.HasAccess, u.Tier, u.PaidAmount)
	}
	return u
}

func TestSyncTelegramUserDoesNotClobberConcurrentPayment(t *testing.T) {
	service, db := setupTestBot(t)
	const tgID int64 = 555001
	if err := db.Create(&models.User{TelegramID: tgID, Username: "old", FirstName: "Old", Tier: "accelerator"}).Error; err != nil {
		t.Fatal(err)
	}
	confirmPaymentAfterFirstUserRead(t, db, tgID)

	if u := service.syncTelegramUser(&telego.User{ID: tgID, Username: "new", FirstName: "New"}); u == nil {
		t.Fatal("expected a user")
	}

	u := assertPaidState(t, db, tgID)
	if u.Username != "new" || u.FirstName != "New" {
		t.Fatalf("profile not updated: username=%q first_name=%q", u.Username, u.FirstName)
	}
}

func TestSetIntendedTierIfNoAccessKeepsPaidTier(t *testing.T) {
	service, db := setupTestBot(t)
	const tgID int64 = 555002
	if err := db.Create(&models.User{TelegramID: tgID, FirstName: "Lead", Tier: "accelerator"}).Error; err != nil {
		t.Fatal(err)
	}
	// The /start handler read the user before the payment was confirmed.
	stale := service.syncTelegramUser(&telego.User{ID: tgID, FirstName: "Lead"})
	if stale == nil || stale.HasAccess {
		t.Fatal("expected an unpaid user")
	}
	if err := db.Model(&models.User{}).Where("id = ?", stale.ID).
		Updates(map[string]interface{}{"has_access": true, "tier": "vip", "paid_amount": 14900}).Error; err != nil {
		t.Fatal(err)
	}

	if err := service.setIntendedTierIfNoAccess(stale.ID, "basic"); err != nil {
		t.Fatal(err)
	}
	assertPaidState(t, db, tgID)

	// Without access the intended tier is still recorded.
	other := models.User{TelegramID: 555003, FirstName: "Other", Tier: "accelerator"}
	if err := db.Create(&other).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.setIntendedTierIfNoAccess(other.ID, "basic"); err != nil {
		t.Fatal(err)
	}
	db.First(&other, other.ID)
	if other.Tier != "basic" {
		t.Fatalf("expected intended tier basic, got %q", other.Tier)
	}
}

func TestSetUserPromoCodeIfEmptyKeepsExistingCode(t *testing.T) {
	service, db := setupTestBot(t)
	u := models.User{TelegramID: 555004, FirstName: "P", PromoCode: "FIRST"}
	if err := db.Create(&u).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.setUserPromoCodeIfEmpty(u.ID, "SECOND"); err != nil {
		t.Fatal(err)
	}
	db.First(&u, u.ID)
	if u.PromoCode != "FIRST" {
		t.Fatalf("existing promo overwritten: %q", u.PromoCode)
	}
}

func newFakeTelegramBot(t *testing.T) *telego.Bot {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":1,"date":0,"chat":{"id":1,"type":"private"}}}`))
	}))
	t.Cleanup(srv.Close)
	b, err := telego.NewBot("123456:"+strings.Repeat("a", 35), telego.WithAPIServer(srv.URL), telego.WithDiscardLogger())
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// A receipt whose amount happens to equal an old 5%-off price must not tag
// the student with SPARK5: the promo comes only from the session/profile.
func TestReceiptUploadDoesNotInferPromoFromAmount(t *testing.T) {
	service, db := setupTestBot(t)
	service.bot = newFakeTelegramBot(t)
	const tgID int64 = 555005
	if err := db.Create(&models.User{TelegramID: tgID, Username: "student", FirstName: "S", Tier: "accelerator"}).Error; err != nil {
		t.Fatal(err)
	}

	session := service.getSession(tgID)
	session.SelectedTier = "vip"
	session.DiscountPct = 5 // legacy session: discount without a code, 14900 -> 14155

	msg := &telego.Message{
		Chat:  telego.Chat{ID: tgID, Type: "private"},
		From:  &telego.User{ID: tgID, Username: "student", FirstName: "S"},
		Photo: []telego.PhotoSize{{FileID: "file-1", FileUniqueID: "uniq-1"}},
	}
	if err := service.HandleReceiptUpload(msg); err != nil {
		t.Fatal(err)
	}

	var receipt models.PaymentReceipt
	if err := db.Where("telegram_id = ?", tgID).First(&receipt).Error; err != nil {
		t.Fatalf("receipt not stored: %v", err)
	}
	if receipt.Amount != 14155 {
		t.Fatalf("test assumes the old SPARK5 amount 14155, got %v", receipt.Amount)
	}
	var u models.User
	db.Where("telegram_id = ?", tgID).First(&u)
	if u.PromoCode != "" || receipt.PromoCode != "" {
		t.Fatalf("promo must not be inferred from the amount: user=%q receipt=%q", u.PromoCode, receipt.PromoCode)
	}
}
