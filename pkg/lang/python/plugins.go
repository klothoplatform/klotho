package python

import (
	"github.com/klothoplatform/klotho/pkg/compiler"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
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

func (c PythonPlugins) Transform(input *core.InputFiles, constructGraph *core.ConstructGraph) error {
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
