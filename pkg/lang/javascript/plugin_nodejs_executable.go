package javascript

import (
	"encoding/json"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/klothoplatform/klotho/pkg/construct"
	klotho_errors "github.com/klothoplatform/klotho/pkg/errors"
	execunit "github.com/klothoplatform/klotho/pkg/exec_unit"
	"github.com/klothoplatform/klotho/pkg/filter"
	"github.com/klothoplatform/klotho/pkg/io"
	"github.com/klothoplatform/klotho/pkg/logging"
	"go.uber.org/zap"
)

var upstreamDependencyResolver = execunit.SourceFilesResolver{
	UnitFileDependencyResolver: UnitFileDependencyResolver,
	UpstreamAnnotations:        []string{annotation.ExposeCapability},
}

type NodeJSExecutable struct {
}

func (l NodeJSExecutable) Name() string {
	return "nodejs_executable"
}

func (l NodeJSExecutable) Transform(input *types.InputFiles, fileDeps *types.FileDependencies, constructGraph *construct.ConstructGraph) error {
	// TODO: Consider adding ES module config for a unit in this plugin
	inputFiles := input.Files()

	defaultPackageJson, _ := inputFiles["package.json"].(*PackageFile)
	for _, unit := range construct.GetConstructsOfType[*types.ExecutionUnit](constructGraph) {
		if unit.Executable.Type != "" {
			zap.L().Sugar().Debugf("Skipping exececution unit '%s': executable type is already set to '%s'", unit.Name, unit.Executable.Type)
			continue
		}

		packageJson := defaultPackageJson
		packageJsonPath := types.CheckForProjectFile(input, unit, "package.json")
		if packageJsonPath != "" {
			packageJson, _ = inputFiles[packageJsonPath].(*PackageFile)
		}
		if packageJson == nil {
			zap.L().Sugar().Debugf("package.json not found in execution_unit: %s", unit.Name)
			return nil
		}

		unit.AddResource(packageJson.Clone())
		unit.Executable.Type = types.ExecutableTypeNodeJS

		var err error
		for _, file := range unit.FilesOfLang(js) {
			for _, annot := range file.Annotations() {
				cap := annot.Capability
				if cap.Name == annotation.ExecutionUnitCapability && cap.ID == unit.Name {
					unit.AddEntrypoint(file)
				}
			}
		}

		if len(unit.Executable.Entrypoints) == 0 {
			err = addEntrypointFromPackageJson(packageJson, unit)
			if err != nil {
				return klotho_errors.WrapErrf(err, "entrypoint resolution from package.json failed for execution unit: %s", unit.Name)
			}
		}

		if len(unit.Executable.Entrypoints) == 0 {
			resolveDefaultEntrypoint(unit)
		}

		err = refreshSourceFiles(unit)
		if err != nil {
			return err
		}
		refreshUpstreamEntrypoints(unit)
	}
	return nil
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
		return klotho_errors.WrapErrf(err, "file dependency resolution failed for execution unit: %s", unit.Name)
	}
	for k, v := range sourceFiles {
		unit.Executable.SourceFiles[k] = v
		warnIfContainsES6Import(unit.Get(k))
	}
	return err
}

func warnIfContainsES6Import(file io.File) {
	jsF, ok := Language.ID.CastFile(file)
	if !ok {
		return
	}

	for _, p := range FindImportsInFile(jsF).Filter(filter.NewSimpleFilter(IsImportOfKind(ImportKindES))) {
		if p.Kind == ImportKindES {
			zap.L().Sugar().With(logging.FileField(jsF), logging.NodeField(p.ImportNode)).Warn(
				"ES6 import statements are not yet supported: please use CommonJS 'require()' syntax instead")
		}
	}
}

func resolveDefaultEntrypoint(unit *types.ExecutionUnit) {
	if indexJs := unit.Get("index.js"); indexJs != nil {
		zap.L().Sugar().Debugf("Adding execution unit entrypoint: [default] -> [%s] -> %s", unit.Name, indexJs.Path())
		unit.AddEntrypoint(indexJs)
	}
}

func addEntrypointFromPackageJson(packageJson *PackageFile, unit *types.ExecutionUnit) error {
	// if no other roots are detected, add the file indicated in the unit's package.json#main field
	if mainRaw, ok := packageJson.Content.OtherFields["main"]; ok {
		main := ""
		err := json.Unmarshal(mainRaw, &main)
		if err != nil {
			return klotho_errors.WrapErrf(err, "could not unmarshal 'main' from package.json")
		}
		if mainFileR := unit.Get(main); mainFileR != nil {
			if mainFile, ok := mainFileR.(*types.SourceFile); ok {
				unit.AddEntrypoint(mainFile)
				zap.L().Sugar().Debugf("Adding execution unit entrypoint: [package.json#main] -> [%s] -> %s", unit.Name, mainFile.Path())
			}
		}
	}
	return nil
}
