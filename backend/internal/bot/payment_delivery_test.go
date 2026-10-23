package bot

import (
	"testing"
	"time"

	"github.com/spark-school/backend/internal/models"
	"github.com/spark-school/backend/internal/outbox"
)

func TestDeliverPaymentNotice_RecordsAccessDeliveredForFirstPurchaseAndUpgrade(t *testing.T) {
	service, db := setupTestBot(t)
	service.cfg.TelegramInviteLink = "https://t.me/+test-invite"
	user := models.User{TelegramID: 777001, FirstName: "Student", VisitorID: "visitor-777", AttributionToken: "attr-777"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}

	for _, kind := range []string{"access", "upgrade"} {
		paymentID := "notice-" + kind
		err := service.DeliverPaymentNotice(outbox.PaymentNoticePayload{
			Kind:       kind,
			TelegramID: user.TelegramID,
			PaymentID:  paymentID,
			Tier:       "vip",
			Amount:     19990,
			Source:     "sbp_receipt",
		})
		if err != nil {
			t.Fatalf("%s delivery must succeed with a mock bot: %v", kind, err)
		}

		var event models.AnalyticsEvent
		if err := db.Where("step = ? AND payment_id = ?", "access_delivered", paymentID).First(&event).Error; err != nil {
			t.Fatalf("%s payment must record an access_delivered event: %v", kind, err)
		}
		if event.VisitorID != user.VisitorID {
			t.Fatalf("expected visitor id %q on the event, got %q", user.VisitorID, event.VisitorID)
		}
		if event.Tier != "vip" {
			t.Fatalf("expected tier vip, got %q", event.Tier)
		}
	}
}

func TestReceiptConfirmInput_UsesApprovalTimeAsPaymentDate(t *testing.T) {
	receipt := models.PaymentReceipt{
		TelegramID: 777002,
		Username:   "student",
		Tier:       "accelerator",
		Amount:     6900,
		Status:     "pending",
		CreatedAt:  time.Now().AddDate(0, 0, -40),
	}

	input := receiptConfirmInput(receipt)
	if age := time.Since(input.CreatedAt); age > time.Minute {
		t.Fatalf("payment date must be the confirmation time, got %s ago", age)
	}
	if input.PaymentID != "RECEIPT_0" {
		t.Fatalf("unexpected payment id %q", input.PaymentID)
	}
}
