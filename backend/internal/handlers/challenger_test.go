package handlers

import (
	"bytes"
	"encoding/json"
	"math"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/spark-school/backend/internal/models"
)

// TestChallenger_SeatCounterCapacityAutoDoubling rigorously stress-tests the seat counter algorithm.
func TestChallenger_SeatCounterCapacityAutoDoubling(t *testing.T) {
	// Specific test cases mandated by mission
	prescribedCases := []struct {
		n            int64
		expectedCap  int64
		expectedLeft int64
		expectedK    int
	}{
		{n: 0, expectedCap: 25, expectedLeft: 25, expectedK: 0},
		{n: 24, expectedCap: 25, expectedLeft: 1, expectedK: 0},
		{n: 25, expectedCap: 50, expectedLeft: 25, expectedK: 1},
		{n: 49, expectedCap: 50, expectedLeft: 1, expectedK: 1},
		{n: 50, expectedCap: 100, expectedLeft: 50, expectedK: 2},
		{n: 99, expectedCap: 100, expectedLeft: 1, expectedK: 2},
		{n: 100, expectedCap: 200, expectedLeft: 100, expectedK: 3},
		{n: 1000, expectedCap: 1600, expectedLeft: 600, expectedK: 6},
		{n: 1000000, expectedCap: 1638400, expectedLeft: 638400, expectedK: 16},
	}

	for _, tc := range prescribedCases {
		cap, spots := CalculateCapacityAndSpots(tc.n)
		if cap != tc.expectedCap {
			t.Fatalf("For N=%d: got capacity %d, want %d", tc.n, cap, tc.expectedCap)
		}
		if spots != tc.expectedLeft {
			t.Fatalf("For N=%d: got spots_left %d, want %d", tc.n, spots, tc.expectedLeft)
		}

		// Verify C strictly satisfies C = 25 * 2^k
		ratio := float64(cap) / 25.0
		k := math.Log2(ratio)
		if math.Abs(k-math.Round(k)) > 1e-9 {
			t.Fatalf("For N=%d: capacity %d is not of form 25 * 2^k (k=%f)", tc.n, cap, k)
		}
		if int(math.Round(k)) != tc.expectedK {
			t.Fatalf("For N=%d: expected k=%d, got k=%d", tc.n, tc.expectedK, int(math.Round(k)))
		}

		// Verify spots_left = C - N
		if spots != cap-tc.n {
			t.Fatalf("For N=%d: spots_left (%d) != C - N (%d)", tc.n, spots, cap-tc.n)
		}
	}

	// Negative N test cases
	negativeCases := []int64{-1, -10, -25, -50, -100, -1000000}
	for _, negN := range negativeCases {
		cap, spots := CalculateCapacityAndSpots(negN)
		// Capacity must default to C0 = 25 (k=0)
		if cap != 25 {
			t.Fatalf("For negative N=%d: expected capacity 25, got %d", negN, cap)
		}
		// Spots must not exceed C0 (clamped non-negative active students)
		if spots != 25 {
			t.Fatalf("For negative N=%d: expected clamped spots 25, got %d", negN, spots)
		}
		// C = 25 * 2^k with k=0
		ratio := float64(cap) / 25.0
		k := math.Log2(ratio)
		if math.Abs(k) > 1e-9 {
			t.Fatalf("For negative N=%d: k is %f, want 0", negN, k)
		}
	}

	// Property-based fuzzing: 50,000 random non-negative values up to 5,000,000
	r := rand.New(rand.NewSource(42))
	for i := 0; i < 50000; i++ {
		n := r.Int63n(5000000)
		cap, spots := CalculateCapacityAndSpots(n)

		// 1. C strictly satisfies C = 25 * 2^k
		ratio := float64(cap) / 25.0
		k := math.Log2(ratio)
		if math.Abs(k-math.Round(k)) > 1e-9 {
			t.Fatalf("Property failure: N=%d, cap=%d is not 25 * 2^k", n, cap)
		}

		// 2. spots_left strictly satisfies spots_left = C - N
		if spots != cap-n {
			t.Fatalf("Property failure: N=%d, spots=%d != cap-n (%d)", n, spots, cap-n)
		}

		// 3. Invariants: N < C and spots > 0
		if n >= cap {
			t.Fatalf("Property failure: N=%d >= cap=%d", n, cap)
		}
		if spots <= 0 {
			t.Fatalf("Property failure: spots=%d <= 0 for N=%d", spots, n)
		}
		if spots > cap {
			t.Fatalf("Property failure: spots=%d > cap=%d for N=%d", spots, cap, n)
		}
	}
}

