package core

import (
	"os"
	"path/filepath"

	"github.com/klothoplatform/klotho/pkg/graph"
)

type (
	CompilationDocument struct {
		InputFiles     *InputFiles
		Constructs     graph.Directed[Construct]
		CloudResources graph.Directed[ProviderResource]
		OutputFiles    []File
	}
)

func (doc *CompilationDocument) OutputTo(dest string) error {
	errs := make(chan error)
	files := doc.OutputFiles
	for idx := range files {
		go func(f File) {
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
				ovr, ok := f.(NonOverwritable)
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
