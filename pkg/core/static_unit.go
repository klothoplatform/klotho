package core

import (
	"os"
	"path/filepath"

	"github.com/klothoplatform/klotho/pkg/annotation"
)

type (
	StaticUnit struct {
		Name          string
		IndexDocument string
		StaticFiles   ConcurrentMap[string, File]
		SharedFiles   ConcurrentMap[string, File]
	}
)

const STATIC_UNIT_TYPE = "static_unit"

func (p *StaticUnit) Id() ResourceId {
	return ResourceId{
		Provider: AbstractConstructProvider,
		Type:     STATIC_UNIT_TYPE,
		Name:     p.Name,
	}
}

func (p *StaticUnit) AnnotationCapability() string {
	return annotation.StaticUnitCapability
}

func (unit *StaticUnit) OutputTo(dest string) error {
	errs := make(chan error)
	files := unit.Files()
	for idx := range files {
		go func(f File) {
			path := filepath.Join(dest, unit.Name, f.Path())
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

func (unit *StaticUnit) Files() map[string]File {
	m := make(map[string]File)
	for _, f := range unit.SharedFiles.Values() {
		m[f.Path()] = f
	}
	for _, f := range unit.StaticFiles.Values() {
		m[f.Path()] = f
	}
	return m
}

func (unit *StaticUnit) AddStaticFile(f File) {
	if f != nil {
		unit.StaticFiles.Set(f.Path(), f)
	}
}

func (unit *StaticUnit) AddSharedFile(f File) {
	if f != nil {
		unit.SharedFiles.Set(f.Path(), f)
	}
}

func (unit *StaticUnit) RemoveSharedFile(path string) {
	unit.SharedFiles.Delete(path)
}

func (unit *StaticUnit) RemoveStaticFile(path string) {
	unit.StaticFiles.Delete(path)
}

func (unit *StaticUnit) GetSharedFile(path string) File {
	f, _ := unit.SharedFiles.Get(path)
	return f
}

func (unit *StaticUnit) GetStaticFile(path string) File {
	f, _ := unit.StaticFiles.Get(path)
	return f
}
