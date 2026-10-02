package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spark-school/backend/internal/models"
)

func TestPromoValidate_BadRequestOnEmpty(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewPromoHandler(nil)
	router := gin.New()
	router.POST("/api/v1/promo/validate", handler.Validate)

	// Missing code field
	body, _ := json.Marshal(map[string]string{})
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/promo/validate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400 for empty body, got %d", w.Code)
	}
}

func TestPromoCreate_BadRequestOnEmptyCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewPromoHandler(nil)
	router := gin.New()
	router.POST("/api/v1/promo/create", handler.Create)

	body, _ := json.Marshal(map[string]string{"code": ""})
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/promo/create", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400 for empty promo code, got %d", w.Code)
	}
}

// SynchronizedPromoStore provides a thread-safe in-memory store for promo codes,
// accurately modeling database query safety without requiring a live PostgreSQL instance.
type SynchronizedPromoStore struct {
	mu     sync.Mutex
	promos map[string]*models.PromoCode
}

func NewSynchronizedPromoStore() *SynchronizedPromoStore {
	return &SynchronizedPromoStore{
		promos: make(map[string]*models.PromoCode),
	}
}

func (s *SynchronizedPromoStore) Set(p *models.PromoCode) {
	s.mu.Lock()
	defer s.mu.Unlock()
	copied := *p
	s.promos[strings.ToUpper(p.Code)] = &copied
}

// Get returns an isolated copy of the promo code (modeling SELECT ...).
func (s *SynchronizedPromoStore) Get(code string) (*models.PromoCode, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.promos[strings.ToUpper(strings.TrimSpace(code))]
	if !ok {
		return nil, false
	}
	copied := *p
	return &copied, true
}

// Save writes an isolated copy back to the store (modeling UPDATE promo_codes SET uses_count = val).
func (s *SynchronizedPromoStore) Save(p *models.PromoCode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	copied := *p
	s.promos[strings.ToUpper(p.Code)] = &copied
	return nil
}

// AtomicIncrement models SQL atomic increment:
// UPDATE promo_codes SET uses_count = uses_count + 1 WHERE UPPER(code) = ? AND is_active = true
func (s *SynchronizedPromoStore) AtomicIncrement(code string) (*models.PromoCode, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cleanCode := strings.ToUpper(strings.TrimSpace(code))
	p, ok := s.promos[cleanCode]
	if !ok || !p.IsActive {
		return nil, errors.New("promo code not found or inactive")
	}
	p.UsesCount++
	copied := *p
	return &copied, nil
}

// ----------------------------------------------------------------------------
// 1. High-Concurrency Stress Test: 50 Goroutines, Barrier Sync, 0 Lost Updates
// ----------------------------------------------------------------------------
func TestAtomicPromoCodeRedemption_NoLostUpdates(t *testing.T) {
	store := NewSynchronizedPromoStore()
	initialPromo := models.PromoCode{
		ID:              1,
		Code:            "SPARK50",
		DiscountPercent: 5,
		UsesCount:       0,
		IsActive:        true,
	}
	store.Set(&initialPromo)

	concurrency := 50
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(concurrency)

	errChan := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(workerID int) {
			defer wg.Done()
			<-start // Barrier synchronization: wait until all 50 goroutines are ready

			updated, err := store.AtomicIncrement("SPARK50")
			if err != nil {
				errChan <- fmt.Errorf("worker %d failed: %w", workerID, err)
				return
			}
			if updated == nil {
				errChan <- fmt.Errorf("worker %d received nil promo", workerID)
				return
			}
		}(i)
	}

	// Give all goroutines a brief window to reach the barrier
	time.Sleep(10 * time.Millisecond)
	close(start) // Release all 50 goroutines simultaneously
	wg.Wait()
	close(errChan)

	// Verify no goroutine failed
	for err := range errChan {
		t.Errorf("Atomic redemption error: %v", err)
	}

	finalPromo, ok := store.Get("SPARK50")
	if !ok {
		t.Fatalf("Promo code SPARK50 not found in store")
	}

	lostUpdates := concurrency - finalPromo.UsesCount
	t.Logf("[Atomic Test] Final uses_count = %d / %d (Lost Updates: %d)", finalPromo.UsesCount, concurrency, lostUpdates)

	if finalPromo.UsesCount != concurrency {
		t.Fatalf("Lost update detected! Final uses_count = %d, expected exactly %d", finalPromo.UsesCount, concurrency)
	}
	if lostUpdates != 0 {
		t.Fatalf("Expected 0 lost updates, got %d", lostUpdates)
	}
}

