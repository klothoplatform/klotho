package csharp

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/filter/predicate"
	"github.com/klothoplatform/klotho/pkg/logging"
	"github.com/klothoplatform/klotho/pkg/query"
	sitter "github.com/smacker/go-tree-sitter"
	"go.uber.org/zap"
	"strings"
)

/* TODO: implement support for:
   - properties
   - constructors
   - events
   - enums
   - indexers
   - local functions
   - local variables
   - delegates
*/

type Declaration struct {
	Node           *sitter.Node
	Name           string
	Namespace      string
	DeclaringFile  string
	Kind           DeclarationKind
	Visibility     Visibility
	QualifiedName  string
	IsGeneric      bool
	IsNested       bool
	IsStatic       bool
	DeclaringClass string
	AttributeList  *sitter.Node
}

func (d *Declaration) AsDeclaration() Declaration {
	return *d
}

func (d *Declaration) SetDeclaringFile(filepath string) {
	d.DeclaringFile = filepath
}

type Attribute struct {
	Name string
	Node *sitter.Node
}

type AttributeArg struct {
	Name  string
	Value string
}

func (a *Attribute) Args() []AttributeArg {
	var args []AttributeArg
	for _, match := range AllMatches(a.Node, declarationAttribute) {
		nameN := match["arg_name"] // may be nil
		valueN := match["arg_value"]

		if valueN == nil {
			continue // match is empty argument list
		}

		value := valueN.Content()
		if strings.Contains(valueN.Type(), "string_literal") {
			value = stringLiteralContent(valueN)
		}

		args = append(args, AttributeArg{
			Name:  query.NodeContentOrEmpty(nameN),
			Value: value,
		})
	}
	return args
}

type Attributes map[string][]Attribute

func (a Attributes) OfType(types ...string) []Attribute {
	var attrs []Attribute
	for _, attrDeclarations := range a {
		for _, t := range types {
			attr := attrDeclarations[0]
			tNamespace, tName := splitQualifiedName(t)

			if IsValidTypeName(attr.Node.ChildByFieldName("name"), tNamespace, tName) {
				attrs = append(attrs, attrDeclarations...)
				break
			}
		}
	}
	return attrs
}
func (d *Declaration) Attributes() Attributes {
	attributes := make(Attributes)
	if d.AttributeList == nil {
		return attributes
	}
	for i := 0; i < int(d.AttributeList.NamedChildCount()); i++ {
		attr := d.AttributeList.NamedChild(i)
		name := attr.ChildByFieldName("name").Content()
		attributes[name] = append(attributes[name], Attribute{
			Name: name,
			Node: attr,
		})
	}
	return attributes
}

type TypeDeclaration struct {
	Declaration
	Bases      map[string]*sitter.Node
	IsAbstract bool
	IsSealed   bool
	IsPartial  bool
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
	DeclarationKindEnum      = DeclarationKind("enum")
	DeclarationKindIndexer   = DeclarationKind("indexer")
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
	DeclaringClass string
	IsAbstract     bool
	IsSealed       bool
	IsPartial      bool
	IsVirtual      bool
	IsOverride     bool
	Parameters     []Parameter
	ReturnType     string
}

type Parameter struct {
	Name     string
	Type     string
	TypeNode *sitter.Node
}

type FieldDeclaration struct {
	Declaration
	DeclaringClass  string
	HasInitialValue bool
	IsConst         bool
	IsReadOnly      bool
	IsRequired      bool
	Type            string
}

type PropertyDeclaration struct {
	Declaration
	IsAbstract bool
	IsReadOnly bool
	IsRequired bool
	IsVirtual  bool
	IsOverride bool
	Type       string
}

// Declarable simplifies the process of working with various typeName kinds simultaneously
type Declarable interface {
	AsDeclaration() Declaration
	SetDeclaringFile(string)
}

type NamespaceDeclarations[T Declarable] map[string][]T

func (nsd NamespaceDeclarations[T]) Declarations() []T {
	var allDeclarations []T
	for _, nsDeclarations := range nsd {
		for _, d := range nsDeclarations {
			allDeclarations = append(allDeclarations, d)
		}
	}
	return allDeclarations
}
func FindDeclarationsInFile[T Declarable](file *core.SourceFile) NamespaceDeclarations[T] {
	nsDeclarations := FindDeclarationsAtNode[T](file.Tree().RootNode())
	for _, declarations := range nsDeclarations {
		for i, d := range declarations {
			d.SetDeclaringFile(file.Path())
			declarations[i] = d
		}
	}
	return nsDeclarations
}

type declarationSpec[T Declarable] struct {
	node      *sitter.Node
	query     string
	parseFunc func(match query.MatchNodes) T
}

