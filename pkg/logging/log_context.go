package logging

import (
	"context"

	"go.uber.org/zap"
)

type contextKey string

var logKey contextKey = "log"

func GetLogger(ctx context.Context) *zap.Logger {
	l := ctx.Value(logKey)
	if l == nil {
		return zap.L()
	}
	return l.(*zap.Logger)
}

func WithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, logKey, logger)
}
