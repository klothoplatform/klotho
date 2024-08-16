package tui

import "go.uber.org/zap/zapcore"

type Verbosity int

var (
	VerbosityConcise   Verbosity = 0
	VerbosityVerbose   Verbosity = 1
	VerbosityDebug     Verbosity = 2
	VerbosityDebugMore Verbosity = 3
)

func (v Verbosity) LogLevel() zapcore.Level {
	switch v {
	case VerbosityConcise:
		return zapcore.ErrorLevel

	case VerbosityVerbose:
		return zapcore.InfoLevel

	case VerbosityDebug:
		return zapcore.DebugLevel

	default:
		return zapcore.DebugLevel
	}
}

// CombineLogs controls whether to show all logs commingled in the TUI.
// In other words, sorted by timestamp, not grouped by construct.
func (v Verbosity) CombineLogs() bool {
	return VerbosityDebugMore == v
}
