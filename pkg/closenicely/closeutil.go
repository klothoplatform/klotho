package closenicely

import (
	"go.uber.org/zap"
	"io"
)

func OrDebug(closer io.Closer) {
	if err := closer.Close(); err != nil {
		zap.L().Debug("Failed to close resource", zap.Error(err))
	}
}
