package security

import (
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/spark-school/backend/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestDatabaseRateLimit_SharedWindow(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:rl_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.RateLimit{}); err != nil {
		t.Fatal(err)
	}
	if allowed, _ := allowDatabase(db, "login:1.2.3.4", 2, time.Minute); !allowed {
		t.Fatal("first request must pass")
	}
	if allowed, _ := allowDatabase(db, "login:1.2.3.4", 2, time.Minute); !allowed {
		t.Fatal("second request must pass")
	}
	allowed, retry := allowDatabase(db, "login:1.2.3.4", 2, time.Minute)
	if allowed {
		t.Fatal("third request must be limited")
	}
	if retry <= 0 {
		t.Fatal("expected retry duration")
	}
	// A different key is independent.
	if allowed, _ := allowDatabase(db, "login:5.6.7.8", 2, time.Minute); !allowed {
		t.Fatal("different IP must have its own bucket")
	}
}