// FindDeclarationsAtNode returns a map containing a list of declarations for each namespace starting from the supplied node.
func FindDeclarationsAtNode[T Declarable](node *sitter.Node) NamespaceDeclarations[T] {
	empty := NamespaceDeclarations[T]{}
	switch any(empty).(type) {
	case NamespaceDeclarations[*TypeDeclaration]:
		return any(findDeclarationsWithSpec(declarationSpec[*TypeDeclaration]{node: node, query: typeDeclarations, parseFunc: parseTypeDeclaration})).(NamespaceDeclarations[T])
	case NamespaceDeclarations[*MethodDeclaration]:
		return any(findDeclarationsWithSpec(declarationSpec[*MethodDeclaration]{node: node, query: methodDeclarations, parseFunc: parseMethodDeclaration})).(NamespaceDeclarations[T])
	case NamespaceDeclarations[*FieldDeclaration]:
		return any(findDeclarationsWithSpec(declarationSpec[*FieldDeclaration]{node: node, query: fieldDeclarations, parseFunc: parseFieldDeclaration})).(NamespaceDeclarations[T])
	case NamespaceDeclarations[Declarable]:
		var allDeclarations NamespaceDeclarations[T]
		for name, declarations := range findDeclarationsWithSpec(declarationSpec[*TypeDeclaration]{node: node, query: typeDeclarations, parseFunc: parseTypeDeclaration}) {
			allDeclarations[name] = append(allDeclarations[name], any(declarations).([]T)...)
		}
		for name, declarations := range findDeclarationsWithSpec(declarationSpec[*MethodDeclaration]{node: node, query: methodDeclarations, parseFunc: parseMethodDeclaration}) {
			allDeclarations[name] = append(allDeclarations[name], any(declarations).([]T)...)
		}
		for name, declarations := range findDeclarationsWithSpec(declarationSpec[*FieldDeclaration]{node: node, query: fieldDeclarations, parseFunc: parseFieldDeclaration}) {
			allDeclarations[name] = append(allDeclarations[name], any(declarations).([]T)...)
		}
		return allDeclarations
	default:
		zap.L().With(logging.NodeField(node)).Panic("invalid typeName type cannot be parsed")
		return empty
	}
}

func findDeclarationsWithSpec[T Declarable](spec declarationSpec[T]) NamespaceDeclarations[T] {
	namespaceDeclarations := NamespaceDeclarations[T]{}
	nextMatch := DoQuery(spec.node, spec.query)
	for {
		match, found := nextMatch()
		if !found {
			break
		}
		parsedDeclaration := spec.parseFunc(match)
		namespace := parsedDeclaration.AsDeclaration().Namespace
		i := namespaceDeclarations[namespace]
		namespaceDeclarations[namespace] = append(i, parsedDeclaration)
	}

	return namespaceDeclarations
}

func FindDeclarationAtNode[T Declarable](node *sitter.Node) (T, bool) {
	var declaration T
	found := false
	switch any(declaration).(type) {
	case *TypeDeclaration:
		tDec, tFound := findDeclarationWithSpec(declarationSpec[*TypeDeclaration]{node: node, query: typeDeclarations, parseFunc: parseTypeDeclaration})
		declaration = any(tDec).(T)
		found = tFound
	case *MethodDeclaration:
		mDec, mFound := findDeclarationWithSpec(declarationSpec[*MethodDeclaration]{node: node, query: methodDeclarations, parseFunc: parseMethodDeclaration})
		declaration = any(mDec).(T)
		found = mFound
	case *FieldDeclaration:
		fDec, fFound := findDeclarationWithSpec(declarationSpec[*FieldDeclaration]{node: node, query: fieldDeclarations, parseFunc: parseFieldDeclaration})
		declaration = any(fDec).(T)
		found = fFound
	default:
		zap.L().With(logging.NodeField(node)).Panic("invalid typeName type cannot be parsed")
	}
	return declaration, found

}

func findDeclarationWithSpec[T Declarable](spec declarationSpec[T]) (T, bool) {
	nextMatch := DoQuery(spec.node, spec.query)
	if match, found := nextMatch(); found {
		return spec.parseFunc(match), true
	}
	var empty T
	return empty, false
}

func parseTypeDeclaration(match query.MatchNodes) *TypeDeclaration {
	classDeclaration := match["class_declaration"]
	interfaceDeclaration := match["interface_declaration"]
	recordDeclaration := match["record_declaration"]
	structDeclaration := match["struct_declaration"]
	name := match["name"]
	bases := match["bases"]
	attributes := match["attribute_list"]

	declaration := TypeDeclaration{
		Declaration: Declaration{
			Name:          name.Content(),
			AttributeList: attributes,
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

	modifiers := parseModifiers(declaration.Node)
	declaration.Visibility = modifiers.Visibility
	declaration.IsAbstract = modifiers.IsAbstract
	declaration.IsSealed = modifiers.IsSealed
	declaration.IsPartial = modifiers.IsPartial
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

	// handle static classes
	if declaration.IsStatic {
		declaration.IsSealed = true
	}

	return &declaration
}
func parseMethodDeclaration(match query.MatchNodes) *MethodDeclaration {
	methodDeclaration := match["method_declaration"]
	returnType := match["return_type"]
	name := match["name"]
	parameters := match["parameters"]

	declaration := MethodDeclaration{
		Declaration: Declaration{
			Name:          name.Content(),
			Node:          methodDeclaration,
			Kind:          DeclarationKindMethod,
			AttributeList: match["attribute_list"],
		},
		ReturnType: returnType.Content(),
	}

	declaration.Namespace = resolveNamespace(declaration.Node)

	modifiers := parseModifiers(declaration.Node)
	declaration.Visibility = modifiers.Visibility
	declaration.IsAbstract = modifiers.IsAbstract
	declaration.IsSealed = modifiers.IsSealed
	declaration.IsStatic = modifiers.IsStatic
	declaration.IsPartial = modifiers.IsPartial
	declaration.IsVirtual = modifiers.IsVirtual
	declaration.IsOverride = modifiers.IsOverride
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

	return &declaration
}

func declaringClass(declaration *sitter.Node, qualifiedName string) string {
	if outer := query.FirstAncestorOfType(declaration.Parent(), "class_declaration"); outer != nil {
		if !strings.Contains(qualifiedName, ".") {
			return ""
		}
		return qualifiedName[0:strings.LastIndex(qualifiedName, ".")]
	}
	return ""
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
				Type:     child.ChildByFieldName("type").Content(),
				TypeNode: child.ChildByFieldName("type"),
			})
		}
	}
	return parameters
}

