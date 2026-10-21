package database

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// EnsureAnalyticsPartitions creates monthly partitions for the current month
// and the next 18 months. The default partition catches anything outside the
// range, so a missed schedule cannot lose events.
func EnsureAnalyticsPartitions(db *gorm.DB, now time.Time) error {
	if db == nil {
		return nil
	}
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -1, 0)
	for i := 0; i < 20; i++ {
		from := start.AddDate(0, i, 0)
		to := from.AddDate(0, 1, 0)
		table := fmt.Sprintf("analytics_events_%s", from.Format("2006_01"))
		stmt := fmt.Sprintf(
			"CREATE TABLE IF NOT EXISTS %s PARTITION OF analytics_events FOR VALUES FROM ('%s') TO ('%s')",
			table, from.Format("2006-01-02"), to.Format("2006-01-02"),
		)
		if err := db.Exec(stmt).Error; err != nil {
			return fmt.Errorf("create analytics partition %s: %w", table, err)
		}
	}
	return nil
}
