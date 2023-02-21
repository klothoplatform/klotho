package csharp

import (
	"github.com/klothoplatform/klotho/pkg/query"
)

type FieldDeclaration struct {
	Declaration
	DeclaringClass  string
	HasInitialValue bool
	Type            string
}

func parseFieldDeclaration(match query.MatchNodes) *FieldDeclaration {
	fieldDeclaration := match["field_declaration"]
	fieldType := match["type"]
	name := match["name"]
	equalsValueClause := match["equals_value_clause"]

	declaration := &FieldDeclaration{
		Declaration: Declaration{
			Name: name.Content(),
			Node: fieldDeclaration,
			Kind: DeclarationKindField,
		},
		Type: fieldType.Content(),
	}

	declaration.Namespace = resolveNamespace(declaration.Node)
	declaration.Visibility = parseVisibilityModifiers(declaration.Node)
	declaration.IsGeneric = fieldType.Type() == "generic_name"
	declaration.IsNested = isNested(declaration.Node)
	declaration.QualifiedName = resolveQualifiedName(declaration.Node)
	declaration.DeclaringClass = declaringClass(declaration.Node, declaration.QualifiedName)

	// handle default visibility
	if declaration.Visibility == "" {
		if declaration.IsNested {
			declaration.Visibility = VisibilityPrivate
		} else {
			declaration.Visibility = VisibilityInternal
		}
	}

	if equalsValueClause != nil {
		declaration.HasInitialValue = true
	}

	return declaration
}
