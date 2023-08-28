package errors

import (
	"fmt"
	"runtime"

	"github.com/pkg/errors"
)

type WrappedError struct {
	Message string
	Cause   error
	Stack   errors.StackTrace
}

func (err *WrappedError) Error() string {
	if err.Message != "" {
		return err.Message + ": " + err.Cause.Error()
	}
	return err.Cause.Error()
}

func (err *WrappedError) Format(s fmt.State, verb rune) {
	if err.Message != "" {
		fmt.Fprint(s, err.Message+": ")
	}
	if len(err.Stack) > 0 && s.Flag('+') {
		err.Stack.Format(s, verb)
	}
	if formatter, ok := err.Cause.(fmt.Formatter); ok {
		formatter.Format(s, verb)
	} else {
		fmt.Fprint(s, err.Cause.Error())
	}
}

func (err *WrappedError) Unwrap() error {
	return err.Cause
}

func WrapErrf(err error, msg string, args ...interface{}) *WrappedError {
	w := &WrappedError{
		Message: fmt.Sprintf(msg, args...),
		Cause:   err,
		Stack:   callers(2),
	}
	return w
}

func callers(depth int) errors.StackTrace {
	const maxDepth = 32

	var pcs [maxDepth]uintptr
	n := runtime.Callers(depth+1, pcs[:])

	frames := make([]errors.Frame, n)
	for i, frame := range pcs[:n] {
		frames[i] = errors.Frame(frame)
	}
	return errors.StackTrace(frames)
}
