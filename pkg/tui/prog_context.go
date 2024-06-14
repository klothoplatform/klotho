package tui

import (
	"context"

	"github.com/klothoplatform/klotho/pkg/logging"
)

type contextKey string

var progressKey contextKey = "progress"

func GetProgress(ctx context.Context) Progress {
	p := ctx.Value(progressKey)
	if p == nil {
		return LogProgress{Logger: logging.GetLogger(ctx)}
	}
	return p.(Progress)
}

func WithProgress(ctx context.Context, progress Progress) context.Context {
	return context.WithValue(ctx, progressKey, progress)
}
