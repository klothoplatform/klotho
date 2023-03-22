package csharp

import (
	"github.com/klothoplatform/klotho/pkg/compiler"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/graph"
	"go.uber.org/zap"
)

type (
	CSharpPlugins struct {
		Plugins []compiler.AnalysisAndTransformationPlugin
	}
)

func NewCSharpPlugins(cfg *config.Application, runtime Runtime) *CSharpPlugins {
	return &CSharpPlugins{
		Plugins: []compiler.AnalysisAndTransformationPlugin{
			&Expose{},
			&AddExecRuntimeFiles{
				runtime: runtime,
				cfg:     cfg,
			},
		},
	}
}

func (c CSharpPlugins) Name() string { return "C#" }

func (c CSharpPlugins) Transform(input *core.InputFiles, constructGraph *graph.Directed[core.Construct]) error {
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
