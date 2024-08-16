package tui

import (
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/klothoplatform/klotho/pkg/logging"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type LogCore struct {
	zapcore.Core
	verbosity Verbosity
	program   *tea.Program
	enc       zapcore.Encoder

	construct string
}

func NewLogCore(opts logging.LogOpts, verbosity Verbosity, program *tea.Program) zapcore.Core {
	enc := opts.Encoder()
	leveller := zap.NewAtomicLevel()
	leveller.SetLevel(verbosity.LogLevel())

	core := zapcore.NewCore(enc, os.Stderr, leveller)
	core = &LogCore{
		Core:      core,
		verbosity: verbosity,
		program:   program,
		enc:       enc,
	}
	core = opts.EntryLeveller(core)
	core = opts.CategoryCore(core)
	return core
}

func (c *LogCore) With(f []zapcore.Field) zapcore.Core {
	nc := *c
	nc.Core = c.Core.With(f)
	nc.enc = c.enc.Clone()
	for _, field := range f {
		if field.Key == "construct" {
			nc.construct = field.String

			if c.verbosity.CombineLogs() {
				field.AddTo(nc.enc)
			}
			// else (if the field is the construct, and we're not combining logs) don't add it to the encoder
			// because the log lines will already be in its own construct section of the output.
		} else {
			field.AddTo(nc.enc)
		}
	}
	return &nc
}

func (c *LogCore) Check(e zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(e.Level) {
		return ce.AddCore(e, c)
	}
	return ce
}

func (c *LogCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	if c.verbosity.CombineLogs() {
		buf, err := c.enc.EncodeEntry(ent, fields)
		if err != nil {
			return err
		}
		s := buf.String()
		s = strings.TrimSuffix(s, "\n")
		c.program.Println(s)
		buf.Free()
		return nil
	}

	construct := c.construct
	nonConstructFields := make([]zapcore.Field, 0, len(fields))
	for _, f := range fields {
		if f.Key == "construct" {
			construct = f.String
		} else {
			nonConstructFields = append(nonConstructFields, f)
		}
	}

	buf, err := c.enc.EncodeEntry(ent, nonConstructFields)
	if err != nil {
		return err
	}
	s := buf.String()
	s = strings.TrimSuffix(s, "\n")

	if c.construct == "" && zapcore.ErrorLevel.Enabled(ent.Level) {
		c.program.Send(ErrorMessage{
			Message: s,
		})
		buf.Free()
		return nil
	}

	c.program.Send(LogMessage{
		Construct: construct,
		Message:   s,
	})

	buf.Free()
	return nil
}
