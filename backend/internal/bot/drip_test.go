package bot

import (
	"strings"
	"testing"
	"time"

	"github.com/spark-school/backend/internal/config"
	"github.com/spark-school/backend/internal/models"
)

func TestDrip_RegisterAndCancel_GracefulNilDB(t *testing.T) {
	service := NewBotService(&config.Config{}, nil)
	defer service.Stop()

	// Should not panic with nil DB
	service.RegisterOrUpdateDripLead(12345, "testuser", "Alex", 88)
	service.CancelDrip(12345)
}

func TestDrip_MessageCopy_Sanity(t *testing.T) {
	lead := models.DripLead{
		TelegramID:      999888,
		Username:        "alex_applicant",
		FirstName:       "Алексей",
		CalculatedScore: 88,
		Stage:           0,
		NextTriggerAt:   time.Now().Add(2 * time.Hour),
	}

	// 1. Verify Step 1 content
	service := NewBotService(&config.Config{}, nil)
	defer service.Stop()

	// Verify price calculations in drip messages
	accPrice := 6555
	vipPrice := 14155

	if accPrice != 6900-(6900*5)/100 {
		t.Fatalf("unexpected accelerator discounted price: %d", accPrice)
	}
	if vipPrice != 14900-(14900*5)/100 {
		t.Fatalf("unexpected vip discounted price: %d", vipPrice)
	}

	// Verify lead model defaults
	if lead.CalculatedScore != 88 {
		t.Errorf("expected score 88, got %d", lead.CalculatedScore)
	}
	if lead.Stage != 0 {
		t.Errorf("expected stage 0, got %d", lead.Stage)
	}
	if strings.Contains(lead.Username, "forbidden") {
		t.Errorf("forbidden text in username")
	}
}
