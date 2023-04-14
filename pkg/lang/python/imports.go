package python

import (
	"path"
	"strings"

	"github.com/klothoplatform/klotho/pkg/query"
	"go.uber.org/zap"

	"github.com/klothoplatform/klotho/pkg/core"
	sitter "github.com/smacker/go-tree-sitter"
)

// TODO: capture import scope (e.g. IsGlobal/IsLocal)
type Import struct {
	ParentModule string
	Name         string
	// ImportedAttributes is a map from attribute name, to information about that attribute.
	// Given a statement `from foo import bar as the_bar` the attribute name is `bar`.
	ImportedAttributes map[string]Attribute
	Node               *sitter.Node
	ImportedSelf       bool
	Alias              string
}

type Attribute struct {
	Name  string
	Node  *sitter.Node
	Alias string
}

type Imports map[string]Import

// ImportedAs returns the name of the module as it will be used in the (python) code. This is either the (possibly
// qualified) module name, or the alias.
func (imp Import) ImportedAs() string {
	if imp.Alias != "" {
		return imp.Alias
	} else {
		name := imp.Name
		if imp.ParentModule != "" {
			name = imp.ParentModule + "." + name
		}
		return name
	}
}

// FindImports returns a map containing each import statement within the file, as a map keyed by the import's qualified
// name.
//
// For example:
//
//	import module_a                     # key is "module_a"
//	import module_b as the_b            # key is "module_b"
//	import module_c.module_cc as the_c  # key is "module_c.module_cc"
func FindImports(file *core.SourceFile) Imports {
	nextMatch := DoQuery(file.Tree().RootNode(), findImports)
	fileImports := Imports{}
	for {
		match, found := nextMatch()
		if !found {
			break
		}

		// For multi aliased imports, we will support import aiofilesas as b, somethingelse as w
		// For multi aliased imports we do not support import aiofilesas, somethingelse as b w

		standardImport := match["standardImport"]
		fromImport := match["fromImport"]
		module := match["module"]
		importPrefix := match["importPrefix"] // TODO: look into resolving the prefix for a relative import
		attribute := match["attribute"]
		aliasedModule := match["aliasedModule"]
		alias := match["alias"]

		if standardImport != nil && module == nil && aliasedModule == nil {
			continue
		}

		if fromImport != nil && importPrefix == nil && module == nil {
			continue
		}

		prefixContent := ""
		if importPrefix != nil {
			prefixContent = importPrefix.Content()
		}

		moduleContent := ""
		parent := ""
		moduleName := ""
		aliasName := ""

		if module != nil {
			moduleContent = module.Content()
			lastPeriod := strings.LastIndex(moduleContent, ".")

			if lastPeriod != -1 {
				parent = moduleContent[0:lastPeriod]
				moduleName = moduleContent[lastPeriod+1:]
			} else {
				moduleName = moduleContent
			}
		} else if aliasedModule != nil {
			moduleContent = aliasedModule.Content()
			lastPeriod := strings.LastIndex(moduleContent, ".")

			if lastPeriod != -1 {
				parent = moduleContent[0:lastPeriod]
				moduleName = moduleContent[lastPeriod+1:]
			} else {
				moduleName = moduleContent
			}
			aliasName = alias.Content()
		}

		parent = prefixContent + parent
		qualifiedModuleName := parent
		if qualifiedModuleName == "" {
			qualifiedModuleName = moduleName
		} else if module != nil {
			if qualifiedModuleName != "" && qualifiedModuleName[len(qualifiedModuleName)-1] == '.' {
				qualifiedModuleName += moduleName
			} else {
				qualifiedModuleName += "." + moduleName
			}
		}
		i := fileImports[qualifiedModuleName]
		// technically, this may be a submodule, but we can't tell without deeper analysis of the imported file
		if attribute != nil {
			attributeName := attribute.Content()
			ia := i.ImportedAttributes
			if ia == nil {
				ia = map[string]Attribute{}
			}
			aliasFrom := ""
			if alias != nil {
				aliasFrom = alias.Content()
			}
			ia[attributeName] = Attribute{
				Name:  attributeName,
				Node:  attribute,
				Alias: aliasFrom,
			}
			i.ImportedAttributes = ia
		} else {
			i.ImportedSelf = true
			i.Node = module
		}

		i.ParentModule = parent
		i.Name = moduleName
		i.Alias = aliasName

		fileImports[qualifiedModuleName] = i
	}
	return fileImports
}

func UnitFileDependencyResolver(unit *core.ExecutionUnit) (core.FileDependencies, error) {
	return ResolveFileDependencies(unit.Files())
}

