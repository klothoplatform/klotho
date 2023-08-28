package csharp

import (
	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/errors"
	execunit "github.com/klothoplatform/klotho/pkg/exec_unit"
	"github.com/klothoplatform/klotho/pkg/lang/csharp/csproj"
	"go.uber.org/zap"
)

var upstreamDependencyResolver = execunit.SourceFilesResolver{
	UnitFileDependencyResolver: func(unit *types.ExecutionUnit) (types.FileDependencies, error) {
		return types.FileDependencies{}, nil // TODO: implement file dependency resolution for C#
	},
	UpstreamAnnotations: []string{annotation.ExposeCapability},
}

type CSharpExecutable struct {
	Config *config.Application
}

func (l CSharpExecutable) Name() string {
	return "csharp_executable"
}

func (l CSharpExecutable) Transform(input *types.InputFiles, fileDeps *types.FileDependencies, constructGraph *construct.ConstructGraph) error {
	for _, unit := range construct.GetConstructsOfType[*types.ExecutionUnit](constructGraph) {
		if unit.Executable.Type != "" {
			zap.L().Sugar().Debugf("Skipping exececution unit '%s': executable type is already set to '%s'", unit.Name, unit.Executable.Type)
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
			zap.L().Sugar().Debugf(`"MSBuild Project File (.csproj)" not found in execution_unit: %s`, unit.Name)
			return nil
		}

		unit.AddResource(csProjFile.Clone())
		unit.Executable.Type = types.ExecutableTypeCSharp

		// TODO: get sourceFiles using a dependency resolver once we can generate FileDependencies for C#
		var err error
		sourceFiles := unit.FilesOfLang(CSharp)
		for _, file := range sourceFiles {
			unit.AddSourceFile(file)
			for _, annot := range file.Annotations() {
				cap := annot.Capability
				if cap.Name == annotation.ExecutionUnitCapability && cap.ID == unit.Name {
					zap.L().Sugar().Debugf("Adding execution unit entrypoint: [@klotho::expose] -> [%s] -> %s", unit.Name, file.Path())
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

func (l CSharpExecutable) resolveDefaultEntrypoint(unit *types.ExecutionUnit) {
	for _, fallbackPath := range []string{l.Config.AppName + ".cs", "Program.cs", "Application.cs"} {
		if entrypoint := unit.Get(fallbackPath); entrypoint != nil {
			zap.L().Sugar().Debugf("Adding execution unit entrypoint: [default] -> [%s] -> %s", unit.Name, entrypoint.Path())
			unit.AddEntrypoint(entrypoint)
		}
	}
}
func refreshUpstreamEntrypoints(unit *types.ExecutionUnit) {
	for f := range unit.Executable.SourceFiles {
		if file, ok := unit.Get(f).(*types.SourceFile); ok && file.IsAnnotatedWith(annotation.ExposeCapability) {
			zap.L().Sugar().Debugf("Adding execution unit entrypoint: [@klotho::expose] -> [%s] -> %s", unit.Name, f)
			unit.AddEntrypoint(file)
		}
	}
}

func refreshSourceFiles(unit *types.ExecutionUnit) error {
	sourceFiles, err := upstreamDependencyResolver.Resolve(unit)
	if err != nil {
		return errors.WrapErrf(err, "file dependency resolution failed for execution unit: %s", unit.Name)
	}
	for k, v := range sourceFiles {
		unit.Executable.SourceFiles[k] = v
	}
	return err
}
