package closenicely

import "go.uber.org/zap"

type CloserWithError interface {
	Close() error
}

func OrDebug(closer CloserWithError) {
	if err := closer.Close(); err != nil {
		zap.L().Debug("Failed to close resource", zap.Error(err))
	}
}
