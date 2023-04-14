package compiler

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type (
	CompilationDocument struct {
		InputFiles       *core.InputFiles
		FileDependencies *core.FileDependencies
		Constructs       *core.ConstructGraph
		Configuration    *config.Application
		Resources        *core.ResourceGraph
		OutputFiles      []core.File
	}
)

func (doc *CompilationDocument) OutputTo(dest string) error {
	errs := make(chan error)
	files := doc.OutputFiles
	for idx := range files {
		go func(f core.File) {
			path := filepath.Join(dest, f.Path())
			dir := filepath.Dir(path)
			err := os.MkdirAll(dir, 0777)
			if err != nil {
				errs <- err
				return
			}
			file, err := os.OpenFile(path, os.O_RDWR, 0777)
			if os.IsNotExist(err) {
				file, err = os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0777)
			} else if err == nil {
				ovr, ok := f.(core.NonOverwritable)
				if ok && !ovr.Overwrite(file) {
					errs <- nil
					return
				}
				err = file.Truncate(0)
			}
			if err != nil {
				errs <- err
				return
			}
			_, err = f.WriteTo(file)
			file.Close()
			errs <- err
		}(files[idx])
	}

	for i := 0; i < len(files); i++ {
		err := <-errs
		if err != nil {
			return err
		}
	}
	return nil
}

func (document *CompilationDocument) OutputResources() (resourceCounts map[string]int, err error) {
	outDir := document.Configuration.OutDir
	result := document.Constructs

	err = os.MkdirAll(outDir, 0777)
	if err != nil {
		return
	}

	resourceCounts = make(map[string]int)
	var resourcesOutput []interface{}
	var merr multierr.Error
	for _, construct := range result.ListConstructs() {
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

	merr.Append(document.OutputHelpers(outDir))

	merr.Append(document.OutputTo(document.Configuration.OutDir))
	err = merr.ErrOrNil()

	return
}

func (document *CompilationDocument) OutputHelpers(outDir string) error {
	var merr multierr.Error
	for _, resource := range document.Resources.ListResources() {
		if res, ok := resource.(core.HasOutputFiles); ok {
			document.OutputFiles = append(document.OutputFiles, res.GetOutputFiles()...)
		}
		output, ok := resource.(core.HasLocalOutput)
		if !ok {
			continue
		}
		zap.L().Debug("Output", zap.String("provider", resource.Provider()), zap.String("id", resource.Id()))
		err := output.OutputTo(outDir)
		if err != nil {
			merr.Append(errors.Wrapf(err, "error outputting resource %+v", resource.Id()))
		}
	}
	return merr.ErrOrNil()
}
