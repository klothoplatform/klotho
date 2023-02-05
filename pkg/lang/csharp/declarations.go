package csharp

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/query"
	sitter "github.com/smacker/go-tree-sitter"
	"strings"
)

type Declaration struct {
	Node          *sitter.Node
	Name          string
	Namespace     string
	DeclaringFile string
	Kind          DeclarationKind
	Visibility    Visibility
	QualifiedName string
	IsGeneric     bool
	IsNested      bool
}

func (d Declaration) AsDeclaration() Declaration {
	return d
}

type TypeDeclaration struct {
	Declaration
	Bases      []string
	IsAbstract bool
	IsSealed   bool
}

type DeclarationKind string

const (
	DeclarationKindClass     = DeclarationKind("class")
	DeclarationKindInterface = DeclarationKind("interface")
	DeclarationKindRecord    = DeclarationKind("record")
	DeclarationKindStruct    = DeclarationKind("struct")
	DeclarationKindField     = DeclarationKind("field")
	DeclarationKindProperty  = DeclarationKind("property")
	DeclarationKindMethod    = DeclarationKind("method")
)

type Visibility string

const (
	VisibilityPublic            = Visibility("public")
	VisibilityProtected         = Visibility("protected")
	VisibilityInternal          = Visibility("internal")
	VisibilityPrivate           = Visibility("private")
	VisibilityProtectedInternal = Visibility("protected_internal")
	VisibilityPrivateProtected  = Visibility("private_protected")
)

type MethodDeclaration struct {
	Declaration
	IsStatic   bool
	IsAbstract bool
	IsSealed   bool
	Parameters []Parameter
	ReturnType string
}

type Parameter struct {
	Name string
	Type string
}

// Declarable simplifies the process of working with various declaration kinds simultaneously
type Declarable interface {
	AsDeclaration() Declaration
}

type NamespaceTypes map[string][]TypeDeclaration
type NamespaceMethods map[string][]MethodDeclaration

func (nsm NamespaceMethods) Methods() []MethodDeclaration {
	var allMethods []MethodDeclaration
	for _, nsMethods := range nsm {
		for _, m := range nsMethods {
			allMethods = append(allMethods, m)
		}
	}
	return allMethods
}

func (nst NamespaceTypes) Types() []TypeDeclaration {
	var allTypes []TypeDeclaration
	for _, nsTypes := range nst {
		for _, t := range nsTypes {
			allTypes = append(allTypes, t)
		}
	}
	return allTypes
}
func FindTypeDeclarationsInFile(file *core.SourceFile) NamespaceTypes {
	nsTypeDeclarations := FindTypeDeclarationsAtNode(file.Tree().RootNode())
	for _, declarations := range nsTypeDeclarations {
		for i, d := range declarations {
			d.DeclaringFile = file.Path()
			declarations[i] = d
		}
	}
	return nsTypeDeclarations
}

// FindTypeDeclarationsAtNode returns a map containing a list of imports for each import source starting from the supplied node.
func FindTypeDeclarationsAtNode(node *sitter.Node) NamespaceTypes {
	namespaceTypes := NamespaceTypes{}
	matches := AllMatches(node, typeDeclarations)
	for _, match := range matches {
		parsedType := parseTypeDeclaration(match)
		i := namespaceTypes[parsedType.Namespace]
		namespaceTypes[parsedType.Namespace] = append(i, parsedType)
	}

	return namespaceTypes
}

func FindTypeDeclarationAtNode(node *sitter.Node) (TypeDeclaration, bool) {
	nextMatch := DoQuery(node, typeDeclarations)
	if match, found := nextMatch(); found {
		return parseTypeDeclaration(match), true
	}

	return TypeDeclaration{}, false
}

