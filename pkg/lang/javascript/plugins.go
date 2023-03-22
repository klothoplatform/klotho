package javascript

import (
	"github.com/klothoplatform/klotho/pkg/compiler"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/graph"
	"go.uber.org/zap"
)

type (
	JavascriptPlugins struct {
		Plugins []compiler.AnalysisAndTransformationPlugin
	}
)

func NewJavascriptPlugins(cfg *config.Application, runtime Runtime) *JavascriptPlugins {
	return &JavascriptPlugins{
		Plugins: []compiler.AnalysisAndTransformationPlugin{
			ExpressHandler{Config: cfg},
			NestJsHandler{Config: cfg},
			AddExecRuntimeFiles{runtime: runtime},
			Persist{runtime: runtime},
			Pubsub{runtime: runtime},
		},
	}

}

func (c JavascriptPlugins) Name() string { return "javascript" }

func (c JavascriptPlugins) Transform(input *core.InputFiles, constructGraph *graph.Directed[core.Construct]) error {
	for _, p := range c.Plugins {
		log := zap.L().With(zap.String("plugin", p.Name()))
		log.Debug("starting")
		err := p.Transform(input, constructGraph)
		if err != nil {
			return core.NewPluginError(p.Name(), err)
		}
		log.Debug("completed")
	}

	return nil
}
