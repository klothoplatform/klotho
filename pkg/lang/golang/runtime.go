package golang

import (
	"bytes"
	"html/template"
	"path/filepath"
	"strings"

	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/klothoplatform/klotho/pkg/construct"
	klotho_errors "github.com/klothoplatform/klotho/pkg/errors"
	"github.com/klothoplatform/klotho/pkg/io"
	"github.com/klothoplatform/klotho/pkg/lang/dockerfile"
)

type (
	Runtime interface {
		AddExecRuntimeFiles(unit *types.ExecutionUnit, constructGraph *construct.ConstructGraph) error
		GetFsImports() []Import
		GetSecretsImports() []Import
		SetConfigType(id string, isSecret bool)
		ActOnExposeListener(unit *types.ExecutionUnit, f *types.SourceFile, listener *HttpListener, routerName string) error
	}
)

func AddRuntimeFile(unit *types.ExecutionUnit, templateData any, path string, content []byte) error {
	// TODO refactor to consolidate with this method in the javascript package
	if filepath.Ext(path) == ".tmpl" {
		t, err := template.New(path).Parse(string(content))
		if err != nil {
			return klotho_errors.WrapErrf(err, "error parsing template %s", path)
		}
		tmplBuf := new(bytes.Buffer)
		err = t.Execute(tmplBuf, templateData)
		if err != nil {
			return klotho_errors.WrapErrf(err, "error executing template %s", path)
		}

		content = tmplBuf.Bytes()
		path = strings.TrimSuffix(path, ".tmpl")
	}
	switch {
	case filepath.Ext(path) == ".go":
		path = filepath.Join("klotho_runtime", path)
		f, err := NewFile(path, bytes.NewReader(content))
		if err != nil {
			return klotho_errors.WrapErrf(err, "error parsing template %s", path)
		}
		unit.Add(f)
	case path == "Dockerfile":
		dockerF, err := dockerfile.NewFile(path, bytes.NewBuffer(content))
		if err != nil {
			return klotho_errors.WrapErrf(err, "error adding file %s", path)
		}
		unit.Add(dockerF)
	default:
		unit.Add(&io.RawFile{
			FPath:   path,
			Content: content,
		})
	}
	return nil
}
