package csharp

import (
	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	execunit "github.com/klothoplatform/klotho/pkg/exec_unit"
	"github.com/klothoplatform/klotho/pkg/lang/csharp/csproj"
	"go.uber.org/zap"
)

var upstreamDependencyResolver = execunit.SourceFilesResolver{
	UnitFileDependencyResolver: func(unit *core.ExecutionUnit) (core.FileDependencies, error) {
		return core.FileDependencies{}, nil // TODO: implement file dependency resolution for C#
	},
	UpstreamAnnotations: []string{annotation.ExposeCapability},
}

type CSharpExecutable struct {
	Config *config.Application
}

func (l CSharpExecutable) Name() string {
	return "csharp_executable"
}

func (l CSharpExecutable) Transform(input *core.InputFiles, fileDeps *core.FileDependencies, constructGraph *core.ConstructGraph) error {
	for _, unit := range core.GetResourcesOfType[*core.ExecutionUnit](constructGraph) {
		if unit.Executable.Type != "" {
			zap.L().Sugar().Debugf("Skipping exececution unit '%s': executable type is already set to '%s'", unit.ID, unit.Executable.Type)
			continue
		}

		var csProjFile *csproj.CSProjFile
		for _, file := range input.Files() {
			if casted, ok := file.(*csproj.CSProjFile); ok {
				csProjFile = casted
				break
			}
		}
		if csProjFile == nil {
			zap.L().Sugar().Debugf(`"MSBuild Project File (.csproj)" not found in execution_unit: %s`, unit.ID)
			return nil
		}

		unit.AddResource(csProjFile.Clone())
		unit.Executable.Type = core.ExecutableTypeCSharp

		// TODO: get sourceFiles using a dependency resolver once we can generate FileDependencies for C#
		var err error
		sourceFiles := unit.FilesOfLang(CSharp)
		for _, file := range sourceFiles {
			unit.AddSourceFile(file)
			for _, annot := range file.Annotations() {
				cap := annot.Capability
				if cap.Name == annotation.ExecutionUnitCapability && cap.ID == unit.ID {
					zap.L().Sugar().Debugf("Adding execution unit entrypoint: [@klotho::expose] -> [%s] -> %s", unit.ID, file.Path())
					unit.AddEntrypoint(file)
				}
			}
		}

		if len(unit.Executable.Entrypoints) == 0 {
			l.resolveDefaultEntrypoint(unit)
		}

		err = refreshSourceFiles(unit)
		if err != nil {
			return err
		}
		refreshUpstreamEntrypoints(unit)
	}
	return nil
}

func (l CSharpExecutable) resolveDefaultEntrypoint(unit *core.ExecutionUnit) {
	for _, fallbackPath := range []string{l.Config.AppName + ".cs", "Program.cs", "Application.cs"} {
		if entrypoint := unit.Get(fallbackPath); entrypoint != nil {
			zap.L().Sugar().Debugf("Adding execution unit entrypoint: [default] -> [%s] -> %s", unit.ID, entrypoint.Path())
			unit.AddEntrypoint(entrypoint)
		}
	}
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
