package closenicely

import (
	"go.uber.org/zap"
	"io"
)

func OrDebug(closer io.Closer) {
	FuncOrDebug(closer.Close)
}

func FuncOrDebug(closer func() error) {
	if err := closer(); err != nil {
		zap.L().Debug("Failed to close resource", zap.Error(err))
	}
}
