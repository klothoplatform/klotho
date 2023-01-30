package core

import (
	"os"
	"path/filepath"
)

type InfraFiles struct {
	Name  string
	Files ConcurrentMap[string, File]
}

var InfraAsCodeKind = "infra_as_code"

func (iac *InfraFiles) Key() ResourceKey {
	return ResourceKey{
		Name: iac.Name,
		Kind: InfraAsCodeKind,
	}
}

func (iac *InfraFiles) OutputTo(dest string) error {
	errs := make(chan error)
	files := iac.Files.Values()
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

func (unit *InfraFiles) Add(f File) {
	unit.Files.Set(f.Path(), f)
}
