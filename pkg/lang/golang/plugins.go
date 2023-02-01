package golang

import (
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"go.uber.org/zap"
)

type (
	GoPlugins struct {
		Plugins []core.Plugin
	}
)

func NewGoPlugins(cfg *config.Application, runtime Runtime) *GoPlugins {
	return &GoPlugins{
		Plugins: []core.Plugin{
			&Expose{Config: cfg},
			&AddExecRuntimeFiles{cfg: cfg, runtime: runtime},
		},
	}
}

func (c GoPlugins) Name() string { return "go" }

func (c GoPlugins) Transform(result *core.CompilationResult, deps *core.Dependencies) error {
	for _, p := range c.Plugins {
		log := zap.L().With(zap.String("plugin", p.Name()))
		log.Debug("starting")
		err := p.Transform(result, deps)
		if err != nil {
			return core.NewPluginError(p.Name(), err)
		}
		log.Debug("completed")
	}

	return nil
}
