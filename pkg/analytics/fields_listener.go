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

	logLevel := Warn
	switch entry.Level {
	case zapcore.DebugLevel:
		logLevel = Debug
	case zapcore.InfoLevel:
		logLevel = Info
	case zapcore.WarnLevel:
		logLevel = Warn
	case zapcore.ErrorLevel:
		logLevel = Error
	case zapcore.DPanicLevel, zapcore.PanicLevel, zapcore.FatalLevel:
		logLevel = Panic
	}

	p := fl.client.createPayload(logLevel, entry.Level.CapitalString())

	for k, v := range logging.SanitizeFields(allFields, fl.client.Hash) {
		p.Properties[fmt.Sprintf("log.%s", k)] = v
	}

	for _, f := range allFields {
		if f.Key == logging.EntryMessageField {
			p.Event += (" " + entry.Message)
		}
		if err, isError := f.Interface.(error); isError {
			p.addError(err)
		}
	}

	fl.client.Send(p)

	return nil
}

func (fl *fieldListener) Sync() error {
	return nil
}
