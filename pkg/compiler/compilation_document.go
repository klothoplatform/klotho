package compiler

import (
	"os"
	"path/filepath"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
)

type (
	CompilationDocument struct {
		InputFiles     *core.InputFiles
		Constructs     *core.ConstructGraph
		Configuration  *config.Application
		CloudResources *core.ResourceGraph
		OutputFiles    []core.File
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
