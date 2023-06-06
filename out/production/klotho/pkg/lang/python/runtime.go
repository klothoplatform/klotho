package python

import (
	"bytes"
	"embed"
	"io/fs"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/klothoplatform/klotho/pkg/lang/dockerfile"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"github.com/pkg/errors"
	sitter "github.com/smacker/go-tree-sitter"

	"github.com/klothoplatform/klotho/pkg/core"
)

type (
	Runtime interface {
		AddExecRuntimeFiles(*core.ExecutionUnit, *core.ConstructGraph) error
		AddExposeRuntimeFiles(*core.ExecutionUnit) error
		AddKvRuntimeFiles(unit *core.ExecutionUnit) error
		AddFsRuntimeFiles(unit *core.ExecutionUnit, envVarName string, id string) error
		AddOrmRuntimeFiles(unit *core.ExecutionUnit) error
		AddProxyRuntimeFiles(unit *core.ExecutionUnit, proxyType string) error
		AddSecretRuntimeFiles(unit *core.ExecutionUnit) error
		GetKvRuntimeConfig() KVConfig
		GetFsRuntimeImportClass(id string, varName string) string
		GetSecretRuntimeImportClass(varName string) string
		GetAppName() string
	}
)

type KVConfig struct {
	Imports                        string
	CacheClassArg                  FunctionArg
	AdditionalCacheConstructorArgs []FunctionArg
}

func AddRequirements(unit *core.ExecutionUnit, requirements string) {
	foundPip := false
	for _, f := range unit.Files() {
		pip, isPip := f.(*RequirementsTxt)
		if isPip {
			pip.AddLine(requirements)
			foundPip = true
		}
	}
	if !foundPip {
		pip := &RequirementsTxt{path: "requirements.txt"}
		unit.Add(pip)
		pip.AddLine(requirements)
	}
}

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
	case filepath.Ext(path) == ".py":
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

func AddRuntimeFiles(unit *core.ExecutionUnit, files embed.FS, templateData any) error {
	err := fs.WalkDir(files, ".", func(path string, d fs.DirEntry, walkErr error) error {
		// don't immediately exit if walkErr != nil, we want to try everything first to collect all the errors
		if d.IsDir() {
			return nil
		}
		content, err := files.ReadFile(path)
		if err != nil {
			return multierr.Append(walkErr, core.WrapErrf(err, ""))
		}
		err = AddRuntimeFile(unit, templateData, path, content)
		if err != nil {
			return multierr.Append(walkErr, core.WrapErrf(err, "failed to AddRuntimeFile"))
		}
		return nil

	})
	return err
}

// AddRuntimeImport injects the supplied import string above the first non-comment statement in the supplied file.
func AddRuntimeImport(importString string, file *core.SourceFile) error {
	root := file.Tree().RootNode()

	var lastImport *sitter.Node
	var firstExpression *sitter.Node
	for i := 0; i < int(root.ChildCount()); i++ {
		child := root.Child(i)
		if (strings.Contains(child.Type(), "comment")) || isStringLiteralExpression(child) {
			continue
		}
		if containsAny(child.Type(), "import_statement", "import_from_statement") {
			lastImport = child
		} else {
			firstExpression = child
			break
		}
	}

	content := file.Program()
	insertionPoint := uint32(0)
	if lastImport != nil {
		insertionPoint = lastImport.EndByte()
	} else if firstExpression != nil {
		insertionPoint = firstExpression.StartByte()
	}

	contentStr := string(content[0:insertionPoint]) + "\n" + importString + string(content[insertionPoint:])
	err := file.Reparse([]byte(contentStr))
	if err != nil {
		return errors.Wrap(err, "could not reparse inserted import")
	}

	return nil
}

func RuntimePath(sourcePath string, runtimeModule string) (string, error) {
	runtimeImportPath, err := filepath.Rel(sourcePath, "./klotho_runtime")
	if err != nil {
		return "", err
	}
	return filepath.Join(runtimeImportPath, runtimeModule), nil
}

func isStringLiteralExpression(node *sitter.Node) bool {
	return node.Type() == "expression_statement" && node.ChildCount() == 1 && node.Child(0).Type() == "string"
}

func containsAny(input string, matchOptions ...string) bool {
	for _, opt := range matchOptions {
		if strings.Contains(input, opt) {
			return true
		}
	}
	return false
}
