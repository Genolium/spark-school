package bot

// Product feature switches.
//
// Learning tools (plan, status, submissions, audit credits, curator SLA) are
// intentionally paused. The implementation is kept in the repository behind
// this switch so it can be re-enabled without restoring deleted code.
const (
	learningToolsEnabled = false

	// Growth features.
	recoveryEnabled    = true
	experimentsEnabled = true
)
