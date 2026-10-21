package bot

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/mymmrac/telego"
	"github.com/spark-school/backend/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func claimTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set; skipping PostgreSQL integration test")
	}
	prepareSchema(t, dsn)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	if err := db.AutoMigrate(&models.ProcessedTelegramUpdate{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestPostgres_UpdateClaim_TwentyParallel(t *testing.T) {
	db := claimTestDB(t)
	store := NewUpdateClaimStore(db, time.Minute)

	var wg sync.WaitGroup
	var mu sync.Mutex
	claimedCount := 0
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			claimed, err := store.Claim(context.Background(), 424242)
			if err != nil {
				t.Errorf("claim: %v", err)
				return
			}
			if claimed {
				mu.Lock()
				claimedCount++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if claimedCount != 1 {
		t.Fatalf("expected exactly one claim, got %d", claimedCount)
	}
}

func TestPostgres_UpdateClaim_TwoInstances(t *testing.T) {
	db := claimTestDB(t)
	instanceA := NewUpdateClaimStore(db, time.Minute)
	instanceB := NewUpdateClaimStore(db, time.Minute)

	var wg sync.WaitGroup
	results := make([]bool, 2)
	for i, store := range []*UpdateClaimStore{instanceA, instanceB} {
		wg.Add(1)
		go func(idx int, s *UpdateClaimStore) {
			defer wg.Done()
			claimed, err := s.Claim(context.Background(), 777001)
			if err != nil {
				t.Errorf("claim: %v", err)
			}
			results[idx] = claimed
		}(i, store)
	}
	wg.Wait()
	if results[0] == results[1] {
		t.Fatalf("exactly one instance must claim the update, got %v", results)
	}
}

func TestPostgres_UpdateClaim_LeaseExpiryAndFailedRetry(t *testing.T) {
	db := claimTestDB(t)
	store := NewUpdateClaimStore(db, 80*time.Millisecond)

	claimed, err := store.Claim(context.Background(), 501)
	if err != nil || !claimed {
		t.Fatalf("initial claim: claimed=%v err=%v", claimed, err)
	}
	claimed, err = store.Claim(context.Background(), 501)
	if err != nil {
		t.Fatal(err)
	}
	if claimed {
		t.Fatal("active lease must block a second claim")
	}

	time.Sleep(120 * time.Millisecond)
	claimed, err = store.Claim(context.Background(), 501)
	if err != nil || !claimed {
		t.Fatalf("expired lease must allow takeover: claimed=%v err=%v", claimed, err)
	}

	if err := store.Fail(context.Background(), 501, fmt.Errorf("temporary telegram error")); err != nil {
		t.Fatal(err)
	}
	claimed, err = store.Claim(context.Background(), 501)
	if err != nil || !claimed {
		t.Fatalf("failed update must be retryable: claimed=%v err=%v", claimed, err)
	}

	if err := store.Complete(context.Background(), 501); err != nil {
		t.Fatal(err)
	}
	claimed, err = store.Claim(context.Background(), 501)
	if err != nil {
		t.Fatal(err)
	}
	if claimed {
		t.Fatal("completed update must never be claimed again")
	}
	status, _ := store.Status(context.Background(), 501)
	if status != "completed" {
		t.Fatalf("expected completed status, got %q", status)
	}
}

func TestHandleUpdate_MalformedMessageReturnsError(t *testing.T) {
	service := &BotService{}
	update := &telego.Update{UpdateID: 1, Message: &telego.Message{Chat: telego.Chat{ID: 10, Type: "private"}}}
	// A private message without a sender is malformed. It must be reported as
	// an error rather than silently treated as success.
	err := service.HandleUpdate(context.Background(), update)
	if err == nil {
		t.Fatal("malformed message must be returned as an error")
	}
}
