package core

import (
	"io"
	"os"
	"path/filepath"
)

// FileRef is a lightweight representation of a file, deferring reading its contents until `WriteTo` is called.
type FileRef struct {
	FPath          string
	RootConfigPath string
}

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
