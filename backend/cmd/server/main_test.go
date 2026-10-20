package main

import (
	"sync"
	"testing"
)

func TestUpdateDeduplicator_LRU(t *testing.T) {
	capacity := 5
	dedup := NewUpdateDeduplicator(nil, capacity)

	// Initially, no ID is processed
	for id := 1; id <= 10; id++ {
		if dedup.IsProcessed(id) {
			t.Errorf("expected ID %d to be unobserved initially", id)
		}
	}

	// Zero ID should always return false
	if dedup.IsProcessed(0) {
		t.Errorf("expected 0 ID to be false")
	}

	// Add 5 IDs: 1, 2, 3, 4, 5
	for id := 1; id <= 5; id++ {
		dedup.MarkProcessed(id)
	}

	// All 5 should be present
	for id := 1; id <= 5; id++ {
		if !dedup.IsProcessed(id) {
			t.Errorf("expected ID %d to be marked as processed", id)
		}
	}

	// Add 6th ID: should evict oldest (1)
	dedup.MarkProcessed(6)
	if dedup.IsProcessed(1) {
		t.Errorf("expected ID 1 to have been evicted by LRU capacity limit")
	}
	if !dedup.IsProcessed(6) {
		t.Errorf("expected ID 6 to be present")
	}
}

func TestUpdateDeduplicator_ConcurrentRace(t *testing.T) {
	capacity := 100
	dedup := NewUpdateDeduplicator(nil, capacity)

	var wg sync.WaitGroup
	workers := 10
	iterations := 100

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				updateID := (workerID * iterations) + i
				_ = dedup.IsProcessed(updateID)
				dedup.MarkProcessed(updateID)
				if !dedup.IsProcessed(updateID) {
					// Might be evicted if capacity reached, which is fine
				}
			}
		}(w)
	}

	wg.Wait()
}

func TestUpdateDeduplicator_TryClaim_Parallel(t *testing.T) {
	dedup := NewUpdateDeduplicator(nil, 50)
	targetUpdateID := 9999

	var wg sync.WaitGroup
	var claimedCount int
	var mu sync.Mutex

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			claimed, release := dedup.TryClaim(targetUpdateID)
			if claimed {
				mu.Lock()
				claimedCount++
				mu.Unlock()
				// Hold the claim briefly
				defer release(true)
			}
		}()
	}

	wg.Wait()

	if claimedCount != 1 {
		t.Fatalf("expected exactly 1 worker to claim update %d, got %d", targetUpdateID, claimedCount)
	}

	if !dedup.IsProcessed(targetUpdateID) {
		t.Errorf("expected update %d to be marked processed after release(true)", targetUpdateID)
	}
}

func TestUpdateDeduplicator_TryClaim_FailureRetry(t *testing.T) {
	dedup := NewUpdateDeduplicator(nil, 50)
	targetUpdateID := 8888

	claimed1, release1 := dedup.TryClaim(targetUpdateID)
	if !claimed1 {
		t.Fatalf("first claim must succeed")
	}

	// While in flight, second claim fails
	claimed2, _ := dedup.TryClaim(targetUpdateID)
	if claimed2 {
		t.Fatalf("second claim while in-flight must be rejected")
	}

	// Execution fails: release(false)
	release1(false)

	// Since it failed, it must NOT be marked as processed, and can be retried!
	if dedup.IsProcessed(targetUpdateID) {
		t.Fatalf("update %d must not be processed after release(false)", targetUpdateID)
	}

	claimed3, release3 := dedup.TryClaim(targetUpdateID)
	if !claimed3 {
		t.Fatalf("retry claim must succeed after previous failure release")
	}

	// Execution succeeds
	release3(true)

	if !dedup.IsProcessed(targetUpdateID) {
		t.Fatalf("update %d must be processed after release(true)", targetUpdateID)
	}
}
