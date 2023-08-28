package csharp

import (
	"github.com/klothoplatform/klotho/pkg/compiler"
	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/construct"
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

func (c CSharpPlugins) Transform(input *types.InputFiles, fileDeps *types.FileDependencies, constructGraph *construct.ConstructGraph) error {
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
