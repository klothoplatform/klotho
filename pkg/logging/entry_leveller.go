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

func (el *EntryLeveller) Check(e zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if e.LoggerName == "" {
		return el.Core.Check(e, ce)
	}
	if level, ok := el.levels.Load(e.LoggerName); ok {
		if e.Level < level.(zapcore.Level) {
			return nil
		}
		return ce.AddCore(e, el)
	}
	nameParts := strings.Split(e.LoggerName, ".")
	for i := len(nameParts); i > 0; i-- {
		module := strings.Join(nameParts[:i], ".")
		if level, ok := el.levels.Load(module); ok {
			el.levels.Store(e.LoggerName, level)
			if e.Level < level.(zapcore.Level) {
				return nil
			}
			return ce.AddCore(e, el)
		}
	}
	return el.Core.Check(e, ce)
}
