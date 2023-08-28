package python

import (
	"fmt"
	"path"
	"strings"

	"go.uber.org/zap"

	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/klothoplatform/klotho/pkg/io"
	"github.com/klothoplatform/klotho/pkg/logging"
	sitter "github.com/smacker/go-tree-sitter"
)

// TODO: capture import scope (e.g. IsGlobal/IsLocal)
type Import struct {
	ParentModule string
	// Name is the unqualified name of the actual module. This is not necessarily the name by which the python source
	// will refer to this module, because of aliases and qualified imports. For any analysis of the user's source code,
	// you should use UsedAs instead.
	Name string
	// ImportedAttributes is a map from attribute name, to information about that attribute.
	// Given a statement `from foo import bar as the_bar` the attribute name is `bar`.
	ImportedAttributes map[string]Attribute
	Node               *sitter.Node
	// UsedAs is the names of the module as it will be used in the (python) code. These are either the (possibly
	// qualified) module name, or the aliases. If this module was imported several times, each one will have an entry
	// in this set (removing duplicates, of course).
	UsedAs map[string]struct{}
}

type Attribute struct {
	// Name is the module attribute's name. This is not necessarily the name by which the python source will refer to
	// the attribute, because of aliases. For any analysis of the user's source code, you should use UsedAs instead.
	Name string
	Node *sitter.Node
	// UsedAs is the same as [Import.UsedAs], but for attributes
	UsedAs map[string]struct{}
}

type Imports map[string]Import

func (imp Import) FullyQualifiedModule() string {
	moduleRoot := imp.ParentModule
	if imp.Name != "" {
		if moduleRoot != "" && !strings.HasSuffix(moduleRoot, ".") {
			// Only add a '.' delimiter if one doesn't already exist.
			// It should only exist for `.ParentModule` in ['.' and '..'] cases
			moduleRoot += "."
		}
		moduleRoot += imp.Name
	}
	return moduleRoot
}

func FindFileImports(file *types.SourceFile) Imports {
	return FindImports(file.Tree().RootNode())
}

// FindImports returns a map containing each import statement within the file, as a map keyed by the import's qualified
// name.
//
// For example:
//
//	import module_a                     # key is "module_a"
//	import module_b as the_b            # key is "module_b"
//	import module_c.module_cc as the_c  # key is "module_c.module_cc"
func FindImports(node *sitter.Node) Imports {
	nextMatch := DoQuery(node, findImports)
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
		} else if strings.HasSuffix(qualifiedModuleName, ".") {
			qualifiedModuleName += moduleName
		} else {
			qualifiedModuleName += "." + moduleName
		}
		i := fileImports[qualifiedModuleName]
		// this may be a submodule, but we can't tell without deeper analysis of the imported file
		if attribute != nil {
			attributeName := attribute.Content()
			ia := i.ImportedAttributes
			if ia == nil {
				ia = map[string]Attribute{}
			}
			// Upsert a "UsedAs" map for this attribute
			var attributeAliases map[string]struct{}
			if existingIa, exists := ia[attributeName]; exists {
				attributeAliases = existingIa.UsedAs
			}
			if attributeAliases == nil {
				attributeAliases = make(map[string]struct{})
			}
			// Insert either the attribute name, or the alias if it exists
			if alias == nil {
				attributeAliases[attributeName] = struct{}{}
			} else {
				attributeAliases[alias.Content()] = struct{}{}
			}
			ia[attributeName] = Attribute{
				Name:   attributeName,
				Node:   attribute,
				UsedAs: attributeAliases,
			}
			i.ImportedAttributes = ia
		} else {
			i.Node = module
		}

		i.ParentModule = parent
		i.Name = moduleName

		if len(i.ImportedAttributes) == 0 {
			if aliasName == "" {
				// convert it into the (possibly qualified) module name.
				aliasName = moduleName
				if parent != "" {
					aliasName = parent + "." + aliasName
				}
			}
			if i.UsedAs == nil {
				i.UsedAs = make(map[string]struct{})
			}
			i.UsedAs[aliasName] = struct{}{}
		}

		fileImports[qualifiedModuleName] = i
	}
	return fileImports
}

func UnitFileDependencyResolver(unit *types.ExecutionUnit) (types.FileDependencies, error) {
	return ResolveFileDependencies(unit.Files())
}

