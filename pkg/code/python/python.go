package python

import (
	"bytes"
	"context"
	"io"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
	"go.uber.org/zap"
)

var lang = python.GetLanguage()

func NewParser() *sitter.Parser {
	parser := sitter.NewParser()
	parser.SetLanguage(lang)
	return parser
}

func ParseFile(ctx context.Context, f io.Reader) (*sitter.Tree, error) {
	content, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return NewParser().ParseCtx(ctx, nil, content)
}

type PyLSPLogger struct {
	Log *zap.Logger
}

func (p PyLSPLogger) Write(b []byte) (int, error) {
	s := string(bytes.TrimSpace(b))
	parts := strings.SplitN(s, " - ", 4)
	// 2021-08-25 14:00:00,000 - root - INFO - message
	// 0: date
	// 1: logger
	// 2: level
	// 3: message
	if len(parts) < 4 {
		p.Log.Debug(s)
		return len(b), nil
	}

	l := p.Log.Named(parts[2])
	for _, msg := range strings.Split(parts[3], "\n") {
		switch parts[1] {
		case "DEBUG":
			l.Debug(msg)
		case "INFO":
			l.Info(msg)
		case "WARNING":
			l.Warn(msg)
		case "ERROR", "FATAL", "CRITICAL":
			l.Error(msg)
		}
	}
	return len(b), nil
}
