package updater

import (
	"io"

	"github.com/klothoplatform/klotho/pkg/multierr"
)

// TeeReader is like `io.TeeReader` except also implements `io.Closer` to close the underlying Reader/Writer.
type TeeReader struct {
	r io.Reader
	w io.Writer
}

func (t *TeeReader) Read(p []byte) (n int, err error) {
	n, err = t.r.Read(p)
	if n > 0 {
		if n, err := t.w.Write(p[:n]); err != nil {
			return n, err
		}
	}
	return
}

func (t *TeeReader) Close() error {
	var readErr, writeErr error
	if rc, ok := t.r.(io.Closer); ok {
		readErr = rc.Close()
	}
	if wc, ok := t.w.(io.Closer); ok {
		writeErr = wc.Close()
	}
	switch {
	case readErr != nil && writeErr != nil:
		return multierr.Error{readErr, writeErr}
	case readErr != nil:
		return readErr
	case writeErr != nil:
		return writeErr
	}
	return nil
}
