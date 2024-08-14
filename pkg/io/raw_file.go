package io

import (
	"io"
)

// RawFile represents a file with its included `Content` in case the compiler needs to read/manipulate it.
// If the content is not needed except to `WriteTo`, then try using [FileRef] instead.
type RawFile struct {
	FPath   string
	Content []byte
}

type File interface {
	Path() string
	WriteTo(io.Writer) (int64, error)
	Clone() File
}

func (r *RawFile) Clone() File {
	nf := &RawFile{
		FPath: r.FPath,
	}
	nf.Content = make([]byte, len(r.Content))
	copy(nf.Content, r.Content)
	return nf
}

func (r *RawFile) Path() string {
	return r.FPath
}

func (r *RawFile) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(r.Content)
	return int64(n), err
}
