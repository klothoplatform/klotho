package cleanup

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"go.uber.org/zap"
)

type Callback func(signal syscall.Signal) error

var callbacks []Callback
var callbackMu sync.Mutex

func OnKill(callback Callback) {
	callbackMu.Lock()
	defer callbackMu.Unlock()

	callbacks = append(callbacks, callback)
}

func Execute(signal syscall.Signal) error {
	callbackMu.Lock()
	defer callbackMu.Unlock()

	var errs []error

	for _, cb := range callbacks {
		if err := cb(signal); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func InitializeHandler(ctx context.Context) context.Context {
	ctx, cancel := context.WithCancel(ctx)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	// Handle termination signals
	go func() {
		sig := <-sigCh
		zap.S().Infof("Received signal: %v", sig)
		err := Execute(sig.(syscall.Signal))
		if err != nil {
			zap.S().Errorf("Error running executing cleanup: %v", err)
		}
		cancel()
	}()
	return ctx
}

func SignalProcessGroup(pid int, signal syscall.Signal) {
	zap.S().Infof("Sending %s signal to process group: %v", signal, pid)
	// Use the negative PID to signal the entire process group
	err := syscall.Kill(-pid, syscall.SIGTERM)
	if err != nil {
		zap.S().Errorf("Error sending %s to process group: %v", signal, err)
	}
}