func parseTypeDeclaration(match query.MatchNodes) TypeDeclaration {
	classDeclaration := match["class_declaration"]
	interfaceDeclaration := match["interface_declaration"]
	recordDeclaration := match["record_declaration"]
	structDeclaration := match["struct_declaration"]
	name := match["name"]
	bases := match["bases"]

	declaration := TypeDeclaration{
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

	modifiers := parseModifiers(declaration.Node)
	declaration.Visibility = modifiers.Visibility
	declaration.IsAbstract = modifiers.IsAbstract
	declaration.IsSealed = modifiers.IsSealed
	declaration.IsNested = query.FirstAncestorOfType(declaration.Node, "class_declaration") != nil
	declaration.QualifiedName = resolveQualifiedName(declaration.Node)
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

func FindMethodDeclarationsInFile(file *core.SourceFile) NamespaceMethods {
	namespaceMethods := FindMethodDeclarationsAtNode(file.Tree().RootNode())
	for _, declarations := range namespaceMethods {
		for i, d := range declarations {
			d.DeclaringFile = file.Path()
			declarations[i] = d
		}
	}
	return namespaceMethods
}

func FindMethodDeclarationsAtNode(node *sitter.Node) NamespaceMethods {
	namespaceMethods := NamespaceMethods{}
	matches := AllMatches(node, methodDeclarations)
	for _, match := range matches {
		parsedMethod := parseMethodDeclaration(match)
		i := namespaceMethods[parsedMethod.Namespace]
		namespaceMethods[parsedMethod.Namespace] = append(i, parsedMethod)
	}

	return namespaceMethods
}

func parseMethodDeclaration(match query.MatchNodes) MethodDeclaration {
	methodDeclaration := match["method_declaration"]
	returnType := match["return_type"]
	name := match["name"]
	parameters := match["parameters"]

	declaration := MethodDeclaration{
		Declaration: Declaration{
			Name: name.Content(),
			Node: methodDeclaration,
			Kind: DeclarationKindMethod,
		},
		ReturnType: returnType.Content(),
	}

	declaration.Namespace = resolveNamespace(declaration.Node)

	modifiers := parseModifiers(declaration.Node)
	declaration.Visibility = modifiers.Visibility
	declaration.IsAbstract = modifiers.IsAbstract
	declaration.IsSealed = modifiers.IsSealed
	declaration.IsStatic = modifiers.IsStatic
	declaration.IsGeneric = declaration.Node.ChildByFieldName("type_parameters") != nil
	declaration.IsNested = query.FirstAncestorOfType(declaration.Node, "class_declaration") != nil
	declaration.QualifiedName = resolveQualifiedName(declaration.Node)
	declaration.Parameters = parseMethodParameters(parameters)

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
				Name: child.ChildByFieldName("name").Content(),
				Type: child.ChildByFieldName("type").Content(),
			})
		}
	}
	return parameters
}

type modifierSpec struct {
	Visibility Visibility
	IsSealed   bool
	IsAbstract bool
	IsStatic   bool
}

func parseModifiers(declaration *sitter.Node) modifierSpec {
	if declaration == nil {
		return modifierSpec{}
	}
	spec := modifierSpec{}

	for i := 0; i < int(declaration.ChildCount()); i++ {
		child := declaration.Child(i)
		if child.Type() != "modifier" {
			continue
		}

		// C# Visibility: https://learn.microsoft.com/en-us/dotnet/csharp/programming-guide/classes-and-structs/access-modifiers
		switch child.Content() {
		case "private":
			if spec.Visibility == VisibilityProtected {
				spec.Visibility = VisibilityPrivateProtected
			} else {
				spec.Visibility = VisibilityPrivate
			}
		case "protected":
			switch spec.Visibility {
			case VisibilityPrivate:
				spec.Visibility = VisibilityPrivateProtected
			case VisibilityInternal:
				spec.Visibility = VisibilityProtectedInternal
			default:
				spec.Visibility = VisibilityProtected
			}
		case "internal":
			if spec.Visibility == VisibilityProtected {
				spec.Visibility = VisibilityProtectedInternal
			} else {
				spec.Visibility = VisibilityInternal
			}
		case "public":
			spec.Visibility = VisibilityPublic
		case "sealed":
			spec.IsSealed = true
		case "abstract":
			spec.IsAbstract = true
		case "static":
			spec.IsStatic = true
		}
	}
	return spec
}
func parseBaseTypes(baseList *sitter.Node) []string {
	if baseList == nil || baseList.Type() != "base_list" {
		return nil
	}

	var bases []string
	for i := 0; i < int(baseList.ChildCount()); i++ {
		child := baseList.Child(i)
		if t := child.Type(); t == "qualified_name" || t == "identifier" {
			bases = append(bases, child.Content())
		}
	}
	return bases
}

func resolveNamespace(declaration *sitter.Node) string {
	var parents []string
	if declaration == nil {
		return ""
	}

	for node := declaration.Parent(); node != nil; node = node.Parent() {
		if node.Type() == "namespace_declaration" {
			parents = append([]string{node.ChildByFieldName("name").Content()}, parents...)
		}
	}
	return strings.Join(parents, ".")
}

func resolveQualifiedName(declaration *sitter.Node) string {
	if declaration == nil {
		return ""
	}

	components := []string{declaration.ChildByFieldName("name").Content()}

	for node := declaration.Parent(); node != nil; node = node.Parent() {
		if name := node.ChildByFieldName("name"); name != nil {
			components = append([]string{name.Content()}, components...)
		}
	}
	return strings.Join(components, ".")
}
