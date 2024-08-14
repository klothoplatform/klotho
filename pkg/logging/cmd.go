package logging

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"os/exec"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type loggerWriter struct {
	logger *zap.Logger
	level  zapcore.Level
}

type CommandLogger struct {
	RootLogger  *zap.Logger
	StdoutLevel zapcore.Level
	StderrLevel zapcore.Level
}

func NewLoggerWriter(logger *zap.Logger, level zapcore.Level) io.Writer {
	return loggerWriter{logger: logger, level: level}
}

func (w loggerWriter) Write(p []byte) (n int, err error) {
	var lines []string
	if bytes.Contains(p, []byte{'\n'}) {
		lineBytes := bytes.Split(p, []byte{'\n'})
		lines = make([]string, 0, len(lineBytes))
		for _, line := range lineBytes {
			if len(line) != 0 {
				lines = append(lines, string(line))
			}
		}
	} else {
		lines = []string{string(p)}
	}
	for _, line := range lines {
		if ce := w.logger.Check(w.level, line); ce != nil {
			ce.Write()
		}
	}
	return len(p), nil
}

func (w loggerWriter) ReadFrom(r io.Reader) (int64, error) {
	buf := bufio.NewScanner(r)
	var n int64
	for buf.Scan() {
		txt := buf.Text()
		if ce := w.logger.Check(w.level, txt); ce != nil {
			ce.Write()
		}
		n += int64(len(txt))
	}
	return n, buf.Err()
}

func Command(ctx context.Context, cfg CommandLogger, name string, arg ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, arg...)
	cmd.Stdout = &loggerWriter{logger: cfg.RootLogger.Named("stdout"), level: cfg.StdoutLevel}
	cmd.Stderr = &loggerWriter{logger: cfg.RootLogger.Named("stderr"), level: cfg.StderrLevel}
	return cmd
}
