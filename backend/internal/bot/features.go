package bot

// Product feature switches.
//
// Learning tools (plan, status, submissions, audit credits, curator SLA) and
// Zoom bookings/reminders are intentionally paused. The implementation is kept
// in the repository behind these switches so it can be re-enabled without
// restoring deleted code.
const (
	learningToolsEnabled = false
	zoomToolsEnabled     = false

	// Growth features.
	recoveryEnabled    = true
	experimentsEnabled = true
)
