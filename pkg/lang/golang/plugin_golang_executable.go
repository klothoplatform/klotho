package golang

import (
	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/graph"
	"go.uber.org/zap"
)

type GolangExecutable struct {
}

func (l GolangExecutable) Name() string {
	return "golang_executable"
}

func (l GolangExecutable) Transform(input *core.InputFiles, constructGraph *graph.Directed[core.Construct]) error {
	inputFiles := input.Files()

	defaultGoMod, _ := input.Files()["go.mod"].(*GoMod)
	for _, unit := range core.GetResourcesOfType[*core.ExecutionUnit](constructGraph) {
		if unit.Executable.Type != "" {
			zap.L().Sugar().Debugf("Skipping exececution unit '%s': executable type is already set to '%s'", unit.ID, unit.Executable.Type)
			continue
		}

		goMod := defaultGoMod
		goModPath := core.CheckForProjectFile(input, unit, "go.mod")
		if goModPath != "" {
			goMod, _ = inputFiles[goModPath].(*GoMod)
		}
		if goMod == nil {
			zap.L().Sugar().Debugf("go.mod not found in execution_unit: %s", unit.ID)
			return nil
		}

		unit.AddResource(goMod.Clone())
		unit.Executable.Type = core.ExecutableTypeGolang

		// TODO: get sourceFiles using a dependency resolver once we can generate FileDependencies for Golang
		sourceFiles := unit.FilesOfLang(goLang)
		for _, f := range sourceFiles {
			unit.AddSourceFile(f)
		}

		for f := range unit.Executable.SourceFiles {
			if file, ok := unit.Get(f).(*core.SourceFile); ok && file.IsAnnotatedWith(annotation.ExposeCapability) {
				zap.L().Sugar().Debugf("Adding execution unit entrypoint: [@klotho::expose] -> [%s] -> %s", unit.ID, f)
				unit.AddEntrypoint(file)
			}
		}

		if len(unit.Executable.Entrypoints) == 0 {
			resolveDefaultEntrypoint(unit)
		}
	}
	return nil
}

func resolveDefaultEntrypoint(unit *core.ExecutionUnit) {
	for _, fallbackPath := range []string{"main.go"} {
		if entrypoint := unit.Get(fallbackPath); entrypoint != nil {
			zap.L().Sugar().Debugf("Adding execution unit entrypoint: [default] -> [%s] -> %s", unit.ID, entrypoint.Path())
			unit.AddEntrypoint(entrypoint)
		}
	}
}