func ResolveFileDependencies(files map[string]core.File) (core.FileDependencies, error) {
	fileDeps := make(core.FileDependencies) // map of [importing file path] -> Imported
	for filePath, file := range files {
		pyFile, isPy := Language.ID.CastFile(file)
		if !isPy {
			continue
		}
		pyRoot := findPyRoot(filePath, files)
		imported := make(core.Imported) // map of [imported file path] -> References
		fileDeps[filePath] = imported

		// minimal logic for adding a dependency on __init__.py
		// TODO: find __init__.py refs (typically used in the form <package>.<name>)
		initPy := path.Dir(filePath) + "/__init__.py"
		if _, ok := files[initPy]; ok {
			imported[initPy] = core.References{}
		}

		imports := FindImports(pyFile)
		for moduleName, importSpec := range imports {
			if importedFile := findImportedFile(moduleName, filePath, pyRoot, files); importedFile != "" {
				var refs core.References // set of [referenced attributes]
				if currentRefs, found := imported[importedFile]; found {
					refs = currentRefs
				} else {
					refs = make(core.References)
					imported[importedFile] = refs
				}
				if importSpec.ImportedSelf {
					nextAttrUsage := DoQuery(pyFile.Tree().RootNode(), FindQualifiedAttrUsage)
					for {
						attrUsage, found := nextAttrUsage()
						if !found {
							break
						}
						objName, attrName := attrUsage["obj_name"], attrUsage["attr_name"]
						if query.NodeContentEquals(objName, importSpec.ImportedAs()) {
							attrNameStr := attrName.Content()
							refs[attrNameStr] = struct{}{}
						}
					}
				} else {
					for _, attr := range importSpec.ImportedAttributes {
						if attr.Alias != "" {
							refs[attr.Alias] = struct{}{}
						} else {
							refs[attr.Name] = struct{}{}
						}
					}
				}
			} else {
				zap.S().Debugf(`couldn't find file for module [%v] within [%v]`, moduleName, filePath)
			}
		}
	}
	return fileDeps, nil
}

// findImportedFile takes a python module name, and returns the path to the file within the specified file set.
// It returns an empty string if the file doesn't exist in the set.
//
// The value of the fileSet is ignored; we treat the map as if it were a map[string]struct{}.
func findImportedFile[V any](moduleName string, relativeToFilePath string, root string, fileSet map[string]V) string {
	// We're going to set two vars: moduleDir is a dir path, and moduleName will be stripped of any prefixed dots.
	// That way, we'll have the state such that the py file (if it exists) will be at
	// "${moduleDir}/${moduleName}.py" *except* that moduleName itself may have dots (e.g. it could be "foo.bar"),
	// and each one of those should get translated to a path delimiter.
	var moduleDir string
	if strings.HasPrefix(moduleName, ".") {
		// This could be .foo, ..foo, etc. The first dot is the relative dir, and subsequent ones are up a dir.

		// Note: moduleDir is not a dir initially, but will be after the first iteration, and we're guaranteed to have
		// at least one iteration (because of the HasPrefix check above).
		moduleDir = relativeToFilePath
		for strings.HasPrefix(moduleName, ".") {
			if moduleDir == "." {
				// can't go up any more dirs!
				return ""
			}
			moduleName = strings.TrimPrefix(moduleName, ".")
			moduleDir = path.Dir(moduleDir)
		}
	} else {
		moduleDir = ""
	}
	if moduleDir == "." {
		moduleDir = "" // we want to generate "foo.py", not "./foo.py"
	}
	if moduleName == "" {
		// The original import was something like "from .. import foo". Look for an __init__.py in moduleDir
		moduleName = "__init__" // (the ".py" gets added below)
	}
	modulePath := strings.ReplaceAll(moduleName, ".", "/")
	expectFile := path.Join(moduleDir, modulePath) + ".py"
	if _, exists := fileSet[expectFile]; exists {
		return expectFile
	} else {
		return ""
	}
}

// findPyRoot makes takes a guess at where the python files are located. It returns the highest-level dir that contains
// all .py files (in it or its sub-dirs), or "" if there is not one dir that matches that.
func findPyRoot(relativeToFilePath string, files map[string]core.File) string {
	best := strings.Split(path.Dir(relativeToFilePath), "/")
	for filePath, file := range files {
		if _, isPy := Language.ID.CastFile(file); !isPy {
			continue
		}
		dir := path.Dir(filePath)
		dirSegments := strings.Split(dir, "/")
		i := 0
		for ; i < len(dirSegments) && i < len(best); i++ {
			if best[i] != dirSegments[i] {
				break
			}
		}
		// "i" is now the index one *after* the last match, or 0 if there was no match. If there's no match, just ignore
		// this file: it's in a different top-level dir than relativeTo.
		if i > 0 {
			best = best[:i]
		}
	}
	return strings.Join(best, "/")
}
