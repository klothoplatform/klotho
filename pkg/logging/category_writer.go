package logging

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"go.uber.org/zap/zapcore"
)

type CategoryWriter struct {
	Encoder     zapcore.Encoder
	LogRootPath string
	files       *sync.Map // map[string]io.Writer
}

func NewCategoryWriter(enc zapcore.Encoder, logRootPath string) *CategoryWriter {
	return &CategoryWriter{
		Encoder:     enc,
		LogRootPath: logRootPath,
		files:       &sync.Map{},
	}
}

func (c *CategoryWriter) Enabled(lvl zapcore.Level) bool {
	return true
}

func (c *CategoryWriter) With(fields []zapcore.Field) zapcore.Core {
	clone := c.clone()
	for i := range fields {
		fields[i].AddTo(clone.Encoder)
	}
	return clone
}

func (c *CategoryWriter) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

func (c *CategoryWriter) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	if ent.LoggerName == "" {
		return nil
	}
	categ, rest, _ := strings.Cut(ent.LoggerName, ".")
	categ = strings.TrimSpace(categ)
	categ = strings.ReplaceAll(categ, string(os.PathSeparator), "_")
	if categ == "" {
		return nil
	}
	ent.LoggerName = rest // trim the category from the logger name for better readability
	w, ok := c.files.Load(categ)
	if !ok {
		err := os.MkdirAll(c.LogRootPath, 0755)
		if err != nil {
			return err
		}

		logPath := filepath.Join(c.LogRootPath, categ+".log")
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		var loaded bool
		w, loaded = c.files.LoadOrStore(categ, f)
		if loaded {
			f.Close()
		} else {
			// Don't pass `O_TRUNC` to the open in case we open the file simultaneously.
			// Wait until we know for sure that the one we opened is the cannonical one in the map.
			if _, err := f.Seek(0, io.SeekStart); err != nil {
				return err
			}
			if err := f.Truncate(0); err != nil {
				return err
			}
		}
	}

	buf, err := c.Encoder.EncodeEntry(ent, fields)
	if err != nil {
		return err
	}
	_, err = w.(io.Writer).Write(buf.Bytes())
	buf.Free()
	if err != nil {
		return err
	}
	if ent.Level > zapcore.ErrorLevel {
		if syncer, ok := w.(interface{ Sync() error }); ok {
			syncer.Sync() //nolint:errcheck
		}
	}

	return nil
}

func (c *CategoryWriter) Sync() error {
	var errs error
	c.files.Range(func(key, value interface{}) bool {
		if syncer, ok := value.(interface{ Sync() error }); ok {
			errs = errors.Join(errs, syncer.Sync())
		}
		return false
	})
	return errs
}

func (c *CategoryWriter) clone() *CategoryWriter {
	return &CategoryWriter{
		Encoder:     c.Encoder.Clone(),
		LogRootPath: c.LogRootPath,
		files:       c.files,
	}
}
