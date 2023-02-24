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

type (
	// Declaration is the base struct for all C# declarations.
	//
	// Specific declaration kinds can embed Declaration in kind-specific structs
	// that match their more specific needs.
	Declaration struct {
		Node           *sitter.Node
		Name           string
		Namespace      string
		DeclaringFile  string
		Kind           DeclarationKind
		Visibility     Visibility
		QualifiedName  string
		IsGeneric      bool
		IsNested       bool
		DeclaringClass string
	}
	Attribute struct {
		Name string
		Node *sitter.Node
	}

	AttributeArg struct {
		Name  string
		Value string
	}

	// Attributes is a mapping between an attribute name and its declarations in the current context.
	Attributes map[string][]Attribute

	// Declarable simplifies the process of working with various typeName kinds simultaneously.
	Declarable interface {
		// AsDeclaration returns the Declarable's underlying Declaration.
		AsDeclaration() Declaration
		// SetDeclaringFile sets the value of the underlying Declaration's DeclaringFile field.
		SetDeclaringFile(string)
	}

	// NamespaceDeclarations represents the relationship between a namespace and its declarations in the current context.
	NamespaceDeclarations[T Declarable] map[string][]T

	// declarationSpec is used to wire up a declaration type for use higher order generic parsing functions.
	declarationSpec[T Declarable] struct {
		// node is the node to begin the query at.
		node *sitter.Node
		// query is the tree-sitter query to execute to find the declarations of type T.
		query string
		// parseFunc is the function executed to parse the results of a declarationSpec's query.
		parseFunc func(match query.MatchNodes) T
	}
)

// DeclarationKind is the kind of declaration represented by a given Declaration.
type DeclarationKind string

const (
	// Implemented
	DeclarationKindClass     = DeclarationKind("class")
	DeclarationKindInterface = DeclarationKind("interface")
	DeclarationKindRecord    = DeclarationKind("record")
	DeclarationKindStruct    = DeclarationKind("struct")
	DeclarationKindField     = DeclarationKind("field")
	DeclarationKindMethod    = DeclarationKind("method")

	// TODO: implement parsing for remaining unimplemented declaration kinds
	// Unimplemented
	DeclarationKindProperty      = DeclarationKind("property")
	DeclarationKindEnum          = DeclarationKind("enum")
	DeclarationKindIndexer       = DeclarationKind("indexer")
	DeclarationKindDelegate      = DeclarationKind("delegate")
	DeclarationKindLocalVariable = DeclarationKind("local_variable")
	DeclarationKindLocalFunction = DeclarationKind("local_function")
	DeclarationKindEvent         = DeclarationKind("event")
)

// Visibility represents C#'s declaration visibility matrix
type Visibility string

const (
	VisibilityPublic            = Visibility("public")
	VisibilityProtected         = Visibility("protected")
	VisibilityInternal          = Visibility("internal")
	VisibilityPrivate           = Visibility("private")
	VisibilityProtectedInternal = Visibility("protected_internal")
	VisibilityPrivateProtected  = Visibility("private_protected")
)

// AsDeclaration returns the Declaration itself to comply with the Declarable interface.
func (d *Declaration) AsDeclaration() Declaration {
	return *d
}

// SetDeclaringFile sets the Declaration.DeclaringFile file field to the supplied filepath
// to comply with the Declarable interface.
func (d *Declaration) SetDeclaringFile(filepath string) {
	d.DeclaringFile = filepath
}

// Attributes gets the attributes a declaration is annotated with.
func (d *Declaration) Attributes() Attributes {
	attributes := make(Attributes)

	for i := 0; i < int(d.Node.NamedChildCount()); i++ {
		n := d.Node.NamedChild(i)
		if n.Type() != "attribute_list" {
			continue
		}

		for j := 0; j < int(n.NamedChildCount()); j++ {
			attr := n.NamedChild(j)
			if attr.Type() != "attribute" {
				continue
			}
			name := attr.ChildByFieldName("name").Content()
			attributes[name] = append(attributes[name], Attribute{
				Name: name,
				Node: attr,
			})
		}
	}
	return attributes
}

