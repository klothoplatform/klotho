package execunit

import (
	"errors"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
)

type ExecUnitPlugin struct {
	Config *config.Application
}

func (p ExecUnitPlugin) Name() string { return "ExecutionUnit" }

func (p ExecUnitPlugin) Transform(result *core.CompilationResult, deps *core.Dependencies) error {
	inputR := result.GetFirstResource(core.InputFilesKind)
	if inputR == nil {
		return errors.New("no input files")
	}

	unit := &core.ExecutionUnit{
		Name:       "main",
		Executable: core.NewExecutable(),
	}
	unit.ExecType = p.Config.GetExecutionUnit(unit.Name).Type

	for _, f := range inputR.(*core.InputFiles).Files() {
		if _, ok := f.(*core.SourceFile); ok {
			// Only add source files by default.
			// Plugins are responsible for adding in non-source files
			// as required by its features.
			unit.Add(f.Clone())
		}
	}

	if len(unit.Files()) > 0 {
		result.Add(unit)
	}

	return nil
}
