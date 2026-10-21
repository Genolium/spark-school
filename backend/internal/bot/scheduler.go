package bot

import (
	"fmt"
	"os"
	"time"

	"gorm.io/gorm"
)

// acquireSchedulerLease ensures that only one application instance runs a
// background job at a time. Leases expire, so a crashed owner does not block
// the job forever.
func (s *BotService) acquireSchedulerLease(name string, ttl time.Duration) bool {
	if s.db == nil {
		return false
	}
	owner := fmt.Sprintf("%s-%d", hostname(), os.Getpid())
	now := time.Now()
	res := s.db.Exec(`
		INSERT INTO scheduler_leases (name, owner, expires_at)
		VALUES (?, ?, ?)
		ON CONFLICT (name) DO UPDATE
		SET owner = EXCLUDED.owner, expires_at = EXCLUDED.expires_at
		WHERE scheduler_leases.expires_at < ?`,
		name, owner, now.Add(ttl), now)
	if res.Error != nil {
		if res.Error == gorm.ErrRecordNotFound {
			return false
		}
		return false
	}
	return res.RowsAffected == 1
}

func hostname() string {
	h, err := os.Hostname()
	if err != nil || h == "" {
		return "unknown"
	}
	return h
}
