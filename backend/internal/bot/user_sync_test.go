package bot

import (
	"testing"

	"github.com/mymmrac/telego"
	"github.com/spark-school/backend/internal/models"
)

func TestSplitAttributionToken(t *testing.T) {
	cases := []struct {
		in        string
		wantParam string
		wantToken string
	}{
		{"pay_accelerator_a_apy1xadvhvmo1", "pay_accelerator", "apy1xadvhvmo1"},
		{"pay_accelerator_a_apy1xadvhvmo1_ref_partner", "pay_accelerator_ref_partner", "apy1xadvhvmo1"},
		{"pay_accelerator_pe_lower_a_apy1xadvhvmo1", "pay_accelerator_pe_lower", "apy1xadvhvmo1"},
		{"free_audit_a_apy1xadvhvmo1", "free_audit", "apy1xadvhvmo1"},
		{"calc_b1_45_a_apy1xadvhvmo1", "calc_b1_45", "apy1xadvhvmo1"},
		{"a_apy1xadvhvmo1", "", "apy1xadvhvmo1"},
		{"plan_abc123", "plan_abc123", ""},
		{"pay", "pay", ""},
		{"", "", ""},
	}
	for _, tc := range cases {
		param, token := splitAttributionToken(tc.in)
		if param != tc.wantParam || token != tc.wantToken {
			t.Errorf("splitAttributionToken(%q) = (%q, %q), want (%q, %q)", tc.in, param, token, tc.wantParam, tc.wantToken)
		}
	}
}

func TestAttributeLeadRetriesUntilLandingEventExists(t *testing.T) {
	service, db := setupTestBot(t)
	from := &telego.User{ID: 888001, Username: "lead"}
	if err := db.Create(&models.User{TelegramID: from.ID, FirstName: "Lead"}).Error; err != nil {
		t.Fatal(err)
	}

	// The landing event may not be ingested yet when the bot receives /start.
	// The token must be stored so a later /start can finish the attribution.
	if err := service.attributeLead(from, "apy1xadvhvmo1"); err != nil {
		t.Fatal(err)
	}
	var user models.User
	if err := db.Where("telegram_id = ?", from.ID).First(&user).Error; err != nil {
		t.Fatal(err)
	}
	if user.AttributionToken != "apy1xadvhvmo1" {
		t.Fatalf("expected the token to be stored, got %q", user.AttributionToken)
	}
	if user.VisitorID != "" {
		t.Fatalf("visitor must stay empty until the landing event exists, got %q", user.VisitorID)
	}
	var leads int64
	db.Model(&models.AnalyticsEvent{}).Where("step = ?", "bot_lead").Count(&leads)
	if leads != 0 {
		t.Fatalf("must not create a bot_lead without a visitor id, got %d", leads)
	}

	if err := db.Create(&models.AnalyticsEvent{
		VisitorID:        "visitor-x",
		AttributionToken: "apy1xadvhvmo1",
		Step:             "landing_view",
	}).Error; err != nil {
		t.Fatal(err)
	}

	if err := service.attributeLead(from, "apy1xadvhvmo1"); err != nil {
		t.Fatal(err)
	}
	db.Where("telegram_id = ?", from.ID).First(&user)
	if user.VisitorID != "visitor-x" {
		t.Fatalf("expected the visitor id from the landing event, got %q", user.VisitorID)
	}
	db.Model(&models.AnalyticsEvent{}).Where("step = ?", "bot_lead").Count(&leads)
	if leads != 1 {
		t.Fatalf("expected exactly one bot_lead event, got %d", leads)
	}

	// A second account or a repeat /start must not duplicate the lead.
	if err := service.attributeLead(from, "apy1xadvhvmo1"); err != nil {
		t.Fatal(err)
	}
	db.Model(&models.AnalyticsEvent{}).Where("step = ?", "bot_lead").Count(&leads)
	if leads != 1 {
		t.Fatalf("expected bot_lead to stay idempotent, got %d", leads)
	}
}
