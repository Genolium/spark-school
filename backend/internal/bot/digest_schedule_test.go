package bot

import (
	"testing"
	"time"
)

func TestShouldSendDailyDigest_UsesMoscowWallClock(t *testing.T) {
	// 21:00 UTC is 00:00 MSK. The old scheduler used the container clock and
	// sent the digest exactly at midnight, reporting an empty Moscow day.
	utcEvening := time.Date(2026, 10, 25, 21, 0, 0, 0, time.UTC)
	if shouldSendDailyDigest(utcEvening, 21, "") {
		t.Fatal("digest must follow Moscow time, not the UTC container clock")
	}

	mskEvening := time.Date(2026, 10, 26, 21, 0, 0, 0, mskZone)
	if !shouldSendDailyDigest(mskEvening, 21, "") {
		t.Fatal("expected the digest at 21:00 MSK")
	}
	if shouldSendDailyDigest(mskEvening, 21, "2026-10-26") {
		t.Fatal("digest must fire at most once per Moscow day")
	}
	if shouldSendDailyDigest(mskEvening.Add(time.Minute), 21, "") {
		t.Fatal("digest must fire only at the top of the hour")
	}
}
