package io

import "io"

type CountingWriter struct {
	Delegate     io.Writer
	BytesWritten int
}

func (w *CountingWriter) Write(p []byte) (int, error) {
	n, err := w.Delegate.Write(p)
	w.BytesWritten += n
	return n, err
}
