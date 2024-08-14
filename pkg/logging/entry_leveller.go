package logging

import (
	"strings"
	"sync"

	"go.uber.org/zap/zapcore"
)

// EntryLeveller is a zapcore.Core that filters log entries based on the module name
// similar to Log4j or python's logging module.
type EntryLeveller struct {
	zapcore.Core

	levels sync.Map // map[string]zapcore.Level
}

func NewEntryLeveller(core zapcore.Core, levels map[string]zapcore.Level) *EntryLeveller {
	el := &EntryLeveller{Core: core}
	for k, v := range levels {
		el.levels.Store(k, v)
	}
	return el
}

func (el *EntryLeveller) With(f []zapcore.Field) zapcore.Core {
	next := &EntryLeveller{
		Core: el.Core.With(f),
	}
	el.levels.Range(func(k, v interface{}) bool {
		next.levels.Store(k, v)
		return true
	})
	return next
}

func (el *EntryLeveller) checkModule(e zapcore.Entry, ce *zapcore.CheckedEntry, module string) (*zapcore.CheckedEntry, bool) {
	if level, ok := el.levels.Load(module); ok {
		el.levels.Store(e.LoggerName, level)
		if e.Level < level.(zapcore.Level) {
			return nil, true
		}
		return ce.AddCore(e, el), true
	}
	return nil, false
}

func (el *EntryLeveller) Check(e zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if ce, ok := el.checkModule(e, ce, e.LoggerName); ok {
		return ce
	}
	if e.LoggerName == "" {
		return el.Core.Check(e, ce)
	}

	nameParts := strings.Split(e.LoggerName, ".")
	for i := len(nameParts); i > 0; i-- {
		module := strings.Join(nameParts[:i], ".")
		if ce, ok := el.checkModule(e, ce, module); ok {
			return ce
		}
	}
	if ce, ok := el.checkModule(e, ce, ""); ok {
		return ce
	}
	return el.Core.Check(e, ce)
}
