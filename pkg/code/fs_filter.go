package code

import (
	"io/fs"
	"path"
)

type FilteredFS struct {
	fs.FS
	Exclude func(path string) bool
}

func (f FilteredFS) Open(name string) (fs.File, error) {
	if f.Exclude(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	return f.FS.Open(name)
}

func (f FilteredFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if f.Exclude(name) {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrNotExist}
	}
	entries, err := fs.ReadDir(f.FS, name)
	if err != nil {
		return nil, err
	}
	filtered := make([]fs.DirEntry, 0, len(entries))
	for _, entry := range entries {
		if !f.Exclude(path.Join(name, entry.Name())) {
			filtered = append(filtered, entry)
		}
	}
	return filtered, nil
}

func (f FilteredFS) Stat(name string) (fs.FileInfo, error) {
	if f.Exclude(name) {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrNotExist}
	}
	if s, ok := f.FS.(fs.StatFS); ok {
		return s.Stat(name)
	}
	return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrNotExist}
}
