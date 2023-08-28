package execunit

import (
	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/construct"
)

type ExecUnitPlugin struct {
	Config *config.Application
}

func (p ExecUnitPlugin) Name() string { return "ExecutionUnit" }

func (p ExecUnitPlugin) Transform(input *types.InputFiles, fileDeps *types.FileDependencies, constructGraph *construct.ConstructGraph) error {

	unit := &types.ExecutionUnit{Name: "main",
		Executable: types.NewExecutable(),
	}
	cfg := p.Config.GetExecutionUnit(unit.Name)

	for key, value := range cfg.EnvironmentVariables {
		unit.EnvironmentVariables.Add(types.NewEnvironmentVariable(key, nil, value))
	}

	// This set of environment variables is added to all Execution Units
	unit.EnvironmentVariables.Add(types.NewEnvironmentVariable("APP_NAME", nil, p.Config.AppName))
	unit.EnvironmentVariables.Add(types.NewEnvironmentVariable("EXECUNIT_NAME", nil, unit.Name))

	for _, f := range input.Files() {
		if sf, ok := f.(*types.SourceFile); ok {
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
