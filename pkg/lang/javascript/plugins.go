package javascript

import (
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"go.uber.org/zap"
)

type (
	JavascriptPlugins struct {
		Plugins []core.Plugin
	}
)

func NewJavascriptPlugins(cfg *config.Application, runtime Runtime) *JavascriptPlugins {
	return &JavascriptPlugins{
		Plugins: []core.Plugin{
			ExpressHandler{Config: cfg},
			NestJsHandler{Config: cfg},
			AddExecRuntimeFiles{runtime: runtime},
			Persist{runtime: runtime},
			Pubsub{runtime: runtime},
		},
	}

}

func (c JavascriptPlugins) Name() string { return "javascript" }

func (c JavascriptPlugins) Transform(result *core.CompilationResult, deps *core.Dependencies) error {
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
