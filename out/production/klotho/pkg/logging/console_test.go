package logging

import (
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"strings"
	"testing"
)

func TestEncodings(t *testing.T) {
	cases := []struct {
		name    string
		given   logInput
		expect  result
		logFunc func(logger *zap.Logger, message string, fields ...zap.Field)
	}{
		{
			name:  "no fields",
			given: log("hello world"),
			expect: result{
				message:        "hello world\n",
				verboseMessage: "  info hello world\n",
			},
		},
		{
			name:  "plain error field",
			given: log("hello world", zap.Error(fmt.Errorf("my cool error"))),
			expect: result{
				message:        "hello world\n| ERROR: my cool error\n",
				verboseMessage: "  info hello world\n      | ERROR: my cool error\n",
			},
		},
		{
			name:  "error field with stack",
			given: log("hello world", zap.Error(errors.New("my cool error with stack trace"))),
			expect: result{
				message:        "hello world\n| ERROR: my cool error with stack trace\n",
				verboseMessage: "  info hello world\n      | ERROR: my cool error with stack trace\n{{TRACE}}",
			},
		},
		{
			name:    "debug message",
			given:   log("boom"),
			logFunc: (*zap.Logger).Warn,
			expect: result{
				message:        "boom\n",
				verboseMessage: "  warn boom\n",
				warningsFound:  true,
			},
		},
		{
			name:    "error message",
			given:   log("boom"),
			logFunc: (*zap.Logger).Error,
			expect: result{
				message:        "boom\n",
				verboseMessage: " error boom\n",
				warningsFound:  true,
				errorsFound:    true,
			},
		},
		// note: don't test (*zap.Logger).Fatal, because it always calls os.Exit(1)
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			for _, verbose := range []bool{false, true} {
				t.Run(fmt.Sprintf("verbose=%v", verbose), func(t *testing.T) {
					assert := assert.New(t)

					var hadWarnings, hadErrors atomic.Bool
					encoder := NewConsoleEncoder(verbose, &hadWarnings, &hadErrors)

					buf := &bytes.Buffer{}
					logger := zap.New(
						zapcore.NewCore(encoder, zapcore.AddSync(buf), zap.DebugLevel),
						zap.AddCaller(),
						zap.AddCallerSkip(1),
					)

					if tt.logFunc == nil {
						tt.logFunc = (*zap.Logger).Info
					}
					tt.logFunc(logger, tt.given.message, tt.given.fields...)

					var expectMessage string
					if verbose {
						expectMessage = tt.expect.verboseMessage
					} else {
						expectMessage = tt.expect.message
					}

					// for each field that's an Error with a stack trace, replace one instance of "{{TRACE}}" with
					// that stack trace
					for _, field := range tt.given.fields {
						var indent string
						if verbose {
							indent = "      "
						}
						trace := getStackTrace(field, indent)
						if trace != "" {
							expectMessage = strings.Replace(expectMessage, "{{TRACE}}", trace, 1)
						}
					}

					assert.Equal(expectMessage, buf.String())
					assert.Equal(tt.expect.warningsFound, hadWarnings.Load())
					assert.Equal(tt.expect.errorsFound, hadErrors.Load())
				})
			}
		})
	}
}

// getStackTrace gets the stack trace, but only if the field is an error that has a stack trace.
// This makes for a less transparent test, but it's a necessary evil, since the stack trace is dynamic (depending even
// on GOROOT, etc.)
func getStackTrace(field zap.Field, indent string) string {
	if field.Type != zapcore.ErrorType {
		return ""
	}

	type stackTracer interface {
		StackTrace() errors.StackTrace
	}
	trace, hasTrace := field.Interface.(stackTracer)
	if !hasTrace {
		return ""
	}

	var result string
	for _, frame := range trace.StackTrace() {
		for _, frameLine := range strings.Split(fmt.Sprintf("%+v", frame), "\n") {
			result += fmt.Sprintf("%s| %+v\n", indent, frameLine)
		}
	}
	return result
}

type (
	logInput struct {
		message string
		fields  []zap.Field
	}

	result struct {
		message        string
		warningsFound  bool
		errorsFound    bool
		verboseMessage string
	}
)

func log(message string, fields ...zap.Field) logInput {
	return logInput{
		message: message,
		fields:  fields,
	}
}