// IsSealed Evaluates if a declaration is functionally sealed (i.e. has either the "static" or "sealed" modifier).
func (d *Declaration) IsSealed() bool {
	return d.HasAnyModifier("static", "sealed")
}

// HasAnyModifier evaluates if a declaration is modified with any of the supplied modifiers.
func (d *Declaration) HasAnyModifier(modifier string, additional ...string) bool {
	modifiers := append(additional, modifier)
	for i := 0; i < int(d.Node.NamedChildCount()); i++ {
		c := d.Node.NamedChild(i)
		if c.Type() != "modifier" {
			continue
		}

		for _, m := range modifiers {
			if m == c.Content() {
				return true
			}
		}
	}
	return false
}

// HasModifiers evaluates if a declaration is modified with all the supplied modifiers.
func (d *Declaration) HasModifiers(modifier string, additional ...string) bool {
	modifiers := append(additional, modifier)
	matches := make(map[string]bool)
	for _, m := range modifiers {
		matches[m] = false
	}
	for i := 0; i < int(d.Node.NamedChildCount()); i++ {
		c := d.Node.NamedChild(i)
		if c.Type() != "modifier" {
			continue
		}

		m := c.Content()
		if _, ok := matches[m]; ok {
			matches[m] = true
		}
	}
	for _, found := range matches {
		if !found {
			return false
		}
	}
	return true
}

// Args returns a slice containing an Attribute's arguments.
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
			value = normalizedStringContent(valueN)
		}

		args = append(args, AttributeArg{
			Name:  query.NodeContentOrEmpty(nameN),
			Value: value,
		})
	}
	return args
}

// OfType returns a []Attribute filtered by the supplied attribute types.
// The supplied attribute types should be qualified names.
func (a Attributes) OfType(attrType string, additional ...string) []Attribute {
	types := append(additional, attrType)
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

// Declarations converts NamespaceDeclarations[T] to []T to simplify iteration.
func (nsd NamespaceDeclarations[T]) Declarations() []T {
	var allDeclarations []T
	for _, nsDeclarations := range nsd {
		allDeclarations = append(allDeclarations, nsDeclarations...)
	}
	return allDeclarations
}

// FindDeclarationsInFile returns a map containing a list of declarations for each namespace in the supplied file.
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

// FindDeclarationsAtNode returns a map containing a list of declarations of the supplied generic type "T"
// for each namespace starting from the supplied node.
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
		allDeclarations := make(NamespaceDeclarations[Declarable])
		for namespace, declarations := range findDeclarationsWithSpec(declarationSpec[*TypeDeclaration]{node: node, query: typeDeclarations, parseFunc: parseTypeDeclaration}) {
			allDeclarations[namespace] = append(allDeclarations[namespace], toDeclarableSlice(declarations)...)
		}
		for name, declarations := range findDeclarationsWithSpec(declarationSpec[*MethodDeclaration]{node: node, query: methodDeclarations, parseFunc: parseMethodDeclaration}) {
			allDeclarations[name] = append(allDeclarations[name], toDeclarableSlice(declarations)...)
		}
		for name, declarations := range findDeclarationsWithSpec(declarationSpec[*FieldDeclaration]{node: node, query: fieldDeclarations, parseFunc: parseFieldDeclaration}) {
			allDeclarations[name] = append(allDeclarations[name], toDeclarableSlice(declarations)...)
		}
		return any(allDeclarations).(NamespaceDeclarations[T])
	default:
		zap.L().With(logging.NodeField(node)).Panic("invalid typeName type cannot be parsed")
		return empty
	}
}

func toDeclarableSlice[T Declarable](s []T) []Declarable {
	var ds []Declarable
	for _, d := range s {
		ds = append(ds, d)
	}
	return ds
}

// FindDeclarationAtNode finds the next declaration of the supplied Declarable implementation starting from the supplied node.
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

// findDeclarationsWithSpec finds any declarations matching the supplied declarationSpec.
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

