package bot

import (
	"testing"
	"time"

	"github.com/mymmrac/telego"
	"github.com/spark-school/backend/internal/models"
)

func TestDiagnosticPlanLinksTelegram(t *testing.T) {
	service, db := setupTestBot(t)
	record := models.Diagnostic{
		Token: "abcdef123456", AgeCitizenship: true, UniversityYear: "3",
		EnglishLevel: "b2", Score: 88, CreatedAt: time.Now(),
	}
	if err := db.Create(&record).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.handleDiagnosticPlan(tuID(555), &telego.User{ID: 555, Username: "student"}, record.Token); err != nil {
		t.Fatal(err)
	}
	var reloaded models.Diagnostic
	db.First(&reloaded, record.ID)
	if reloaded.TelegramID != 555 || reloaded.LinkedAt == nil {
		t.Fatalf("plan was not linked: %+v", reloaded)
	}
}
