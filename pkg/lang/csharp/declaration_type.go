package csharp

import (
	"github.com/klothoplatform/klotho/pkg/filter/predicate"
	"github.com/klothoplatform/klotho/pkg/query"
	sitter "github.com/smacker/go-tree-sitter"
	"strings"
)

// TypeDeclaration represents a type declaration.
// Type declarations include classes, interfaces, records, and structs.
type TypeDeclaration struct {
	Declaration
	Bases map[string]*sitter.Node
}

func parseTypeDeclaration(match query.MatchNodes) *TypeDeclaration {
	classDeclaration := match["class_declaration"]
	interfaceDeclaration := match["interface_declaration"]
	recordDeclaration := match["record_declaration"]
	structDeclaration := match["struct_declaration"]
	name := match["name"]
	bases := match["bases"]

	declaration := &TypeDeclaration{
		Declaration: Declaration{
			Name: name.Content(),
		},
		Bases: parseBaseTypes(bases),
	}

	if classDeclaration != nil {
		declaration.Node = classDeclaration
		declaration.Kind = DeclarationKindClass
	} else if interfaceDeclaration != nil {
		declaration.Node = interfaceDeclaration
		declaration.Kind = DeclarationKindInterface
	} else if recordDeclaration != nil {
		declaration.Node = recordDeclaration
		declaration.Kind = DeclarationKindRecord
	} else if structDeclaration != nil {
		declaration.Node = structDeclaration
		declaration.Kind = DeclarationKindStruct
	}

	declaration.Namespace = resolveNamespace(declaration.Node)
	declaration.QualifiedName = resolveQualifiedName(declaration.Node)
	declaration.DeclaringClass = declaringClass(declaration.Node, declaration.QualifiedName)
	declaration.Visibility = parseVisibilityModifiers(declaration.Node)
	declaration.IsNested = isNested(declaration.Node)
	declaration.IsGeneric = declaration.Node.ChildByFieldName("type_parameters") != nil

	// handle default visibility
	if declaration.Visibility == "" {
		if declaration.IsNested {
			declaration.Visibility = VisibilityPrivate
		} else {
			declaration.Visibility = VisibilityInternal
		}
	}

	return declaration
}

// parseBaseTypes returns a mapping between the supplied baseList's base names and their respective nodes.
func parseBaseTypes(baseList *sitter.Node) map[string]*sitter.Node {
	if baseList == nil || baseList.Type() != "base_list" {
		return nil
	}

	bases := make(map[string]*sitter.Node)
	for i := 0; i < int(baseList.NamedChildCount()); i++ {
		child := baseList.NamedChild(i)
		if t := child.Type(); t == "qualified_name" || t == "identifier" {
			bases[child.Content()] = child
		}
	}
	return bases
}

// HasBase is a predicate that evaluates whether a *TypeDeclaration's has a specific base.
func HasBase(namespace, typeName string, using Imports) predicate.Predicate[*TypeDeclaration] {
	qualifiedName := namespace + "." + typeName
	return func(d *TypeDeclaration) bool {
		if _, ok := d.Bases[qualifiedName]; ok {
			return true
		}
		for _, baseNode := range d.Bases {
			if IsValidTypeName(baseNode, namespace, typeName) {
				return true
			}
		}
		return false
	}
}

// HasBaseWithSuffix is a predicate that evaluates whether any of a *TypeDeclaration's bases has the supplied suffix.
func HasBaseWithSuffix(suffix string) predicate.Predicate[*TypeDeclaration] {
	return func(d *TypeDeclaration) bool {
		for name := range d.Bases {
			if strings.HasSuffix(name, suffix) {
				return true
			}
		}
		return false
	}
}
