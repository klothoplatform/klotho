package logging

import (
	"fmt"
	"os"
	"strings"
	"time"

	prettyconsole "github.com/thessem/zap-prettyconsole"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/term"
)

type LogOpts struct {
	Verbose         bool
	Color           string
	CategoryLogsDir string
	Encoding        string
	DefaultLevels   map[string]zapcore.Level
}

func (opts LogOpts) Encoder() zapcore.Encoder {
	switch opts.Encoding {
	case "json":
		if opts.Verbose {
			return zapcore.NewJSONEncoder(zap.NewDevelopmentEncoderConfig())
		} else {
			return zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
		}
	case "console", "pretty_console", "":
		useColor := true
		switch opts.Color {
		case "auto":
			useColor = term.IsTerminal(int(os.Stderr.Fd()))
		case "always", "on":
			useColor = true
		case "never", "off":
			useColor = false
		}

		if useColor {
			cfg := prettyconsole.NewEncoderConfig()
			cfg.EncodeTime = TimeOffsetFormatter(time.Now(), useColor)
			return prettyconsole.NewEncoder(cfg)
		}
		cfg := zap.NewDevelopmentEncoderConfig()
		cfg.EncodeTime = TimeOffsetFormatter(time.Now(), useColor)
		return zapcore.NewConsoleEncoder(cfg)
	default:
		panic(fmt.Errorf("unknown encoding %q", opts.Encoding))
	}
}

func (opts LogOpts) EntryLeveller(core zapcore.Core) zapcore.Core {
	levels := opts.DefaultLevels
	if levelEnv, ok := os.LookupEnv("LOG_LEVEL"); ok {
		values := strings.Split(levelEnv, ",")
		levels = make(map[string]zapcore.Level, len(values))
		for _, v := range values {
			k, v, ok := strings.Cut(v, "=")
			if !ok {
				continue
			}
			lvl, err := zapcore.ParseLevel(v)
			if err != nil {
				continue
			}
			levels[k] = lvl
		}
	}

	if len(levels) > 0 {
		core = NewEntryLeveller(core, levels)
	}
	return core
}

func (opts LogOpts) CategoryCore(core zapcore.Core) zapcore.Core {
	if opts.CategoryLogsDir != "" {
		var categEnc zapcore.Encoder
		switch opts.Encoding {
		case "json":
			categEnc = zapcore.NewJSONEncoder(zap.NewDevelopmentEncoderConfig())
		case "console", "pretty_console", "":
			categEnc = zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
		default:
			panic(fmt.Errorf("unknown encoding %q", opts.Encoding))
		}
		core = zapcore.NewTee(
			core,
			NewCategoryWriter(categEnc, opts.CategoryLogsDir),
		)
	}
	return core
}

func (opts LogOpts) NewCore(w zapcore.WriteSyncer) zapcore.Core {
	enc := opts.Encoder()

	leveller := zap.NewAtomicLevel()
	if opts.Verbose {
		leveller.SetLevel(zap.DebugLevel)
	} else {
		leveller.SetLevel(zap.InfoLevel)
	}

	core := zapcore.NewCore(enc, w, leveller)
	core = opts.EntryLeveller(core)
	core = opts.CategoryCore(core)
	return core
}

func (opts LogOpts) NewLogger() *zap.Logger {
	return zap.New(opts.NewCore(os.Stderr))
}

// TimeOffsetFormatter returns a time encoder that formats the time as an offset from the start time.
// This is mostly useful for CLI logging not long-standing services as times beyond a few minutes will
// be less readable.
func TimeOffsetFormatter(start time.Time, color bool) zapcore.TimeEncoder {
	var colStart = "\x1b[90m"
	var colEnd = "\x1b[0m"
	if !color {
		colStart = ""
		colEnd = ""
	}
	return func(t time.Time, e zapcore.PrimitiveArrayEncoder) {
		diff := t.Sub(start)
		if diff < time.Second {
			e.AppendString(fmt.Sprintf(" %s%3dms%s", colStart, diff.Milliseconds(), colEnd))
		} else if diff < 5*time.Minute {
			e.AppendString(fmt.Sprintf("%s%5.1fs%s", colStart, diff.Seconds(), colEnd))
		} else {
			e.AppendString(fmt.Sprintf("%s%5.1fm%s", colStart, diff.Minutes(), colEnd))
		}
	}
}
