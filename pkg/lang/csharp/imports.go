package csharp

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/filter"
	"github.com/klothoplatform/klotho/pkg/filter/predicate"
	"github.com/klothoplatform/klotho/pkg/query"
	sitter "github.com/smacker/go-tree-sitter"
	"regexp"
	"strings"
)

// C# using directive spec: https://learn.microsoft.com/en-us/dotnet/csharp/language-reference/language-specification/namespaces#135-using-directives
// TODO: add support for "extern alias" imports: https://learn.microsoft.com/en-us/dotnet/csharp/language-reference/keywords/extern-alias
type ImportType string

const (
	ImportTypeUsing       = ImportType("using")
	ImportTypeUsingAlias  = ImportType("using_alias")
	ImportTypeUsingStatic = ImportType("using_static")
)

type ImportScope string

const (
	ImportScopeGlobal          = ImportScope("global")
	ImportScopeCompilationUnit = ImportScope("compilation_unit")
	ImportScopeNamespace       = ImportScope("namespace")
)

type Import struct {
	// Name is the exported name of the Import
	Name string

	// Node is the *sitter.Node associated with the Import's using directive
	Node *sitter.Node

	// Alias is the name with which this import is referred to in its enclosing Scope (i.e. module or local)
	Alias string

	Scope     ImportScope
	Type      ImportType
	Namespace string
}

// Imports provides a mapping between import sources and the list of imports for each.
type Imports map[string][]Import

// ImportedAs returns the name of the import as it will be used locally (either the exported name or local alias).
func (p *Import) ImportedAs() string {
	if p.Alias != "" {
		return p.Alias
	}
	return p.Name
}

// Filter applies the supplied Filter to all Import values and returns the filtered list of Import values.
func (imports Imports) Filter(filter filter.Filter[Import]) []Import {
	filteredImports := filter.Apply(imports.AsSlice()...)
	return filteredImports
}

// AsSlice converts an instance of Imports to []Import for simpler iteration over all Import values.
func (imports Imports) AsSlice() []Import {
	var slice []Import
	for _, importsOfSource := range imports {
		slice = append(slice, importsOfSource...)
	}
	return slice
}

// TODO: find a better name
func (imports Imports) HasDeclaration(inputVal, namespace, declaration string) bool {
	qualifiedName := namespace + "." + declaration

	nsImports := imports[namespace]
	for _, nsImport := range nsImports {

		if nsImport.Type == ImportTypeUsingAlias && inputVal == nsImport.Alias+"."+declaration {
			return true
		} else if declaration == inputVal {
			return true
		}
	}

	typeImports := imports[qualifiedName]
	for _, typeImport := range typeImports {
		if inputVal == typeImport.ImportedAs() {
			return true
		}
	}

	if strings.ContainsRune(declaration, '.') {
		endParentIndex := strings.LastIndex(declaration, ".")
		parentClass := declaration[0:endParentIndex]
		child := declaration[endParentIndex+1:]
		parentImports := imports[parentClass]
		for _, pImport := range parentImports {
			if inputVal == pImport.ImportedAs()+"."+child {
				return true
			}
		}
	}

	return inputVal == qualifiedName
}

// FindImportsInFile returns a map containing a list of imports for each import source referenced within the file.
func FindImportsInFile(file *core.SourceFile) Imports {
	return FindImportsAtNode(file.Tree().RootNode())
}

// FindImportsAtNode returns a map containing a list of imports for each import source starting from the supplied node.
func FindImportsAtNode(node *sitter.Node) Imports {
	fileImports := Imports{}
	matches := query.Collect(DoQuery(node, usingDirectives))
	for _, match := range matches {
		parsedImport := parseUsingDirective(match)
		i := fileImports[parsedImport.Name]
		fileImports[parsedImport.Name] = append(i, parsedImport)

	}
	return fileImports
}

func parseUsingDirective(match query.MatchNodes) Import {
	usingDirective, identifier, alias := match["using_directive"], match["identifier"], match["alias"]

	parsedImport := Import{
		Name: identifier.Content(),
		Node: usingDirective,
	}

	if isGlobal(usingDirective) {
		parsedImport.Scope = ImportScopeGlobal
	} else if isFileScoped(usingDirective) {
		parsedImport.Scope = ImportScopeCompilationUnit
	}

	if namespace := namespaceAncestor(usingDirective); namespace != nil {
		parsedImport.Scope = ImportScopeNamespace
		parsedImport.Namespace = namespace.ChildByFieldName("name").Content()
	}

	if isStatic(usingDirective) {
		parsedImport.Type = ImportTypeUsingStatic
	} else if aliasContent := query.NodeContentOrEmpty(alias); aliasContent != "" {
		parsedImport.Type = ImportTypeUsingAlias
		parsedImport.Alias = aliasContent
	} else {
		parsedImport.Type = ImportTypeUsing
	}

	return parsedImport
}

func isGlobal(usingDirective *sitter.Node) bool {
	return query.NodeContentStartWith(usingDirective, "global")
}

func isStatic(usingNode *sitter.Node) bool {
	return query.NodeContentRegex(usingNode, regexp.MustCompile(`\s*static\s`))
}

func isFileScoped(usingNode *sitter.Node) bool {
	return usingNode.Parent().Type() == "compilation_unit"
}

func namespaceAncestor(usingNode *sitter.Node) *sitter.Node {
	return query.FirstAncestorOfType(usingNode, "namespace_declaration")
}

func IsImportOfType(importType ImportType) predicate.Predicate[Import] {
	return func(p Import) bool {
		return p.Type == importType
	}
}

func IsImportInScope(scope ImportScope) predicate.Predicate[Import] {
	return func(p Import) bool {
		return p.Scope == scope
	}
}

func ImportedAs(localName string) predicate.Predicate[Import] {
	return func(p Import) bool {
		return p.ImportedAs() == localName
	}
}

func ImportHasName(name string) predicate.Predicate[Import] {
	return func(p Import) bool {
		return p.Name == name
	}
}

func ascendWhile(node *sitter.Node, predicate predicate.Predicate[*sitter.Node]) *sitter.Node {
	for ; node != nil && predicate(node); node = node.Parent() {
	}
	return node
}

func nodeTypeIs(nodeType string) predicate.Predicate[*sitter.Node] {
	return func(n *sitter.Node) bool {
		return n.Type() == nodeType
	}
}
