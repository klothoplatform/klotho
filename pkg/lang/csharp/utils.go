package csharp

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/filter"
	"github.com/klothoplatform/klotho/pkg/query"
	sitter "github.com/smacker/go-tree-sitter"
	"strings"
)

// normalizedStringContent returns the string literal formatted content for string literal and verbatim string literal nodes
func normalizedStringContent(node *sitter.Node) string {
	if node == nil {
		return ""
	}
	content := node.Content()
	// raw string literals are not supported by tree-sitter-c-sharp until v0.21.0 is released
	isRawStringLiteral := strings.HasPrefix(content, `"""`)
	isVerbatimStringLiteral := strings.HasPrefix(content, "@")
	content = strings.TrimPrefix(content, "@")
	content = strings.TrimPrefix(content, `"`)
	content = strings.TrimSuffix(content, `"`)
	if isRawStringLiteral {
		content = strings.TrimPrefix(content, `""`)
		content = strings.TrimSuffix(content, `""`)
		content = strings.ReplaceAll(content, `\`, `\\`)
		content = strings.ReplaceAll(content, `"`, `\"`)
	} else if isVerbatimStringLiteral {
		content = strings.ReplaceAll(content, `\`, `\\`)
		content = strings.ReplaceAll(content, `""`, `\"`)
	}
	return content
}

// ContainingNamespaces returns the set of namespaces surrounding the current node.
func ContainingNamespaces(node *sitter.Node) map[string]struct{} {
	var namespaces []string
	for _, ns := range query.AncestorsOfType(node, "namespace_declaration") {
		namespaces = append([]string{ns.ChildByFieldName("name").Content()}, namespaces...)
	}
	qualifiedNamespaces := make(map[string]struct{})
	for i := range namespaces {
		qualifiedNamespaces[strings.Join(namespaces[0:i+1], ".")] = struct{}{}
	}
	return qualifiedNamespaces
}

func IsValidTypeName(nameNode *sitter.Node, expectedNamespace, expectedType string) bool {
	qualifiedExpectedType := expectedNamespace + "." + expectedType
	actualName := nameNode.Content()

	if actualName == qualifiedExpectedType {
		return true
	}

	root := query.FirstAncestorOfType(nameNode, "compilation_unit")

	// check if type with name actualName is declared in the same file and namespace
	actualNamespace := resolveNamespace(nameNode)
	declarations := filter.NewSimpleFilter(
		IsInNamespace[*TypeDeclaration](actualNamespace),
		HasName[*TypeDeclaration](actualName),
	).Apply(FindDeclarationsAtNode[*TypeDeclaration](root).Declarations()...)
	if len(declarations) == 1 {
		return true
	}

	// check if type is available in imported namespace
	validLocalNamespaces := ContainingNamespaces(nameNode)
	imports := FindImportsAtNode(root)
	nsImports := imports[expectedNamespace]
	for _, nsImport := range nsImports {
		if nsImport.Type == ImportTypeUsingAlias && actualName == nsImport.Alias+"."+expectedType {
			_, ok := validLocalNamespaces[nsImport.Namespace]
			return nsImport.Namespace == "" || ok
		} else if expectedType == actualName {
			if _, ok := validLocalNamespaces[nsImport.Namespace]; nsImport.Namespace == "" || ok {
				return true
			}
		}
	}

	// check if type matches aliased type import
	typeImports := imports[qualifiedExpectedType]
	for _, typeImport := range typeImports {
		if _, ok := validLocalNamespaces[typeImport.Namespace]; ok || typeImport.Namespace == "" {
			if actualName == typeImport.ImportedAs() {
				return true
			}
		}
	}

	// check if type belongs to aliased namespace
	if strings.ContainsRune(expectedType, '.') {
		endParentIndex := strings.LastIndex(expectedType, ".")
		parentClass := expectedType[0:endParentIndex]
		child := expectedType[endParentIndex+1:]
		parentImports := imports[parentClass]
		for _, pImport := range parentImports {
			if actualName == pImport.ImportedAs()+"."+child {
				return true
			}
		}
	}

	return false
}

func splitQualifiedName(qualifiedName string) (scope string, name string) {
	separator := strings.LastIndex(qualifiedName, ".")

	if separator != -1 {
		scope = qualifiedName[0:separator]
		name = qualifiedName[separator+1:]
	} else {
		name = qualifiedName
	}
	return scope, name
}

func FindSubtypes(unit *core.ExecutionUnit, baseNamespace, baseType string) []*TypeDeclaration {
	var declarations []*TypeDeclaration
	for _, csFile := range unit.FilesOfLang(CSharp) {
		types := FindDeclarationsInFile[*TypeDeclaration](csFile).Declarations()
		for _, t := range types {
			for _, bNode := range t.Bases {
				if IsValidTypeName(bNode, baseNamespace, baseType) {
					declarations = append(declarations, t)
					break
				}
			}
		}
	}
	return declarations
}
