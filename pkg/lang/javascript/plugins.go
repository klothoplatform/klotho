package javascript

import (
	"github.com/klothoplatform/klotho/pkg/compiler"
	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/construct"
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

func (c JavascriptPlugins) Transform(input *types.InputFiles, fileDeps *types.FileDependencies, constructGraph *construct.ConstructGraph) error {
	for _, p := range c.Plugins {
		log := zap.L().With(zap.String("plugin", p.Name()))
		log.Debug("starting")
		err := p.Transform(input, fileDeps, constructGraph)
		if err != nil {
			return types.NewPluginError(p.Name(), err)
		}
		log.Debug("completed")
	}

	return nil
}
