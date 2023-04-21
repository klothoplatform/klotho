package python

import (
	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/core"
	execunit "github.com/klothoplatform/klotho/pkg/exec_unit"
	"go.uber.org/zap"
)

var upstreamDependencyResolver = execunit.SourceFilesResolver{
	UnitFileDependencyResolver: UnitFileDependencyResolver,
	UpstreamAnnotations:        []string{annotation.ExposeCapability},
}

type PythonExecutable struct {
}

func (l PythonExecutable) Name() string {
	return "python_executable"
}

func (l PythonExecutable) Transform(input *core.InputFiles, fileDeps *core.FileDependencies, constructGraph *core.ConstructGraph) error {
	inputFiles := input.Files()

	defaultRequirementsTxt, _ := inputFiles["requirements.txt"].(*RequirementsTxt)
	for _, unit := range core.GetConstructsOfType[*core.ExecutionUnit](constructGraph) {
		if unit.Executable.Type != "" {
			zap.L().Sugar().Debugf("Skipping exececution unit '%s': executable type is already set to '%s'", unit.ID, unit.Executable.Type)
			continue
		}

		requirementsTxt := defaultRequirementsTxt
		requirementsTxtPath := core.CheckForProjectFile(input, unit, "requirements.txt")
		if requirementsTxtPath != "" {
			requirementsTxt, _ = inputFiles[requirementsTxtPath].(*RequirementsTxt)
		}
		if requirementsTxt == nil {
			zap.L().Sugar().Debugf("requirements.txt not found in execution_unit: %s", unit.ID)
			return nil
		}

		unit.AddResource(requirementsTxt.Clone())
		unit.Executable.Type = core.ExecutableTypePython

		for _, file := range unit.FilesOfLang(py) {
			for _, annot := range file.Annotations() {
				cap := annot.Capability
				if cap.Name == annotation.ExecutionUnitCapability && cap.ID == unit.ID {
					unit.AddEntrypoint(file)
				}
			}
		}

		if len(unit.Executable.Entrypoints) == 0 {
			resolveDefaultEntrypoint(unit)
		}

		err := refreshSourceFiles(unit)
		if err != nil {
			return err
		}
		refreshUpstreamEntrypoints(unit)
	}
	return nil
}

func refreshUpstreamEntrypoints(unit *core.ExecutionUnit) {
	for f := range unit.Executable.SourceFiles {
		if file, ok := unit.Get(f).(*core.SourceFile); ok && file.IsAnnotatedWith(annotation.ExposeCapability) {
			zap.L().Sugar().Debugf("Adding execution unit entrypoint: [@klotho::expose] -> [%s] -> %s", unit.ID, f)
			unit.AddEntrypoint(file)
		}
	}
}

func refreshSourceFiles(unit *core.ExecutionUnit) error {
	sourceFiles, err := upstreamDependencyResolver.Resolve(unit)
	if err != nil {
		return core.WrapErrf(err, "file dependency resolution failed for execution unit: %s", unit.ID)
	}
	for k, v := range sourceFiles {
		unit.Executable.SourceFiles[k] = v
	}
	return err
}

func resolveDefaultEntrypoint(unit *core.ExecutionUnit) {
	for _, fallbackPath := range []string{"main.py", "app/main.py", "app.py", "app/app.py"} {
		if entrypoint := unit.Get(fallbackPath); entrypoint != nil {
			zap.L().Sugar().Debugf("Adding execution unit entrypoint: [default] -> [%s] -> %s", unit.ID, entrypoint.Path())
			unit.AddEntrypoint(entrypoint)
		}
	}
}
