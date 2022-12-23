package core

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/fatih/color"
	"github.com/pkg/errors"
)

type (
	PluginError struct {
		Plugin string
		Cause  error
	}

	CompileError struct {
		File       *SourceFile
		Annotation *Annotation
		Cause      error
	}

	WrappedError struct {
		Message string
		Cause   error
		Stack   errors.StackTrace
	}
)

var (
	errorColour = color.New(color.FgRed)
)

func NewPluginError(name string, cause error) *PluginError {
	if subPlugin, ok := cause.(*PluginError); ok {
		return &PluginError{
			Plugin: name + "/" + subPlugin.Plugin,
			Cause:  subPlugin.Cause,
		}
	}
	return &PluginError{
		Plugin: name,
		Cause:  cause,
	}
}

func (err *PluginError) Error() string {
	return fmt.Sprintf("error in plugin %s: %v", err.Plugin, err.Cause)
}

func (err *PluginError) Format(s fmt.State, verb rune) {
	fmt.Fprintf(s, "error in plugin %s: ", err.Plugin)
	if formatter, ok := err.Cause.(fmt.Formatter); ok {
		formatter.Format(s, verb)
	} else {
		fmt.Fprint(s, err.Error())
	}
}

func (err *PluginError) Unwrap() error {
	return err.Cause
}

func NewCompilerError(f *SourceFile, annotation *Annotation, cause error) *CompileError {
	return &CompileError{
		File:       f,
		Annotation: annotation,
		Cause:      cause,
	}
}

func (err *CompileError) Error() string {
	sb := new(strings.Builder)
	if err.File != nil {
		fmt.Fprintf(sb, "error in %s", err.File.Path())
	}
	if err.Annotation.Capability != nil {
		start := err.Annotation.Node.StartPoint()
		fmt.Fprintf(sb, ":%d:%d (in %s)", start.Row, start.Column, err.Annotation.Capability.Name)
	}
	fmt.Fprintf(sb, ": %v", err.Cause)
	return sb.String()
}

func (err *CompileError) Format(s fmt.State, verb rune) {
	if err.File != nil {
		errorColour.Fprintf(s, "Error in %s:", err.File.Path())

		if err.Annotation.Capability != nil {
			fmt.Fprint(s, "\n")
			err.Annotation.Format(s, verb)
			fmt.Fprintf(s, "\nin %s\n", err.File.Path())
			fnode := &NodeContent{
				Endpoints: err.Annotation.Node,
				Content:   err.Annotation.Node.Content(err.File.Program()),
			}
			fnode.Format(s, verb)
		}
	}
	errorColour.Fprint(s, "\n-> Error: ")
	if formatter, ok := err.Cause.(fmt.Formatter); ok {
		// TODO add errorColour to this
		formatter.Format(s, verb)
	} else {
		errorColour.Fprint(s, err.Cause.Error())
	}
}

func (err *CompileError) Unwrap() error {
	return err.Cause
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
