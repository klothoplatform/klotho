package closenicely

import (
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"io"
	"syscall"
)

func OrDebug(closer io.Closer) {
	FuncOrDebug(closer.Close)
}

func FuncOrDebug(closer func() error) {
	// zap.Logger.Sync() always returns a syscall.ENOTTY error when logging to stdout
	// see: https://github.com/uber-go/zap/issues/991#issuecomment-962098428
	if err := closer(); err != nil && !errors.Is(err, syscall.ENOTTY) {
		zap.L().Debug("Failed to close resource", zap.Error(err))
	}
}
