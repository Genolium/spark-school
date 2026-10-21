package bot

import (
	"context"
	"fmt"
	"time"

	"github.com/spark-school/backend/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UpdateClaimStore coordinates exactly-once Telegram update processing across
// all application instances using PostgreSQL as the source of truth.
//
// A claim is inserted before any business action. If a worker crashes, the
// lease expires and another worker may take over. Completed updates can never
// be claimed again. Failed updates are retryable.
type UpdateClaimStore struct {
	db    *gorm.DB
	lease time.Duration
	now   func() time.Time
}

func NewUpdateClaimStore(db *gorm.DB, lease time.Duration) *UpdateClaimStore {
	if lease <= 0 {
		lease = 5 * time.Minute
	}
	return &UpdateClaimStore{db: db, lease: lease, now: time.Now}
}

// Claim returns claimed=true if this process is allowed to execute the update.
func (s *UpdateClaimStore) Claim(ctx context.Context, updateID int) (bool, error) {
	if updateID == 0 {
		// Synthetic updates (tests/tools) have no Telegram identity to persist.
		return true, nil
	}
	if s.db == nil {
		return false, fmt.Errorf("update claim store has no database")
	}

	now := s.now()
	leaseUntil := now.Add(s.lease)
	record := models.ProcessedTelegramUpdate{
		UpdateID:    updateID,
		Status:      "processing",
		Attempts:    1,
		LockedUntil: &leaseUntil,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	res := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "update_id"}},
		DoNothing: true,
	}).Create(&record)
	if res.Error != nil {
		return false, fmt.Errorf("claim telegram update %d: %w", updateID, res.Error)
	}
	if res.RowsAffected == 1 {
		return true, nil
	}

	// Existing row: take over only a failed update or an expired lease.
	res = s.db.WithContext(ctx).Model(&models.ProcessedTelegramUpdate{}).
		Where("update_id = ?", updateID).
		Where("status = ? OR (status = ? AND (locked_until IS NULL OR locked_until < ?))", "failed", "processing", now).
		Updates(map[string]interface{}{
			"status":       "processing",
			"locked_until": &leaseUntil,
			"attempts":     gorm.Expr("attempts + 1"),
			"updated_at":   now,
			"last_error":   "",
		})
	if res.Error != nil {
		return false, fmt.Errorf("take over telegram update %d: %w", updateID, res.Error)
	}
	return res.RowsAffected == 1, nil
}

func (s *UpdateClaimStore) Complete(ctx context.Context, updateID int) error {
	if updateID == 0 || s.db == nil {
		return nil
	}
	res := s.db.WithContext(ctx).Model(&models.ProcessedTelegramUpdate{}).
		Where("update_id = ? AND status = ?", updateID, "processing").
		Updates(map[string]interface{}{
			"status":       "completed",
			"locked_until": nil,
			"updated_at":   s.now(),
			"last_error":   "",
		})
	if res.Error != nil {
		return fmt.Errorf("complete telegram update %d: %w", updateID, res.Error)
	}
	if res.RowsAffected != 1 {
		return fmt.Errorf("complete telegram update %d: claim was lost", updateID)
	}
	return nil
}

func (s *UpdateClaimStore) Fail(ctx context.Context, updateID int, cause error) error {
	if updateID == 0 || s.db == nil {
		return nil
	}
	msg := ""
	if cause != nil {
		msg = cause.Error()
	}
	if len(msg) > 2000 {
		msg = msg[:2000]
	}
	res := s.db.WithContext(ctx).Model(&models.ProcessedTelegramUpdate{}).
		Where("update_id = ? AND status = ?", updateID, "processing").
		Updates(map[string]interface{}{
			"status":       "failed",
			"locked_until": nil,
			"updated_at":   s.now(),
			"last_error":   msg,
		})
	return res.Error
}

// Status returns the current persisted status, or "" if the update was never
// claimed. It is used only to shape the HTTP response for duplicates.
func (s *UpdateClaimStore) Status(ctx context.Context, updateID int) (string, error) {
	if updateID == 0 || s.db == nil {
		return "", nil
	}
	var record models.ProcessedTelegramUpdate
	if err := s.db.WithContext(ctx).Select("status").Where("update_id = ?", updateID).First(&record).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", nil
		}
		return "", err
	}
	return record.Status, nil
}
