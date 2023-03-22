package golang

import (
	"bytes"
	"html/template"
	"path/filepath"
	"strings"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/graph"
	"github.com/klothoplatform/klotho/pkg/lang/dockerfile"
)

type (
	Runtime interface {
		AddExecRuntimeFiles(unit *core.ExecutionUnit, constructGraph *graph.Directed[core.Construct]) error
		GetFsImports() []Import
		GetSecretsImports() []Import
		SetConfigType(id string, isSecret bool)
		ActOnExposeListener(unit *core.ExecutionUnit, f *core.SourceFile, listener *HttpListener, routerName string) error
	}
)

func AddRuntimeFile(unit *core.ExecutionUnit, templateData any, path string, content []byte) error {
	// TODO refactor to consolidate with this method in the javascript package
	if filepath.Ext(path) == ".tmpl" {
		t, err := template.New(path).Parse(string(content))
		if err != nil {
			return core.WrapErrf(err, "error parsing template %s", path)
		}
		tmplBuf := new(bytes.Buffer)
		err = t.Execute(tmplBuf, templateData)
		if err != nil {
			return core.WrapErrf(err, "error executing template %s", path)
		}

		content = tmplBuf.Bytes()
		path = strings.TrimSuffix(path, ".tmpl")
	}
	switch {
	case filepath.Ext(path) == ".go":
		path = filepath.Join("klotho_runtime", path)
		f, err := NewFile(path, bytes.NewReader(content))
		if err != nil {
			return core.WrapErrf(err, "error parsing template %s", path)
		}
		unit.Add(f)
	case path == "Dockerfile":
		dockerF, err := dockerfile.NewFile(path, bytes.NewBuffer(content))
		if err != nil {
			return core.WrapErrf(err, "error adding file %s", path)
		}
		unit.Add(dockerF)
	default:
		unit.Add(&core.RawFile{
			FPath:   path,
			Content: content,
		})
	}
	return nil
}
