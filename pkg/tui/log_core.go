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
	for _, field := range f {
		if field.Key == "construct" {
			nc.construct = field.String
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

	c.program.Send(LogMessage{
		Construct: construct,
		Message:   s,
	})

	buf.Free()
	return nil
}
