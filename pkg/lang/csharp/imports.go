package csharp

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/filter"
	"github.com/klothoplatform/klotho/pkg/filter/predicate"
	"github.com/klothoplatform/klotho/pkg/query"
	sitter "github.com/smacker/go-tree-sitter"
	"regexp"
)

type ImportType string

const (
	ImportTypeUsing       = ImportType("using")
	ImportTypeUsingAlias  = ImportType("using_alias")
	ImportTypeUsingStatic = ImportType("using_static")
)

type ImportScope string

const (
	ImportScopeGlobal    = ImportScope("global")
	ImportScopeFile      = ImportScope("file")
	ImportScopeNamespace = ImportScope("namespace")
	ImportScopeLocal     = ImportScope("local")
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

// FindImportsInFile returns a map containing a list of imports for each import source referenced within the file.
func FindImportsInFile(file *core.SourceFile) Imports {
	return FindImportsAtNode(file.Tree().RootNode())
}

// FindImportsAtNode returns a map containing a list of imports for each import source starting from the supplied node.
func FindImportsAtNode(node *sitter.Node) Imports {
	fileImports := Imports{}
	matches := queryImports(node)
	for _, match := range matches {
		parsedImport := parseUsingDirective(match)
		i := fileImports[parsedImport.Name]
		fileImports[parsedImport.Name] = append(i, parsedImport)

	}
	return fileImports
}

func queryImports(node *sitter.Node) []query.MatchNodes {
	nextMatch := DoQuery(node, usingDirectives)

	var matches []query.MatchNodes
	for {
		if match, found := nextMatch(); found {
			matches = append(matches, match)
		} else {
			break
		}
	}

	return matches
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
		parsedImport.Scope = ImportScopeFile
	}

	if namespace := namespaceAncestor(usingDirective); namespace != nil {
		parsedImport.Scope = ImportScopeNamespace
		parsedImport.Namespace = namespace.ChildByFieldName("name").Content()
	} else if parsedImport.Scope == "" {
		parsedImport.Scope = ImportScopeLocal
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
