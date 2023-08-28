package python

import (
	"github.com/klothoplatform/klotho/pkg/compiler"
	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/construct"
	"go.uber.org/zap"
)

type (
	PythonPlugins struct {
		Plugins []compiler.AnalysisAndTransformationPlugin
	}
)

func NewPythonPlugins(cfg *config.Application, runtime Runtime) *PythonPlugins {
	return &PythonPlugins{
		Plugins: []compiler.AnalysisAndTransformationPlugin{
			&Expose{},
			&AddExecRuntimeFiles{cfg: cfg, runtime: runtime},
			&Persist{runtime: runtime},
		},
	}
}

func (c PythonPlugins) Name() string { return "python" }

func (c PythonPlugins) Transform(input *types.InputFiles, fileDeps *types.FileDependencies, constructGraph *construct.ConstructGraph) error {
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
