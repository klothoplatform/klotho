package debug

import (
	"context"
	"os"
)

type contextKey string

var debugDirKey contextKey = "debugDir"

func GetDebugDir(ctx context.Context) string {
	d := ctx.Value(debugDirKey)
	if d == nil {
		return os.Getenv("KLOTHO_DEBUG_DIR")
	}
	return d.(string)
}

func WithDebugDir(ctx context.Context, debugDir string) context.Context {
	return context.WithValue(ctx, debugDirKey, debugDir)
}
