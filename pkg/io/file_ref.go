package io

import (
	"io"
	"os"
	"path/filepath"
)

type (

	// FileRef is a lightweight representation of a file, deferring reading its contents until `WriteTo` is called.
	FileRef struct {
		FPath          string
		RootConfigPath string
	}
)

func (r *FileRef) Clone() File {
	return r
}

func (r *FileRef) Path() string {
	return r.FPath
}

func (r *FileRef) WriteTo(w io.Writer) (int64, error) {
	f, err := os.Open(filepath.Join(r.RootConfigPath, r.FPath))
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return io.Copy(w, f)
}

func OutputTo(files []File, dest string) error {

	errs := make(chan error)
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
