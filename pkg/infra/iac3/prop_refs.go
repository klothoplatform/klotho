package iac3

import (
	"bytes"
	"fmt"
	"sync"
	"text/template"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
)

var bufPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

func getBuffer() *bytes.Buffer {
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

func releaseBuffer(buf *bytes.Buffer) {
	bufPool.Put(buf)
}

func executeToString(tmpl *template.Template, data any) (string, error) {
	buf := getBuffer()
	defer releaseBuffer(buf)
	err := tmpl.Execute(buf, data)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (tc *TemplatesCompiler) PropertyRefValue(ref construct.PropertyRef) (any, error) {
	tmpl, err := tc.templates.ResourceTemplate(ref.Resource)
	if err != nil {
		return nil, err
	}
	refRes, err := tc.graph.Vertex(ref.Resource)
	if err != nil {
		return nil, err
	}

	if mapping, ok := tmpl.PropertyTemplates[ref.Property]; ok {
		inputArgs, err := tc.getInputArgs(refRes)
		if err != nil {
			return nil, err
		}
		data := PropertyTemplateData{
			Object: tc.vars[ref.Resource],
			Input:  inputArgs,
		}
		return executeToString(mapping, data)
	}
	path, err := refRes.PropertyPath(ref.Property)
	if err != nil {
		return nil, err
	}
	if path != nil {
		return tc.convertArg(path.Get())
	}
	return nil, fmt.Errorf("unsupported property ref %s", ref)
}
