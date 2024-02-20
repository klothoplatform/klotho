package logging

import (
	"fmt"
	"os"
	"strings"
	"time"

	prettyconsole "github.com/thessem/zap-prettyconsole"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type LogOpts struct {
	Verbose         bool
	CategoryLogsDir string
	Encoding        string
	DefaultLevels   map[string]zapcore.Level
}

func (opts LogOpts) NewLogger() *zap.Logger {
	var enc zapcore.Encoder
	leveller := zap.NewAtomicLevel()
	switch opts.Encoding {
	case "json":
		if opts.Verbose {
			enc = zapcore.NewJSONEncoder(zap.NewDevelopmentEncoderConfig())
		} else {
			enc = zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
		}
	case "console", "pretty_console", "":
		cfg := prettyconsole.NewEncoderConfig()
		cfg.EncodeTime = TimeOffsetFormatter(time.Now())
		enc = prettyconsole.NewEncoder(cfg)
	default:
		panic(fmt.Errorf("unknown encoding %q", opts.Encoding))
	}
	if opts.Verbose {
		leveller.SetLevel(zap.DebugLevel)
	} else {
		leveller.SetLevel(zap.InfoLevel)
	}

	core := zapcore.NewCore(enc, os.Stderr, leveller)

	levels := opts.DefaultLevels
	levelEnv, ok := os.LookupEnv("LOG_LEVEL")
	if ok {
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

	if levels != nil {
		core = NewEntryLeveller(core, levels)
	}
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

	return zap.New(core)
}

// TimeOffsetFormatter returns a time encoder that formats the time as an offset from the start time.
// This is mostly useful for CLI logging not long-standing services as times beyond a few minutes will
// be less readable.
func TimeOffsetFormatter(start time.Time) zapcore.TimeEncoder {
	const colStart = "\x1b[90m"
	const colEnd = "\x1b[0m"
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
