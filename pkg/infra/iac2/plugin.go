package iac2

import (
	"bytes"
	"github.com/klothoplatform/klotho/pkg/core"
)

type (
	Plugin struct {
	}
)

func (p Plugin) Name() string {
	return "pulumi2"
}

func (p Plugin) Translate(cloudGraph *core.ResourceGraph) ([]core.File, error) {

	// TODO We'll eventually want to split the output into different files, but we don't know exactly what that looks
	// like yet. For now, just write to a single file, "new_index.ts"
	buf := &bytes.Buffer{}
	tc := CreateTemplatesCompiler(cloudGraph.Underlying)

	// index.ts
	if err := tc.RenderImports(buf); err != nil {
		return nil, err
	}
	if _, err := buf.WriteString("\n\n"); err != nil {
		return nil, err
	}

	if err := tc.RenderBody(buf); err != nil {
		return nil, err
	}
	indexTs := &core.RawFile{
		FPath:   `iac2/index.ts`,
		Content: bytes.Clone(buf.Bytes()),
	}
	buf.Reset()

	// TODO also write a package.json

	return []core.File{indexTs}, nil
}
