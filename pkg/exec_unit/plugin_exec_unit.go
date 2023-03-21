package execunit

import (
	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/graph"
)

type ExecUnitPlugin struct {
	Config *config.Application
}

func (p ExecUnitPlugin) Name() string { return "ExecutionUnit" }

func (p ExecUnitPlugin) Transform(input *core.InputFiles, constructGraph *graph.Directed[core.Construct]) error {

	unit := &core.ExecutionUnit{
		AnnotationKey: core.AnnotationKey{
			ID:         "main",
			Capability: annotation.ExecutionUnitCapability,
		},
		Executable: core.NewExecutable(),
	}
	cfg := p.Config.GetExecutionUnit(unit.ID)

	for key, value := range cfg.EnvironmentVariables {
		unit.EnvironmentVariables.Add(core.NewEnvironmentVariable(key, nil, value))
	}

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
		constructGraph.AddVertex(unit)
	}

	return nil
}
