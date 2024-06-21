package tui

type Verbosity int

var (
	VerbosityConcise   Verbosity = 0
	VerbosityVerbose   Verbosity = 1
	VerbosityDebug     Verbosity = 2
	VerbosityDebugMore Verbosity = 3
)

// DebugLogs controls zap logging verbosity, true = debug, false = info
func (v Verbosity) DebugLogs() bool {
	return v >= 2
}

// ShowLogs controls whether to show logs in the TUI
func (v Verbosity) ShowLogs() bool {
	return v >= 1
}

// CombineLogs controls whether to show all logs comingled in the TUI. In other words,
// sorted by timestamp, not grouped by construct.
func (v Verbosity) CombineLogs() bool {
	return v >= 3
}
