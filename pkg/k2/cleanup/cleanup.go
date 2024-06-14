package cleanup

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/klothoplatform/klotho/pkg/multierr"
	"go.uber.org/zap"
)

type Callback func(signal syscall.Signal) error

var callbacks []Callback

func OnKill(callback Callback) {
	callbacks = append(callbacks, callback)
}

func Execute(signal syscall.Signal) error {
	var merr multierr.Error

	for _, cb := range callbacks {
		if err := cb(signal); err != nil {
			merr.Append(err)
		}
	}
	return merr.ErrOrNil()
}

func InitializeHandler() {
	// Set up signal handling
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
		os.Exit(1)
	}()
}

func SignalProcessGroup(pid int, signal syscall.Signal) {
	zap.S().Infof("Sending %s signal to process group: %v", signal, pid)
	// Use the negative PID to signal the entire process group
	err := syscall.Kill(-pid, syscall.SIGTERM)
	if err != nil {
		zap.S().Errorf("Error sending %s to process group: %v", signal, err)
	}
}
