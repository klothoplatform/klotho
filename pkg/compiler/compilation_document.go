package compiler

import (
	"encoding/json"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/logging"
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
		OutputOptions    OutputOptions
	}

	OutputOptions struct {
		PostWriteHooks map[string]string `yaml:"post-write-hooks,omitempty"`
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

			fileExt := filepath.Ext(path)
			if strings.HasPrefix(fileExt, ".") {
				fileExt = fileExt[1:]
				if hook, found := doc.OutputOptions.PostWriteHooks[fileExt]; found {
					log := zap.S().With(logging.FileField(f))
					hookErr := postCompileHook(dest, f, hook, log)
					if hookErr != nil {
						zap.S().With(zap.Error(hookErr), logging.FileField(f)).Warnf(
							`failed to apply post-output hook to %s: %s`,
							f.Path(),
							hookErr.Error())
					}
				}
			}

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

func postCompileHook(dir string, file core.File, hook string, log *zap.SugaredLogger) error {
	hookSegments := strings.Split(hook, " ")
	if len(hookSegments) == 0 {
		return errors.New(`empty formatter command`)
	}
	var args []string
	for _, arg := range hookSegments[1:] {
		if arg == "{}" {
			arg = file.Path()
		}
		args = append(args, arg)
	}

	cmd := exec.Command(hookSegments[0], args...)
	log.Infof(`running post-output hook: %s`, strings.Join(cmd.Args, " "))
	cmd.Dir = dir
	return cmd.Run()
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
	}
	return merr.ErrOrNil()
}
