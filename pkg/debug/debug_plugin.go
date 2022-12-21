package debug

import (
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"go.uber.org/zap"
)

// DebugPlugin is a simple plugin useful for debugging a compilation. Insert it anywhere in the compilation
// and it will print out debug statements about the current state of the config, results, and dependencies.
type DebugPlugin struct {
	Config *config.Application
}

func (p DebugPlugin) Name() string { return "Debug" }

func (p DebugPlugin) Transform(result *core.CompilationResult, deps *core.Dependencies) error {
	l := zap.S()

	l.Debugf("Config: %+v", p.Config)

	if result.Len() == 0 {
		l.Debug("Res: []")
	}

	for _, res := range result.Resources() {
		l.Debugf("Res: %+v", res)
	}

	l.Debugf("Deps: %s", deps)

	return nil
}