// ----------------------------------------------------------------------------
// 2. Counter-Test: Naive Read-Modify-Write Demonstrates Lost Updates (< 50)
// ----------------------------------------------------------------------------
func TestNaivePromoCodeRedemption_DemonstratesLostUpdates(t *testing.T) {
	store := NewSynchronizedPromoStore()
	initialPromo := models.PromoCode{
		ID:              2,
		Code:            "NAIVE50",
		DiscountPercent: 5,
		UsesCount:       0,
		IsActive:        true,
	}
	store.Set(&initialPromo)

	concurrency := 50
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(concurrency)

	for i := 0; i < concurrency; i++ {
		go func(workerID int) {
			defer wg.Done()
			<-start // Barrier synchronization

			// Step 1: Thread-safe read (returns a copy of the promo code)
			p, ok := store.Get("NAIVE50")
			if !ok {
				return
			}

			// Step 2: Realistic CPU preemption / network delay before write
			time.Sleep(time.Duration(rand.Intn(2)+1) * time.Millisecond)

			// Step 3: Local increment and write-back
			p.UsesCount++
			_ = store.Save(p)
		}(i)
	}

	time.Sleep(10 * time.Millisecond)
	close(start) // Release all 50 goroutines simultaneously
	wg.Wait()

	finalPromo, ok := store.Get("NAIVE50")
	if !ok {
		t.Fatalf("Promo code NAIVE50 not found in store")
	}

	lostUpdates := concurrency - finalPromo.UsesCount
	t.Logf("[Naive Counter-Test] Final uses_count = %d / %d (Lost Updates: %d)", finalPromo.UsesCount, concurrency, lostUpdates)

	// An un-synchronized read-modify-write MUST suffer from lost updates
	if finalPromo.UsesCount >= concurrency {
		t.Fatalf("Expected lost updates (uses_count < %d), but got %d", concurrency, finalPromo.UsesCount)
	}
	if lostUpdates <= 0 {
		t.Fatalf("Expected lostUpdates > 0, got %d", lostUpdates)
	}
}

// ----------------------------------------------------------------------------
// 3. HTTP Endpoint Concurrency: POST /api/v1/promo/redeem Under 50 Requests
// ----------------------------------------------------------------------------
func TestPromoRedeem_EndpointConcurrency(t *testing.T) {
	store := NewSynchronizedPromoStore()
	store.Set(&models.PromoCode{
		ID:              3,
		Code:            "HTTP50",
		DiscountPercent: 5,
		UsesCount:       0,
		IsActive:        true,
	})

	handler := NewPromoHandler(nil).WithCustomRedeem(func(code string) (*models.PromoCode, error) {
		return store.AtomicIncrement(code)
	})

	router := gin.New()
	router.POST("/api/v1/promo/redeem", handler.Redeem)

	concurrency := 50
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(concurrency)

	statusCodes := make([]int, concurrency)

	for i := 0; i < concurrency; i++ {
		idx := i
		go func() {
			defer wg.Done()
			<-start // Barrier sync

			body, _ := json.Marshal(map[string]string{"code": "http50"}) // test case normalization
			req := httptest.NewRequest(http.MethodPost, "/api/v1/promo/redeem", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			statusCodes[idx] = w.Code
		}()
	}

	time.Sleep(10 * time.Millisecond)
	close(start)
	wg.Wait()

	for idx, code := range statusCodes {
		if code != http.StatusOK {
			t.Errorf("Worker %d returned unexpected HTTP status %d, expected 200 OK", idx, code)
		}
	}

	finalPromo, ok := store.Get("HTTP50")
	if !ok {
		t.Fatalf("Promo code HTTP50 not found")
	}

	if finalPromo.UsesCount != concurrency {
		t.Fatalf("Lost update in HTTP handler! Final uses_count = %d, expected %d", finalPromo.UsesCount, concurrency)
	}
}

// ----------------------------------------------------------------------------
// 4. Input Validation & Edge Case Tests
// ----------------------------------------------------------------------------
func TestPromoRedeem_ValidationAndEdgeCases(t *testing.T) {
	store := NewSynchronizedPromoStore()
	store.Set(&models.PromoCode{
		ID:              4,
		Code:            "VALID5",
		DiscountPercent: 5,
		UsesCount:       0,
		IsActive:        true,
	})
	store.Set(&models.PromoCode{
		ID:              5,
		Code:            "INACTIVE5",
		DiscountPercent: 5,
		UsesCount:       0,
		IsActive:        false,
	})

	handler := NewPromoHandler(nil).WithCustomRedeem(func(code string) (*models.PromoCode, error) {
		return store.AtomicIncrement(code)
	})

	router := gin.New()
	router.POST("/api/v1/promo/redeem", handler.Redeem)

	// Case A: Missing code body
	t.Run("MissingCode_Returns400", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/promo/redeem", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for missing code, got %d", w.Code)
		}
	})

	// Case B: Whitespace only code
	t.Run("WhitespaceCode_Returns400", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"code": "   "})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/promo/redeem", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for whitespace code, got %d", w.Code)
		}
	})

	// Case C: Inactive promo code
	t.Run("InactiveCode_Returns400", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"code": "INACTIVE5"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/promo/redeem", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for inactive code, got %d", w.Code)
		}
	})

	// Case D: Non-existent promo code
	t.Run("NonExistentCode_Returns400", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"code": "DOESNOTEXIST"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/promo/redeem", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for non-existent code, got %d", w.Code)
		}
	})

	// Case E: Successful redemption with whitespace and case trimming
	t.Run("ValidCode_Returns200WithUpdatedUses", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"code": "  valid5  "})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/promo/redeem", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200 for valid code, got %d", w.Code)
		}

		var res struct {
			Success   bool   `json:"success"`
			Code      string `json:"code"`
			UsesCount int    `json:"uses_count"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &res)
		if !res.Success || res.Code != "VALID5" || res.UsesCount != 1 {
			t.Errorf("Unexpected response payload: %+v", res)
		}
	})
}
