package cli

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/graph"
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

func OutputResources(result *graph.Directed[core.Construct], outDir string) (resourceCounts map[string]int, err error) {
	err = os.MkdirAll(outDir, 0777)
	if err != nil {
		return
	}

	resourceCounts = make(map[string]int)
	var resourcesOutput []interface{}
	var merr multierr.Error
	for _, construct := range result.GetAllVertices() {
		resourceCounts[construct.Provenance().Capability] = resourceCounts[construct.Provenance().Capability] + 1

		switch r := construct.(type) {
		case *core.ExecutionUnit:
			resOut := map[string]interface{}{
				"Type": r.Provenance().Capability,
				"Name": r.Provenance().ID,
			}
			var files []string
			for _, f := range r.Files() {
				files = append(files, f.Path())
			}
			resOut["Files"] = files
			resourcesOutput = append(resourcesOutput, resOut)
		default:
			resourcesOutput = append(resourcesOutput, r)
		}

		output, ok := construct.(core.HasLocalOutput)
		if !ok {
			continue
		}
		zap.L().Debug("Output", zap.String("type", construct.Provenance().Capability), zap.String("name", construct.Provenance().ID))
		err = output.OutputTo(outDir)
		if err != nil {
			merr.Append(errors.Wrapf(err, "error outputting resource %+v", construct.Provenance()))
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

func GetLanguagesUsed(result *graph.Directed[core.Construct]) []core.ExecutableType {
	executableLangs := []core.ExecutableType{}
	for _, u := range core.GetResourcesOfType[*core.ExecutionUnit](result) {
		executableLangs = append(executableLangs, u.Executable.Type)
	}
	return executableLangs
}

func GetResourceCount(counts map[string]int) (resourceCounts []string) {
	for key, num := range counts {
		for i := 0; i < num; i++ {
			resourceCounts = append(resourceCounts, key)
		}
	}
	return
}

func GetResourceTypeCount(result *graph.Directed[core.Construct], cfg *config.Application) (resourceCounts []string) {
	for _, res := range result.GetAllVertices() {
		resType := cfg.GetResourceType(res)
		if resType != "" {
			resourceCounts = append(resourceCounts, resType)
		}
	}
	return
}

func CloseTreeSitter(result *graph.Directed[core.Construct]) {
	for _, eu := range core.GetResourcesOfType[*core.ExecutionUnit](result) {
		for _, f := range eu.Files() {
			if astFile, ok := f.(*core.SourceFile); ok {
				astFile.Tree().Close()
			}
		}
	}
}
