package analytics

// This file is basically cribbed from zapcore/hook.go, but with the `funcs` modified to take the []Field fields as well
// as the main entry.

import (
	"fmt"
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

	message := entry.Level.CapitalString()

	for k, v := range logging.SanitizeFields(allFields, fl.client.Hash) {
		property := fmt.Sprintf("log.%s", k)
		fl.client.Properties[property] = v
		defer func() { fl.client.DeleteProperty(property) }()
	}

	for _, f := range allFields {
		if f.Key == logging.EntryMessageField {
			message += (" " + entry.Message)
		}
		if err, isError := f.Interface.(error); isError {
			fl.client.Properties["error"] = fmt.Sprintf("%+v", err)
			defer fl.client.DeleteProperty("error")
		}
	}

	switch entry.Level {
	case zapcore.DebugLevel:
		fl.client.Debug(message)
	case zapcore.InfoLevel:
		fl.client.Info(message)
	case zapcore.WarnLevel:
		fl.client.Warn(message)
	case zapcore.ErrorLevel:
		fl.client.Error(message)
	case zapcore.DPanicLevel, zapcore.PanicLevel, zapcore.FatalLevel:
		fl.client.Panic(message)
	default:
		fl.client.Warn(message) // shouldn't happen, but just to be safe
	}
	return nil
}

func (fl *fieldListener) Sync() error {
	return nil
}
