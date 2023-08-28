package types

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
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
	if err.Annotation.Capability != nil && err.Annotation.Node != nil {
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
			fmt.Fprintf(s, "\nin %s", err.File.Path())
			if err.Annotation.Node != nil {
				fmt.Fprint(s, "\n")
				fnode := &NodeContent{
					Endpoints: err.Annotation.Node,
					Content:   err.Annotation.Node.Content(),
				}
				fnode.Format(s, verb)
			}
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
