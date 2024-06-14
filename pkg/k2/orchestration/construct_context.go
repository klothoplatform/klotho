package orchestration

import (
	"context"
	"path/filepath"

	"github.com/klothoplatform/klotho/pkg/engine/debug"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/klothoplatform/klotho/pkg/logging"
)

func ConstructContext(ctx context.Context, construct model.URN) context.Context {
	ctx = logging.WithLogger(ctx, logging.GetLogger(ctx).Named(construct.ResourceID))
	ctx = debug.WithDebugDir(ctx, filepath.Join(debug.GetDebugDir(ctx), construct.ResourceID))
	return ctx
}
