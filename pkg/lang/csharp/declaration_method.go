package csharp

import (
	"github.com/klothoplatform/klotho/pkg/query"
	sitter "github.com/smacker/go-tree-sitter"
)

type (
	MethodDeclaration struct {
		Declaration
		Parameters []Parameter
		ReturnType string
	}
	Parameter struct {
		Name     string
		TypeNode *sitter.Node
	}
)

func parseMethodDeclaration(match query.MatchNodes) *MethodDeclaration {
	methodDeclaration := match["method_declaration"]
	returnType := match["return_type"]
	name := match["name"]
	parameters := match["parameters"]

	declaration := &MethodDeclaration{
		Declaration: Declaration{
			Name: name.Content(),
			Node: methodDeclaration,
			Kind: DeclarationKindMethod,
		},
		ReturnType: returnType.Content(),
	}

	declaration.Namespace = resolveNamespace(declaration.Node)
	declaration.Visibility = parseVisibilityModifiers(declaration.Node)
	declaration.IsGeneric = declaration.Node.ChildByFieldName("type_parameters") != nil
	declaration.IsNested = isNested(declaration.Node)
	declaration.QualifiedName = resolveQualifiedName(declaration.Node)
	declaration.Parameters = parseMethodParameters(parameters)
	declaration.DeclaringClass = declaringClass(declaration.Node, declaration.QualifiedName)

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

func parseMethodParameters(parameterList *sitter.Node) []Parameter {
	if parameterList == nil {
		return nil
	}

	var parameters []Parameter
	for i := 0; i < int(parameterList.ChildCount()); i++ {
		child := parameterList.Child(i)
		if child.Type() == "parameter" {
			parameters = append(parameters, Parameter{
				Name:     child.ChildByFieldName("name").Content(),
				TypeNode: child.ChildByFieldName("type"),
			})
		}
	}
	return parameters
}