// findDeclarationWithSpec finds the first declaration matching the supplied declarationSpec.
func findDeclarationWithSpec[T Declarable](spec declarationSpec[T]) (T, bool) {
	nextMatch := DoQuery(spec.node, spec.query)
	if match, found := nextMatch(); found {
		return spec.parseFunc(match), true
	}
	var empty T
	return empty, false
}

// declaringClass returns the name of the declaration's parent class
// if the supplied declaration is a class member or nested class.
func declaringClass(declaration *sitter.Node, qualifiedName string) string {
	if outer := query.FirstAncestorOfType(declaration.Parent(), "class_declaration"); outer != nil {
		if !strings.Contains(qualifiedName, ".") {
			return ""
		}
		return qualifiedName[0:strings.LastIndex(qualifiedName, ".")]
	}
	return ""
}

// parseVisibilityModifiers returns the supplied declaration node's modifier-based visibility designation
// Note: default visibility (when no modifiers are present) must be handled by the caller.
func parseVisibilityModifiers(declaration *sitter.Node) Visibility {
	var vis Visibility
	if declaration == nil {
		return vis
	}

	for i := 0; i < int(declaration.NamedChildCount()); i++ {
		child := declaration.NamedChild(i)
		if child.Type() != "modifier" {
			continue
		}

		switch child.Content() {
		// C# Visibility: https://learn.microsoft.com/en-us/dotnet/csharp/programming-guide/classes-and-structs/access-modifiers
		case "private":
			if vis == VisibilityProtected {
				vis = VisibilityPrivateProtected
			} else {
				vis = VisibilityPrivate
			}
		case "protected":
			switch vis {
			case VisibilityPrivate:
				vis = VisibilityPrivateProtected
			case VisibilityInternal:
				vis = VisibilityProtectedInternal
			default:
				vis = VisibilityProtected
			}
		case "internal":
			if vis == VisibilityProtected {
				vis = VisibilityProtectedInternal
			} else {
				vis = VisibilityInternal
			}
		case "public":
			vis = VisibilityPublic
		}
	}
	return vis
}

// resolveNamespace returns the namespace that the supplied declaration was declared in.
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

// resolveQualifiedName returns the namespace-qualified name of the supplied declaration node.
// e.g. Class "MyClass" in namespace "My.Namespace" -> My.Namespace.MyClass
func resolveQualifiedName(declaration *sitter.Node) string {
	if declaration == nil {
		return ""
	}

	nameNode := declaration.ChildByFieldName("name")
	// handle fields which use identifiers instead of names
	if nameNode == nil {
		nameNode = query.FirstChildOfType(declaration, "identifier")
	}

	if nameNode == nil {
		return ""
	}

	components := []string{nameNode.Content()}

	for node := declaration.Parent(); node != nil; node = node.Parent() {
		if name := node.ChildByFieldName("name"); name != nil {
			components = append([]string{name.Content()}, components...)
		}
	}
	return strings.Join(components, ".")
}

// isNested evaluates if the supplied declaration node is a nested class or nested class member.
func isNested(declaration *sitter.Node) bool {
	outer := query.FirstAncestorOfType(declaration.Parent(), "class_declaration")
	if outer == nil {
		return false
	}
	if declaration.Type() == "class_declaration" {
		return true
	}
	outer = query.FirstAncestorOfType(outer.Parent(), "class_declaration")
	return outer != nil
}

// IsInNamespace is a predicate that evaluates if a Declarable is declared in the supplied namespace.
func IsInNamespace[T Declarable](namespace string) predicate.Predicate[T] {
	return func(d T) bool {
		return namespace == d.AsDeclaration().Namespace
	}
}

// HasName is a predicate that evaluates if a Declarable has the supplied name.
func HasName[T Declarable](name string) predicate.Predicate[T] {
	return func(d T) bool {
		return name == d.AsDeclaration().Name
	}
}

// HasAttribute is a predicate that evaluates if a Declarable is annotated with the given attribute.
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

// NameHasSuffix is a predicate that evaluates if a Declarable's name matches the supplied suffix.
func NameHasSuffix[T Declarable](suffix string) predicate.Predicate[T] {
	return func(d T) bool {
		return strings.HasSuffix(d.AsDeclaration().Name, suffix)
	}
}
