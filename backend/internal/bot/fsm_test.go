package bot

import (
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/spark-school/backend/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestSessionManager_PersistsAcrossInstances(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:fsm_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.BotSession{}); err != nil {
		t.Fatal(err)
	}

	first := NewSessionManager(30*time.Minute, db)
	session := first.Get(555)
	session.State = StateWaitingReceipt
	session.SelectedTier = "vip"
	session.PromoCode = "SPARK5"
	session.DiscountPct = 5
	session.CalcEnglishLevel = "b2"
	first.Save(555, session)

	second := NewSessionManager(30*time.Minute, db)
	loaded := second.Get(555)
	if loaded.State != StateWaitingReceipt || loaded.SelectedTier != "vip" || loaded.PromoCode != "SPARK5" || loaded.CalcEnglishLevel != "b2" {
		t.Fatalf("session was not restored: %+v", loaded)
	}

	second.Reset(555)
	third := NewSessionManager(30*time.Minute, db)
	if got := third.Get(555); got.State != StateDefault || got.PromoCode != "" {
		t.Fatalf("reset must remove persisted session: %+v", got)
	}
}
