package csharp

import (
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"go.uber.org/zap"
)

type (
	PythonPlugins struct {
		Plugins []core.Plugin
	}
)

func NewCSharpPlugins(cfg *config.Application, runtime Runtime) *PythonPlugins {
	return &PythonPlugins{
		Plugins: []core.Plugin{},
	}
}

func (c PythonPlugins) Name() string { return "python" }

func (c PythonPlugins) Transform(result *core.CompilationResult, deps *core.Dependencies) error {
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
