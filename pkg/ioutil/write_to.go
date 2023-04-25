package ioutil

import (
	"fmt"
	"io"

	"github.com/klothoplatform/klotho/pkg/multierr"
)

type (
	// WriteToHelper simplifies the use of a [io.Writer], and specifically in a way that helps you implement
	// [io.WriterTo]. It does so by wrapping the Writer along with a reference to the count and err that WriterTo
	// requires. When you write to the WriteToHelper, it either delegates to the Writer if there has not been an error,
	// or else ignores the write if there has been. If it delegates, it also updates the count and err values.
	WriteToHelper struct {
		out   io.Writer
		count *int64
		err   *error
	}
)

// NewWriteToHelper creates a new WriteToHelper which delegates to the given Writer and updates the given count and err
// as needed.
//
// A good pattern for how to use this is:
//
//	func (wt *MyWriterTo) (w io.Writer) (count int64, err error)
//		wh := ioutil.NewWriteToHelper(w, &count, &err)
//		wh.Write("hello")
//		wh.Write("world")
//		return
//	}
//
// The "wh" helper will delegate each of its writes to the "w" Writer, updating count and err as needed along the way.
// If the Writer ever returns a non-nil error, subsequent write operations on the "wh" helper will be ignored.
func NewWriteToHelper(out io.Writer, count *int64, err *error) WriteToHelper {
	return WriteToHelper{
		out:   out,
		count: count,
		err:   err,
	}
}

func (w WriteToHelper) AddErr(err error) {
	if *w.err == nil {
		*w.err = err
	} else if multiErr, ok := (*w.err).(multierr.Error); ok {
		multiErr.Append(err)
	} else {
		multiErr = multierr.Error{}
		multiErr.Append(*w.err)
		multiErr.Append(err)
		*w.err = multiErr
	}
}

func (w WriteToHelper) Write(s string) {
	w.Writef(`%s`, s)
}

func (w WriteToHelper) Writef(format string, a ...any) {
	if *w.err != nil {
		return
	}

	count, err := fmt.Fprintf(w.out, format, a...)
	*w.count += int64(count)
	*w.err = err
}
