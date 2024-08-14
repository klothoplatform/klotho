package cleanup

import (
	"context"
	"errors"
	"os"
	"strings"
	"syscall"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"go.uber.org/zap/zaptest/observer"
)

func TestOnKill(t *testing.T) {
	callbacks = nil // Clear existing callbacks

	mockCallback := func(signal syscall.Signal) error {
		return nil
	}

	OnKill(mockCallback)
	if len(callbacks) != 1 {
		t.Fatalf("expected 1 callback, got %d", len(callbacks))
	}
}

func TestExecute(t *testing.T) {
	callbacks = nil // Clear existing callbacks

	err1 := syscall.Errno(1)
	err2 := syscall.Errno(2)

	callbacks = append(callbacks, func(signal syscall.Signal) error {
		return err1
	})
	callbacks = append(callbacks, func(signal syscall.Signal) error {
		return err2
	})

	err := Execute(syscall.SIGTERM)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}

	if !errors.Is(err, err1) || !errors.Is(err, err2) {
		t.Fatalf("expected error to contain err1 and err2, got %v", err)
	}
}

func TestInitializeHandler(t *testing.T) {
	ctx := context.Background()
	ctx = InitializeHandler(ctx)

	select {
	case <-ctx.Done():
		t.Fatal("context should not be done yet")
	default:
	}

	proc, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("failed to find process: %v", err)
	}
	if proc == nil {
		t.Fatal("process is nil")
	}

	err = syscall.Kill(proc.Pid, syscall.SIGTERM)
	if err != nil {
		t.Fatalf("failed to send SIGTERM: %v", err)
	}

	<-ctx.Done() // Should be done after signal
}

func TestSignalProcessGroup(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)
	logger := zaptest.NewLogger(t, zaptest.WrapOptions(zap.Hooks(func(entry zapcore.Entry) error {
		if err := core.Write(entry, nil); err != nil {
			return err
		}
		return nil
	})))
	zap.ReplaceGlobals(logger)

	proc, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("failed to find process: %v", err)
	}
	if proc == nil {
		t.Fatal("process is nil")
	}

	SignalProcessGroup(proc.Pid, syscall.SIGTERM)

	logEntries := logs.TakeAll()
	if len(logEntries) != 2 {
		t.Fatalf("expected 2 log entries, got %d", len(logEntries))
	}
	if !strings.Contains(logEntries[0].Message, "Sending terminated signal to process group") {
		t.Fatalf("expected first log to contain 'Sending terminated signal to process group', got %s", logEntries[0].Message)
	}
	if !strings.Contains(logEntries[1].Message, "Error sending terminated to process group") {
		t.Fatalf("expected second log to contain 'Error sending terminated to process group', got %s", logEntries[1].Message)
	}
}
