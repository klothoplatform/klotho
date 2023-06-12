package execunit

import (
	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
)

type ExecUnitPlugin struct {
	Config *config.Application
}

func (p ExecUnitPlugin) Name() string { return "ExecutionUnit" }

func (p ExecUnitPlugin) Transform(input *core.InputFiles, fileDeps *core.FileDependencies, constructGraph *core.ConstructGraph) error {

	unit := &core.ExecutionUnit{Name: "main",
		Executable: core.NewExecutable(),
	}
	cfg := p.Config.GetExecutionUnit(unit.Name)

	for key, value := range cfg.EnvironmentVariables {
		unit.EnvironmentVariables.Add(core.NewEnvironmentVariable(key, nil, value))
	}

	// This set of environment variables is added to all Execution Units
	unit.EnvironmentVariables.Add(core.NewEnvironmentVariable("APP_NAME", nil, p.Config.AppName))
	unit.EnvironmentVariables.Add(core.NewEnvironmentVariable("EXECUNIT_NAME", nil, unit.Name))

	for _, f := range input.Files() {
		if sf, ok := f.(*core.SourceFile); ok {
			// Only add source files by default.
			// Plugins are responsible for adding in non-source files
			// as required by its features.
			if sf.IsAnnotatedWith(annotation.ExecutionUnitCapability) {
				unit.AddSourceFile(f.Clone())
			} else {
				unit.Add(f.Clone())
			}
		}
	}

	if len(unit.Files()) > 0 {
		constructGraph.AddConstruct(unit)
	}

	return nil
}