func parseFieldDeclaration(match query.MatchNodes) *FieldDeclaration {
	fieldDeclaration := match["field_declaration"]
	fieldType := match["type"]
	name := match["name"]
	equalsValueClause := match["equals_value_clause"]

	declaration := &FieldDeclaration{
		Declaration: Declaration{
			Name:          name.Content(),
			Node:          fieldDeclaration,
			Kind:          DeclarationKindMethod,
			AttributeList: match["attribute_list"],
		},
		Type: fieldType.Content(),
	}

	declaration.Namespace = resolveNamespace(declaration.Node)

	modifiers := parseModifiers(declaration.Node)
	declaration.Visibility = modifiers.Visibility
	declaration.IsConst = modifiers.IsConst
	declaration.IsRequired = modifiers.IsRequired
	declaration.IsReadOnly = modifiers.IsReadOnly
	declaration.IsStatic = modifiers.IsStatic
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

type modifierSpec struct {
	Visibility Visibility
	IsSealed   bool
	IsAbstract bool
	IsStatic   bool
	IsConst    bool
	IsReadOnly bool
	IsRequired bool
	IsPartial  bool
	IsVirtual  bool
	IsOverride bool
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

		switch child.Content() {
		// C# Visibility: https://learn.microsoft.com/en-us/dotnet/csharp/programming-guide/classes-and-structs/access-modifiers
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
		// Non-visibility modifiers
		case "sealed":
			spec.IsSealed = true
		case "abstract":
			spec.IsAbstract = true
		case "partial":
			spec.IsPartial = true
		case "static":
			spec.IsStatic = true
		case "const":
			spec.IsConst = true
		case "readonly":
			spec.IsReadOnly = true
		case "required":
			spec.IsRequired = true
		case "virtual":
			spec.IsVirtual = true
		case "override":
			spec.IsOverride = true
		}
	}
	return spec
}
func parseBaseTypes(baseList *sitter.Node) map[string]*sitter.Node {
	if baseList == nil || baseList.Type() != "base_list" {
		return nil
	}

	bases := make(map[string]*sitter.Node)
	for i := 0; i < int(baseList.ChildCount()); i++ {
		child := baseList.Child(i)
		if t := child.Type(); t == "qualified_name" || t == "identifier" {
			bases[child.Content()] = child
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

func isNested(declaration *sitter.Node) bool {
	outer := query.FirstAncestorOfType(declaration.Parent(), "class_declaration")
	if outer == nil {
		return false
	}
	if declaration.Type() == "class_declaration" && outer != nil {
		return true
	}
	outer = query.FirstAncestorOfType(declaration.Parent(), "class_declaration")
	if outer != nil {
		return true
	}
	return false
}

func IsInNamespace[T Declarable](namespace string) predicate.Predicate[T] {
	return func(d T) bool {
		return namespace == d.AsDeclaration().Namespace
	}
}

func HasName[T Declarable](name string) predicate.Predicate[T] {
	return func(d T) bool {
		return name == d.AsDeclaration().Name
	}
}

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

func HasBaseWithSuffix(suffix string) predicate.Predicate[*TypeDeclaration] {
	return func(d *TypeDeclaration) bool {
		for name, _ := range d.Bases {
			if strings.HasSuffix(name, suffix) {
				return true
			}
		}
		return false
	}
}

func HasAttribute[T Declarable](attribute string) predicate.Predicate[T] {
	namespace, name := splitQualifiedName(attribute)
	return func(d T) bool {
		declaration := d.AsDeclaration()
		attrs := declaration.Attributes()
		// has qualified attribute
		if attr, ok := attrs[attribute]; ok {
			if IsValidTypeName(attr[0].Node, namespace, name) {
				return true
			}
		}
		// has namespace import + attribute
		if attr, ok := attrs[name]; ok {
			if IsValidTypeName(attr[0].Node, namespace, name) {
				return true
			}
		}
		return false
	}
}

func NameHasSuffix[T Declarable](suffix string) predicate.Predicate[T] {
	return func(d T) bool {
		return strings.HasSuffix(d.AsDeclaration().Name, suffix)
	}
}
