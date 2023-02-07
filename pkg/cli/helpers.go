package cli

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	execunit "github.com/klothoplatform/klotho/pkg/exec_unit"
	"github.com/klothoplatform/klotho/pkg/infra/kubernetes"
	"github.com/klothoplatform/klotho/pkg/lang"
	"github.com/klothoplatform/klotho/pkg/logging"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func OutputAST(input *core.InputFiles, outDir string) error {
	for _, efile := range input.Files() {
		path := filepath.Join(outDir, "input", efile.Path())
		err := os.MkdirAll(filepath.Dir(path), 0777)
		if err != nil {
			return errors.Wrapf(err, "could not create dirs for %s", efile.Path())
		}

		if astFile, ok := efile.(*core.SourceFile); ok {
			astPath := strings.TrimSuffix(path, filepath.Ext(path)) + ".ast.json"
			f, err := os.OpenFile(astPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0777)
			if err != nil {
				return errors.Wrapf(err, "could not create ast file %s", efile.Path())
			}
			err = lang.WriteAST(astFile.Tree().RootNode(), f)
			f.Close()
			if err != nil {
				return errors.Wrapf(err, "could not write ast file content for %s", efile.Path())
			}
			zap.L().Debug("Wrote file", logging.FileField(&core.RawFile{FPath: astPath}))
		}
	}
	return nil
}

func OutputCapabilities(input *core.InputFiles, outDir string) error {
	for _, efile := range input.Files() {
		path := filepath.Join(outDir, "input", efile.Path())
		err := os.MkdirAll(filepath.Dir(path), 0777)
		if err != nil {
			return errors.Wrapf(err, "could not create dirs for %s", efile.Path())
		}
		capPath := strings.TrimSuffix(path, filepath.Ext(path)) + ".caps.json"
		f, err := os.OpenFile(capPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0777)
		if err != nil {
			return errors.Wrapf(err, "could not create caps file %s", efile.Path())
		}
		if astFile, ok := efile.(*core.SourceFile); ok {
			err = lang.PrintCapabilities(astFile.Annotations(), f)
		}
		f.Close()
		if err != nil {
			return errors.Wrapf(err, "could not write caps file content for %s", efile.Path())
		}
		zap.L().Debug("Wrote file", logging.FileField(&core.RawFile{FPath: capPath}))
	}
	return nil
}

func OutputResources(result *core.CompilationResult, outDir string) (resourceCounts map[string]int, err error) {
	err = os.MkdirAll(outDir, 0777)
	if err != nil {
		return
	}

	resourceCounts = make(map[string]int)
	var resourcesOutput []interface{}
	var merr multierr.Error
	for _, res := range result.Resources() {
		resourceCounts[res.Key().Kind] = resourceCounts[res.Key().Kind] + 1

		switch r := res.(type) {
		case *core.ExecutionUnit:
			resOut := map[string]interface{}{
				"Type": r.Key().Kind,
				"Name": r.Key().Name,
			}
			var files []string
			for _, f := range r.Files() {
				files = append(files, f.Path())
			}
			resOut["Files"] = files
			resourcesOutput = append(resourcesOutput, resOut)

		case *kubernetes.KlothoHelmChart:
			resOut := map[string]interface{}{
				"Type": r.Key().Kind,
				"Name": r.Key().Name,
			}
			var files []string
			for _, f := range r.Files {
				files = append(files, f.Path())
			}
			resOut["Files"] = files
			resourcesOutput = append(resourcesOutput, resOut)

		case *core.InfraFiles:
			resOut := map[string]interface{}{
				"Type": r.Key().Kind,
				"Name": r.Key().Name,
			}
			var files []string
			for _, f := range r.Files.Values() {
				files = append(files, f.Path())
			}
			resOut["Files"] = files
			resourcesOutput = append(resourcesOutput, resOut)

		case *core.InputFiles:
			resOut := map[string]interface{}{
				"Type": r.Key().Kind,
			}
			var files []string
			for _, f := range r.Files() {
				files = append(files, f.Path())
			}
			resOut["Files"] = files
			resourcesOutput = append(resourcesOutput, resOut)

		case *execunit.FileDependencies:
			resOut := map[string]interface{}{
				"Type":     r.Key().Kind,
				"FileDeps": r,
			}
			resourcesOutput = append(resourcesOutput, resOut)

		default:
			resourcesOutput = append(resourcesOutput, r)
		}

		output, ok := res.(core.HasLocalOutput)
		if !ok {
			continue
		}
		zap.L().Debug("Output", zap.String("type", res.Key().Kind), zap.String("name", res.Key().Name))
		err = output.OutputTo(outDir)
		if err != nil {
			merr.Append(errors.Wrapf(err, "error outputting resource %+v", res.Key()))
		}
	}

	f, err := os.Create(path.Join(outDir, "resources.json"))
	if err != nil {
		merr.Append(errors.Wrap(err, "error creating resource dump"))
	} else {
		defer f.Close()

		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		err = enc.Encode(resourcesOutput)
		if err != nil {
			merr.Append(errors.Wrap(err, "error writing resources"))
		}
	}

	err = merr.ErrOrNil()

	return
}

func GetLanguagesUsed(result *core.CompilationResult) []core.ExecutableType {
	executableLangs := []core.ExecutableType{}
	for _, u := range core.GetResourcesOfType[*core.ExecutionUnit](result) {
		executableLangs = append(executableLangs, u.Executable.Type)
	}
	return executableLangs
}

func GetResourceTypeCount(result *core.CompilationResult, cfg *config.Application) (resourceCounts []string) {
	for _, res := range result.Resources() {
		resType := cfg.GetResourceType(res)
		if resType != "" {
			resourceCounts = append(resourceCounts, resType)
		}
	}
	return
}

func CloseTreeSitter(result *core.CompilationResult) {
	for _, eu := range core.GetResourcesOfType[*core.ExecutionUnit](result) {
		for _, f := range eu.Files() {
			if astFile, ok := f.(*core.SourceFile); ok {
				astFile.Tree().Close()
			}
		}
	}
}
