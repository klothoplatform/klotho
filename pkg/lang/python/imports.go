package python

import (
	"fmt"
	"path"
	"strings"

	"go.uber.org/zap"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/logging"
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

func (imp Import) ModuleDir() string {
	moduleRoot := ""
	if imp.ParentModule != "" {
		moduleRoot = imp.ParentModule
	}
	if imp.Name != "" {
		if moduleRoot != "" {
			moduleRoot += "."
		}
		moduleRoot += imp.Name
	}
	return moduleRoot
}

func (attr Attribute) UsedAs() string {
	if attr.Alias != "" {
		return attr.Alias
	} else {
		return attr.Name
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
		} else if strings.HasSuffix(qualifiedModuleName, ".") {
			qualifiedModuleName += moduleName
		} else {
			qualifiedModuleName += "." + moduleName
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
		imported := make(core.Imported) // map of [imported file path] -> References
		fileDeps[filePath] = imported
		log := zap.S().With(logging.FileField(file))

		// minimal logic for adding a dependency on __init__.py
		// TODO: find __init__.py refs (typically used in the form <package>.<name>)
		initPy := path.Dir(filePath) + "/__init__.py"
		if _, ok := files[initPy]; ok {
			imported[initPy] = core.References{}
		}

		imports := FindImports(pyFile)
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
func dependenciesForImport(relativeToPath string, spec Import, files map[string]core.File) (core.Imported, error) {
	deps := make(core.Imported)

	moduleRoot := spec.ModuleDir()

	importedModule, err := findImportedFile(spec.Name, relativeToPath, files)
	if err != nil {
		return nil, err
	}

	if importedModule == "" {
		importedModule, err = findImportedFile(moduleRoot, relativeToPath, files)
		if err != nil {
			return nil, err
		}
	}

	if importedModule != "" {
		refs, ok := deps[importedModule]
		if !ok {
			refs = make(core.References)
			deps[importedModule] = refs
		}

		if len(spec.ImportedAttributes) == 0 {
			sourceFile := files[relativeToPath].(*core.SourceFile)
			importRefs := referencesForImport(sourceFile.Tree().RootNode(), spec.ImportedAs())
			refs.AddAll(importRefs)
		} else {
			for _, attr := range spec.ImportedAttributes {
				refs.Add(attr.Name)
			}
		}
		return deps, nil
	}

	if moduleRoot != "" && !strings.HasSuffix(moduleRoot, ".") {
		moduleRoot += "."
	}
	for _, attr := range spec.ImportedAttributes {
		modulePath, err := findImportedFile(moduleRoot+attr.Name, relativeToPath, files)
		if err != nil {
			return nil, err
		}
		if modulePath == "" {
			continue
		}
		moduleFile := files[modulePath].(*core.SourceFile)
		refs, ok := deps[modulePath]
		if !ok {
			refs = make(core.References)
			deps[modulePath] = refs
		}

		importRefs := referencesForImport(moduleFile.Tree().RootNode(), attr.UsedAs())
		refs.AddAll(importRefs)
	}

	return deps, nil
}

// referencesForImport returns all references of importModule within the program. If the import is aliased, importModule should be the alias
// and not the real module name.
func referencesForImport(program *sitter.Node, importModule string) core.References {
	refs := make(core.References)
	nextAttrUsage := DoQuery(program, FindQualifiedAttrUsage)
	for {
		attrUsage, found := nextAttrUsage()
		if !found {
			break
		}
		objName, attrName := attrUsage["obj_name"], attrUsage["attr_name"]
		if objName.Content() == importModule {
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

	if dotCount > 1 {
		for i := 0; i < dotCount-1; i++ {
			if moduleDir == "." {
				return "", fmt.Errorf("can't go up %v dirs from %v (module '%s')", dotCount, relativeToFilePath, module)
			}
			moduleDir = path.Dir(moduleDir)
		}
	}

	return path.Join(moduleDir, modulePath) + ".py", nil
}
