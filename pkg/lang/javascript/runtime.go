package javascript

import (
	"bytes"
	"embed"
	"io/fs"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/lang/dockerfile"
	"github.com/klothoplatform/klotho/pkg/multierr"
)

func RuntimePath(sourcePath string, runtimeModule string) (string, error) {
	runtimeImportPath, err := filepath.Rel(filepath.Dir(sourcePath), "./klotho_runtime")
	if err != nil {
		return "", err
	}
	runtimeImportPath = filepath.Join(runtimeImportPath, runtimeModule)

	// Do this last because `filepath` functions tend to strip the (in their opinion) unnecessary leading './'
	if !strings.HasPrefix(runtimeImportPath, ".") {
		// relative imports in JS need the `./` prefix to distinguish from node_modules modules
		runtimeImportPath = "./" + runtimeImportPath
	}
	return runtimeImportPath, nil
}

// TransformResult is the result of a runtime applying a transformation to a specific file and annotation.
type TransformResult struct {
	// NewFileContent contains the new file content in its entirety, after transformations have been applied. This additionally allows
	// for changes to the file outside of the annotation's node (such as adding new imports).
	NewFileContent string
	// NewAnnotationContent contains just the new annotation node's content. This is required for any further
	// plugin transformations, applied after the runtime's transformation.
	NewAnnotationContent string
}

type Runtime interface {
	// TransformPersist applies any runtime-specific transformations to the given file for the annotation. Returns the modified source code, to be `Reparse`d by the caller.
	TransformPersist(file *core.SourceFile, annot *core.Annotation, kind core.PersistKind, content string) (TransformResult, error)
	AddKvRuntimeFiles(unit *core.ExecutionUnit) error
	AddFsRuntimeFiles(unit *core.ExecutionUnit) error
	AddSecretRuntimeFiles(unit *core.ExecutionUnit) error
	AddOrmRuntimeFiles(unit *core.ExecutionUnit) error
	AddRedisNodeRuntimeFiles(unit *core.ExecutionUnit) error
	AddRedisClusterRuntimeFiles(unit *core.ExecutionUnit) error
	AddPubsubRuntimeFiles(unit *core.ExecutionUnit) error
	AddProxyRuntimeFiles(unit *core.ExecutionUnit, proxyType string) error
	AddExecRuntimeFiles(unit *core.ExecutionUnit, result *core.CompilationResult, deps *core.Dependencies) error
}

func AddRuntimeFile(unit *core.ExecutionUnit, templateData any, path string, content []byte) error {

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
	case path == "package.json":
		runtimePkg, err := NewPackageFile(path, bytes.NewBuffer(content))
		if err != nil {
			return err
		}
		for _, f := range unit.Files() {
			pkg, ok := f.(*PackageFile)
			if !ok {
				continue
			}
			pkg.Content.Merge(runtimePkg.Content)
		}

	case filepath.Ext(path) == ".js":
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

// EnsureRuntimeImportFile makes sure that `file` has an import for `runtimePath` with name based on `varPrefix`.
// `runtimePath` is relative to klotho_runtime (ie, don't do "klotho_runtime/my_runtime_module") and is often the same
// as varPrefix
// ! Calls `file.Reparse` if the import is missing.
func EnsureRuntimeImportFile(runtimePath string, varPrefix string, file *core.SourceFile) (err error) {
	buf := new(bytes.Buffer)

	rtimp := RuntimeImport{
		VarName: varPrefix,
	}
	rtimp.FilePath, err = RuntimePath(file.Path(), runtimePath)
	if err != nil {
		return err
	}
	if err := NewRuntimeImport(rtimp, buf); err != nil {
		return err
	}
	hasImport := bytes.Contains(file.Program(), buf.Bytes())
	if hasImport {
		// import already present, do nothing
		return nil
	}

	buf.Write(file.Program())
	return file.Reparse(buf.Bytes())
}

// EnsureRuntimeImport makes sure that `file` has an import for `runtimePath` (relative to `filepath`) with name based on `varPrefix`.
// `runtimePath` is relative to klotho_runtime (ie, don't do "klotho_runtime/my_runtime_module") and is often the same
// as varPrefix
func EnsureRuntimeImport(filepath string, runtimePath string, varPrefix string, content string) (newContent string, err error) {
	buf := new(bytes.Buffer)

	rtimp := RuntimeImport{
		VarName: varPrefix,
	}
	rtimp.FilePath, err = RuntimePath(filepath, runtimePath)
	if err != nil {
		return content, err
	}
	if err := NewRuntimeImport(rtimp, buf); err != nil {
		return content, err
	}
	if strings.Contains(content, buf.String()) {
		// import already present, do nothing
		return content, nil
	}
	buf.WriteString(content)
	return buf.String(), err
}
