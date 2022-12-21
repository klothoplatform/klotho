package analytics

// This file is basically cribbed from zapcore/hook.go, but with the `funcs` modified to take the []Field fields as well
// as the main entry.

import (
	"github.com/klothoplatform/klotho/pkg/logging"
	"go.uber.org/zap/zapcore"
)

type fieldListener struct {
	zapcore.LevelEnabler
	client *Client
	fields []zapcore.Field
}

func (client *Client) NewFieldListener(level zapcore.LevelEnabler) zapcore.Core {
	return &fieldListener{
		LevelEnabler: level,
		client:       client,
	}
}

func (fl *fieldListener) With(fields []zapcore.Field) zapcore.Core {
	allFields := make([]zapcore.Field, 0, len(fields)+len(fl.fields))
	allFields = append(allFields, fl.fields...)
	allFields = append(allFields, fields...)
	return &fieldListener{
		LevelEnabler: fl.LevelEnabler,
		client:       fl.client,
		fields:       allFields,
	}
}

func (fl *fieldListener) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if fl.Enabled(entry.Level) {
		return ce.AddCore(entry, fl)
	}
	return ce
}

func (fl *fieldListener) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	allFields := make([]zapcore.Field, 0, len(fields)+len(fl.fields))
	allFields = append(allFields, fl.fields...)
	allFields = append(allFields, fields...)

	safeLogsStr := logging.SanitizeFields(allFields, fl.client.Hash)
	switch entry.Level {
	case zapcore.DebugLevel:
		fl.client.Debug(safeLogsStr)
	case zapcore.InfoLevel:
		fl.client.Info(safeLogsStr)
	case zapcore.WarnLevel:
		fl.client.Warn(safeLogsStr)
	case zapcore.ErrorLevel:
		fl.client.Error(safeLogsStr)
	case zapcore.DPanicLevel, zapcore.PanicLevel, zapcore.FatalLevel:
		fl.client.Panic(safeLogsStr)
	default:
		fl.client.Warn(safeLogsStr) // shouldn't happen, but just to be safe
	}
	return nil
}

func (fl *fieldListener) Sync() error {
	return nil
}
