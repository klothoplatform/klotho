package types

import (
	"os"
	"path/filepath"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/async"
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/io"
)

type (
	StaticUnit struct {
		Name          string
		IndexDocument string
		StaticFiles   async.ConcurrentMap[string, io.File]
		SharedFiles   async.ConcurrentMap[string, io.File]
	}
)

const STATIC_UNIT_TYPE = "static_unit"

func (p *StaticUnit) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: construct.AbstractConstructProvider,
		Type:     STATIC_UNIT_TYPE,
		Name:     p.Name,
	}
}

func (p *StaticUnit) AnnotationCapability() string {
	return annotation.StaticUnitCapability
}

func (p *StaticUnit) Functionality() construct.Functionality {
	return construct.Storage
}

func (p *StaticUnit) Attributes() map[string]any {
	return map[string]any{
		"blob": nil,
	}
}

func (unit *StaticUnit) OutputTo(dest string) error {
	errs := make(chan error)
	files := unit.Files()
	for idx := range files {
		go func(f io.File) {
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
				ovr, ok := f.(io.NonOverwritable)
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

func (unit *StaticUnit) Files() map[string]io.File {
	m := make(map[string]io.File)
	for _, f := range unit.SharedFiles.Values() {
		m[f.Path()] = f
	}
	for _, f := range unit.StaticFiles.Values() {
		m[f.Path()] = f
	}
	return m
}

func (unit *StaticUnit) AddStaticFile(f io.File) {
	if f != nil {
		unit.StaticFiles.Set(f.Path(), f)
	}
}

func (unit *StaticUnit) AddSharedFile(f io.File) {
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

func (unit *StaticUnit) GetSharedFile(path string) io.File {
	f, _ := unit.SharedFiles.Get(path)
	return f
}

func (unit *StaticUnit) GetStaticFile(path string) io.File {
	f, _ := unit.StaticFiles.Get(path)
	return f
}
