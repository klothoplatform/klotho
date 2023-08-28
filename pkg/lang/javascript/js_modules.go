package javascript

import (
	"path"
	"path/filepath"
	"strings"

	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/io"
	"github.com/klothoplatform/klotho/pkg/query"
	sitter "github.com/smacker/go-tree-sitter"
)

func GetFileForModule(constructGraph *construct.ConstructGraph, moduleName string) *types.SourceFile {
	moduleName = strings.TrimPrefix(moduleName, "./") // Convert relative import to file name

	original, f := types.GetExecUnitForPath(moduleName, constructGraph)
	if original == nil && !strings.HasSuffix(moduleName, ".js") {
		original, f = types.GetExecUnitForPath(moduleName+".js", constructGraph)
	}
	if original == nil {
		original, f = types.GetExecUnitForPath(path.Join(moduleName, "index.js"), constructGraph)
	}
	if original == nil {
		return nil
	}
	jsFile, ok := Language.ID.CastFile(f)
	if ok {
		return jsFile
	}
	return nil
}

// FindFileForImport is the reverse of `FindImportOfFile`.
func FindFileForImport(files map[string]io.File, importingFilePath string, module string) (f io.File, err error) {
	path, err := filepath.Rel(".", filepath.Join(filepath.Dir(importingFilePath), module))
	if err != nil {
		return nil, err
	}
	if path == "." {
		if f, ok := files["index.js"]; ok {
			return f, nil
		}
	}
	if f, ok := files[path]; ok {
		return f, nil
	}
	if f, ok := files[path+".js"]; ok {
		return f, nil
	}
	if f, ok := files[path+"/index.js"]; ok {
		return f, nil
	}
	return nil, nil
}

func FindDefaultExport(n *sitter.Node) *sitter.Node {
	nextMatch := DoQuery(n, modulesDefault)
	var last *sitter.Node
	for {
		match, found := nextMatch()
		if !found || match == nil {
			break
		}

		obj, prop := match["obj"], match["prop"]
		if obj != nil && !query.NodeContentEquals(obj, "module") {
			continue
		}

		if !query.NodeContentEquals(prop, "exports") {
			continue
		}
		last = match["last"]
	}

	return last
}

// FindExportForVar returns the local variable that is exported as 'varName' (to handle cases where they don't match).
func FindExportForVar(n *sitter.Node, varName string) *sitter.Node {
	nextMatch := DoQuery(n, modulesExport)
	var last *sitter.Node
	for {
		match, found := nextMatch()
		if !found {
			break
		}

		obj, prop, right := match["obj"], match["prop"], match["right"]

		if !query.NodeContentEquals(obj, "exports") {
			continue
		}
		if !query.NodeContentEquals(prop, varName) {
			continue
		}
		if right.Type() == "identifier" {
			last = right
		} else {
			last = obj.Parent()
		}
	}

	return last
}

// FileToLocalModule removes all the extraneous parts of the file path while still being resolvable by node's require.
// Also appends a leading `./` if the path is not already relative to convert from file paths in an execution unit
// which do not have the leading `./` (required for relative imports by node).
func FileToLocalModule(path string) (module string) {
	module = FileToModule(path)
	if !strings.HasPrefix(module, ".") {
		if module != "" {
			module = "./" + module
		} else {
			// This makes "index.js" resolve to "." instead of "./"
			// which matches how "../index.js" would resolve to ".."
			// which happens because `module == ""` until the following.
			module = "."
		}
	}
	return
}

// FileToModule removes all the extraneous parts of the file path while still being resolvable by node's require.
func FileToModule(path string) (module string) {
	module = path
	module = strings.TrimSuffix(module, ".js")
	module = strings.TrimSuffix(module, "index")
	module = strings.TrimSuffix(module, "/")
	return
}
