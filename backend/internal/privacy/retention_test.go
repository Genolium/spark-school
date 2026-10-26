package privacy

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/spark-school/backend/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func privacyDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:privacy_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.AnalyticsEvent{}, &models.ExperimentAssignment{}, &models.DripLead{}, &models.PaymentTransaction{}, &models.CampaignLink{}, &models.CampaignLinkClick{}); err != nil {
		t.Fatal(err)
	}
	return db
}

// campaignLink creates a minimal link row so click inserts satisfy the foreign
// key on PostgreSQL and stay readable in SQLite.
func campaignLink(t *testing.T, db *gorm.DB, slug string) models.CampaignLink {
	t.Helper()
	link := models.CampaignLink{Slug: slug, Title: slug, TargetType: "landing", IsActive: true}
	if err := db.Create(&link).Error; err != nil {
		t.Fatal(err)
	}
	return link
}

func TestCleaner_Retention(t *testing.T) {
	db := privacyDB(t)
	old := time.Now().AddDate(0, 0, -200)
	recent := time.Now()
	db.Create(&models.AnalyticsEvent{VisitorID: "v-old", Step: "landing_view", CreatedAt: old})
	db.Create(&models.AnalyticsEvent{VisitorID: "v-new", Step: "landing_view", CreatedAt: recent})
	db.Create(&models.ExperimentAssignment{Experiment: "hero_cta_copy", Variant: "control", VisitorID: "v-old", CreatedAt: old})
	db.Create(&models.ExperimentAssignment{Experiment: "hero_cta_copy", Variant: "control", VisitorID: "v-new", CreatedAt: recent})
	link := campaignLink(t, db, "retention")
	db.Create(&models.CampaignLinkClick{LinkID: link.ID, VisitorID: "v-old", CreatedAt: old})
	db.Create(&models.CampaignLinkClick{LinkID: link.ID, VisitorID: "v-new", CreatedAt: recent})

	cleaner := NewCleaner(db, 180)
	deleted, err := cleaner.RunOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 3 {
		t.Fatalf("expected 3 deleted rows, got %d", deleted)
	}
	var events, assignments, clicks int64
	db.Model(&models.AnalyticsEvent{}).Count(&events)
	db.Model(&models.ExperimentAssignment{}).Count(&assignments)
	db.Model(&models.CampaignLinkClick{}).Count(&clicks)
	if events != 1 || assignments != 1 || clicks != 1 {
		t.Fatalf("recent rows must be kept: events=%d assignments=%d clicks=%d", events, assignments, clicks)
	}
	var kept models.CampaignLinkClick
	if err := db.First(&kept).Error; err != nil {
		t.Fatal(err)
	}
	if kept.VisitorID != "v-new" {
		t.Fatalf("recent click must be kept, got visitor %q", kept.VisitorID)
	}
}

func TestEraseUser_AnonymizesAndKeepsFinance(t *testing.T) {
	db := privacyDB(t)
	now := time.Now()
	user := models.User{
		TelegramID: 777001, Username: "student", FirstName: "Иван", LastName: "Иванов",
		PhotoURL: "https://example.com/photo.jpg", VisitorID: "visitor-erase", AttributionToken: "a_erase",
		HasAccess: true,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	db.Create(&models.AnalyticsEvent{VisitorID: "visitor-erase", AttributionToken: "a_erase", Step: "landing_view", CreatedAt: now})
	db.Create(&models.AnalyticsEvent{VisitorID: "other", Step: "landing_view", CreatedAt: now})
	db.Create(&models.ExperimentAssignment{Experiment: "hero_cta_copy", Variant: "control", VisitorID: "visitor-erase", CreatedAt: now})
	link := campaignLink(t, db, "erase")
	db.Create(&models.CampaignLinkClick{LinkID: link.ID, VisitorID: "visitor-erase", CreatedAt: now})
	db.Create(&models.CampaignLinkClick{LinkID: link.ID, AttributionToken: "a_erase", CreatedAt: now})
	db.Create(&models.CampaignLinkClick{LinkID: link.ID, VisitorID: "other", CreatedAt: now})
	db.Create(&models.PaymentTransaction{PaymentID: "keep-me", TelegramID: user.TelegramID, Amount: 6900, AmountKopecks: 690000, Status: "SUCCESS", CreatedAt: now})

	if err := EraseUser(db, user.ID); err != nil {
		t.Fatal(err)
	}
	var reloaded models.User
	db.First(&reloaded, user.ID)
	if reloaded.FirstName != "Удалённый пользователь" || reloaded.Username != "" || reloaded.VisitorID != "" || reloaded.AttributionToken != "" {
		t.Fatalf("user was not anonymized: %+v", reloaded)
	}
	if reloaded.AnonymizedAt == nil {
		t.Fatal("anonymized_at must be set")
	}
	var events int64
	db.Model(&models.AnalyticsEvent{}).Where("visitor_id = ?", "visitor-erase").Count(&events)
	if events != 0 {
		t.Fatalf("user analytics must be deleted, got %d", events)
	}
	var assignments int64
	db.Model(&models.ExperimentAssignment{}).Where("visitor_id = ?", "visitor-erase").Count(&assignments)
	if assignments != 0 {
		t.Fatalf("user experiment assignments must be deleted, got %d", assignments)
	}
	var clicks int64
	db.Model(&models.CampaignLinkClick{}).Where("visitor_id = ? OR attribution_token = ?", "visitor-erase", "a_erase").Count(&clicks)
	if clicks != 0 {
		t.Fatalf("user click rows must be deleted, got %d", clicks)
	}
	var otherClicks int64
	db.Model(&models.CampaignLinkClick{}).Where("visitor_id = ?", "other").Count(&otherClicks)
	if otherClicks != 1 {
		t.Fatalf("unrelated click must be preserved, got %d", otherClicks)
	}
	var txCount int64
	db.Model(&models.PaymentTransaction{}).Where("payment_id = ?", "keep-me").Count(&txCount)
	if txCount != 1 {
		t.Fatal("financial transaction must be preserved")
	}
	var otherEvents int64
	db.Model(&models.AnalyticsEvent{}).Where("visitor_id = ?", "other").Count(&otherEvents)
	if otherEvents != 1 {
		t.Fatal("unrelated analytics must be preserved")
	}
}