// TestChallenger_ReceiptDeduplication_Adversarial tests receipt deduplication against adversarial vectors.
func TestChallenger_ReceiptDeduplication_Adversarial(t *testing.T) {
	receipts := make(map[string]*models.PaymentReceipt)
	var mu sync.Mutex

	handler := NewReceiptHandler(nil).WithCustomStore(
		func(fileHash, fileUniqueID string) (*models.PaymentReceipt, bool) {
			mu.Lock()
			defer mu.Unlock()
			for _, r := range receipts {
				if (fileHash != "" && r.FileHash == fileHash) || (fileUniqueID != "" && r.FileUniqueID == fileUniqueID) {
					return r, true
				}
			}
			return nil, false
		},
		func(receipt *models.PaymentReceipt) error {
			mu.Lock()
			defer mu.Unlock()
			receipt.ID = uint(len(receipts) + 1)
			key := receipt.FileUniqueID
			if key == "" {
				key = receipt.FileHash
			}
			receipts[key] = receipt
			return nil
		},
	)

	r := gin.New()
	r.POST("/api/v1/receipts/verify", handler.SubmitReceipt)

	// Sub-test 1: Deduplication by identical file_unique_id
	t.Run("Deduplication_By_FileUniqueID", func(t *testing.T) {
		uniqID := "AQAD_adversarial_unique_1001"
		body1, _ := json.Marshal(map[string]interface{}{
			"file_unique_id": uniqID,
			"telegram_id":    111,
			"username":       "alice",
			"amount":         6900.0,
			"tier":           "accelerator",
		})
		req1 := httptest.NewRequest("POST", "/api/v1/receipts/verify", bytes.NewReader(body1))
		req1.Header.Set("Content-Type", "application/json")
		w1 := httptest.NewRecorder()
		r.ServeHTTP(w1, req1)

		if w1.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created for first submission, got %d", w1.Code)
		}

		// Re-submitting identical file_unique_id must return 409 Conflict
		body2, _ := json.Marshal(map[string]interface{}{
			"file_unique_id": uniqID,
			"telegram_id":    222,
			"username":       "attacker",
			"amount":         6900.0,
			"tier":           "accelerator",
		})
		req2 := httptest.NewRequest("POST", "/api/v1/receipts/verify", bytes.NewReader(body2))
		req2.Header.Set("Content-Type", "application/json")
		w2 := httptest.NewRecorder()
		r.ServeHTTP(w2, req2)

		if w2.Code != http.StatusConflict {
			t.Fatalf("expected 409 Conflict for duplicate file_unique_id, got %d", w2.Code)
		}
		var res2 map[string]interface{}
		_ = json.Unmarshal(w2.Body.Bytes(), &res2)
		if res2["duplicate"] != true {
			t.Fatalf("expected duplicate=true, got %v", res2["duplicate"])
		}
	})

	// Sub-test 2: Deduplication by identical file_hash
	t.Run("Deduplication_By_FileHash", func(t *testing.T) {
		hashVal := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
		body1, _ := json.Marshal(map[string]interface{}{
			"file_hash":   hashVal,
			"telegram_id": 333,
			"username":    "bob",
			"amount":      2900.0,
			"tier":        "basic",
		})
		req1 := httptest.NewRequest("POST", "/api/v1/receipts/verify", bytes.NewReader(body1))
		req1.Header.Set("Content-Type", "application/json")
		w1 := httptest.NewRecorder()
		r.ServeHTTP(w1, req1)

		if w1.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created for first submission of hash, got %d", w1.Code)
		}

		// Duplicate hash submission must return 409 Conflict
		body2, _ := json.Marshal(map[string]interface{}{
			"file_hash":   hashVal,
			"telegram_id": 444,
			"username":    "fraudster",
			"amount":      2900.0,
			"tier":        "basic",
		})
		req2 := httptest.NewRequest("POST", "/api/v1/receipts/verify", bytes.NewReader(body2))
		req2.Header.Set("Content-Type", "application/json")
		w2 := httptest.NewRecorder()
		r.ServeHTTP(w2, req2)

		if w2.Code != http.StatusConflict {
			t.Fatalf("expected 409 Conflict for duplicate file_hash, got %d", w2.Code)
		}
	})

	// Sub-test 3: Whitespace handling in file_hash
	t.Run("Whitespace_Tolerance_And_Conflict", func(t *testing.T) {
		rawHash := "abcdef0123456789abcdef0123456789"
		body1, _ := json.Marshal(map[string]interface{}{
			"file_hash":   "  " + rawHash + "  ",
			"telegram_id": 555,
		})
		req1 := httptest.NewRequest("POST", "/api/v1/receipts/verify", bytes.NewReader(body1))
		req1.Header.Set("Content-Type", "application/json")
		w1 := httptest.NewRecorder()
		r.ServeHTTP(w1, req1)

		if w1.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d", w1.Code)
		}

		// Second submission without whitespace must conflict
		body2, _ := json.Marshal(map[string]interface{}{
			"file_hash":   rawHash,
			"telegram_id": 666,
		})
		req2 := httptest.NewRequest("POST", "/api/v1/receipts/verify", bytes.NewReader(body2))
		req2.Header.Set("Content-Type", "application/json")
		w2 := httptest.NewRecorder()
		r.ServeHTTP(w2, req2)

		if w2.Code != http.StatusConflict {
			t.Fatalf("expected 409 Conflict for trimmed hash duplicate, got %d", w2.Code)
		}
	})

	// Sub-test 4: Missing both hash and unique_id returns 400 Bad Request
	t.Run("Missing_Identifiers_Returns_400", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"file_hash":      "   ",
			"file_unique_id": "",
			"telegram_id":    777,
		})
		req := httptest.NewRequest("POST", "/api/v1/receipts/verify", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for empty hash/unique_id, got %d", w.Code)
		}
	})

	// Sub-test 5: Concurrent submission race condition
	t.Run("Concurrent_Duplicate_Race_Condition", func(t *testing.T) {
		sharedUniqID := "AQAD_concurrent_race_test_9999"
		concurrency := 20
		var wg sync.WaitGroup
		wg.Add(concurrency)

		statusCodes := make([]int, concurrency)
		for i := 0; i < concurrency; i++ {
			idx := i
			go func() {
				defer wg.Done()
				b, _ := json.Marshal(map[string]interface{}{
					"file_unique_id": sharedUniqID,
					"telegram_id":    int64(1000 + idx),
					"username":       "racer",
					"amount":         14900.0,
					"tier":           "vip",
				})
				rq := httptest.NewRequest("POST", "/api/v1/receipts/verify", bytes.NewReader(b))
				rq.Header.Set("Content-Type", "application/json")
				rc := httptest.NewRecorder()
				r.ServeHTTP(rc, rq)
				statusCodes[idx] = rc.Code
			}()
		}
		wg.Wait()

		createdCount := 0
		conflictCount := 0
		for _, code := range statusCodes {
			if code == http.StatusCreated {
				createdCount++
			} else if code == http.StatusConflict {
				conflictCount++
			} else {
				t.Errorf("unexpected status code in race: %d", code)
			}
		}

		if createdCount != 1 {
			t.Fatalf("expected exactly 1 Created (201) in concurrent submission, got %d", createdCount)
		}
		if conflictCount != concurrency-1 {
			t.Fatalf("expected %d Conflict (409) responses, got %d", concurrency-1, conflictCount)
		}
	})
}