func ResolveFileDependencies(files map[string]io.File) (types.FileDependencies, error) {
	fileDeps := make(types.FileDependencies) // map of [importing file path] -> Imported
	for filePath, file := range files {
		pyFile, isPy := Language.ID.CastFile(file)
		if !isPy {
			continue
		}
		imported := make(types.Imported) // map of [imported file path] -> References
		fileDeps[filePath] = imported
		log := zap.S().With(logging.FileField(file))

		// minimal logic for adding a dependency on __init__.py
		// TODO: find __init__.py refs (typically used in the form <package>.<name>)
		initPy := path.Dir(filePath) + "/__init__.py"
		if _, ok := files[initPy]; ok {
			imported[initPy] = types.References{}
		}

		imports := FindFileImports(pyFile)
		for _, importSpec := range imports {
			deps, err := dependenciesForImport(filePath, importSpec, files)
			if err != nil {
				return nil, err
			}
			if len(deps) == 0 {
				log.Debugf(`couldn't find file for module %+v`, importSpec)
			}
			imported.AddAll(deps)
		}
		log.Debugf("found imports: %v", imported)
	}
	return fileDeps, nil
}

// dependenciesForImport returns all imports specified by spec
func dependenciesForImport(relativeToPath string, spec Import, files map[string]io.File) (types.Imported, error) {
	deps := make(types.Imported)
	moduleDir := spec.FullyQualifiedModule()
	importerFile := files[relativeToPath].(*types.SourceFile)

	rootModule, err := findImportedFile(moduleDir, relativeToPath, files)
	if err != nil {
		return nil, err
	}
	if rootModule != "" {
		deps[rootModule] = referencesForImport(importerFile.Tree().RootNode(), spec.UsedAs)
	}

	for _, attr := range spec.ImportedAttributes {
		attrImport := Import{ParentModule: moduleDir, Name: attr.Name}
		attrModule, err := findImportedFile(attrImport.FullyQualifiedModule(), relativeToPath, files)
		if err != nil {
			return nil, err
		}
		if attrModule == "" {
			if rootRefs, ok := deps[rootModule]; ok {
				rootRefs.Add(attr.Name)
			}
		} else {
			deps[attrModule] = referencesForImport(importerFile.Tree().RootNode(), attr.UsedAs)
		}
	}

	return deps, nil
}

// referencesForImport returns all references of importModule within the program. If the import is aliased, importModule should be the alias
// and not the real module name.
func referencesForImport(program *sitter.Node, importModules map[string]struct{}) types.References {
	refs := make(types.References)
	nextAttrUsage := DoQuery(program, FindQualifiedAttrUsage)
	for {
		attrUsage, found := nextAttrUsage()
		if !found {
			break
		}
		objName, attrName := attrUsage["obj_name"], attrUsage["attr_name"]
		if _, found := importModules[objName.Content()]; found {
			attrNameStr := attrName.Content()
			refs[attrNameStr] = struct{}{}
		}
	}
	return refs
}

// findImportedFile takes a python module name, and returns the path to the file within the specified file set.
// It returns an empty string if the file doesn't exist in the set.
//
// The value of the fileSet is ignored; we treat the map as if it were a map[string]struct{}.
func findImportedFile[V any](moduleName string, relativeToFilePath string, fileSet map[string]V) (string, error) {
	modulePath, err := pythonModuleToPath(moduleName, relativeToFilePath)
	if err != nil {
		return "", err
	}

	if _, ok := fileSet[modulePath]; ok {
		return modulePath, nil
	}

	modulePath = strings.Replace(modulePath, ".py", "/__init__.py", 1)

	if _, ok := fileSet[modulePath]; ok {
		return modulePath, nil
	}

	return "", nil
}

// pythonModuleToPath converts a python module name to a file path, using relativeToFilePath as the base path for relative modules.
func pythonModuleToPath(module string, relativeToFilePath string) (string, error) {
	if !strings.HasPrefix(module, ".") {
		return strings.ReplaceAll(module, ".", "/") + ".py", nil
	}

	dotCount := 0
	for _, c := range module {
		if c == '.' {
			dotCount++
		} else {
			break
		}
	}

	modulePath := strings.ReplaceAll(module[dotCount:], ".", "/")
	moduleDir := path.Dir(relativeToFilePath)

	for i := 0; i < dotCount-1; i++ {
		if moduleDir == "." {
			return "", fmt.Errorf("can't go up %v dirs from %v (module '%s')", dotCount, relativeToFilePath, module)
		}
		moduleDir = path.Dir(moduleDir)
	}

	return path.Join(moduleDir, modulePath) + ".py", nil
}
