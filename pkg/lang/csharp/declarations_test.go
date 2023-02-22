package csharp

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

type declarationTestCase struct {
	name                 string
	program              string
	expectedDeclarations []testDeclarable
}

func TestFindDeclarationAttributes(t *testing.T) {
	tests := []declarationTestCase{
		{
			name: "Parses attributes",
			program: `
			[Route("/path"), AcceptVerbs("GET", "POST")]
			[AcceptVerbs("PUT", Route="/other")]
			class c1 {}
			`,
			expectedDeclarations: []testDeclarable{
				asTestDeclarable(&testTypeDeclaration{
					testDeclaration: testDeclaration{
						Name:          "c1",
						Kind:          DeclarationKindClass,
						Visibility:    VisibilityInternal,
						QualifiedName: "c1",
						Attributes: map[string][]testAttribute{
							"Route": {{Name: "Route", Args: []testArg{{Value: "/path"}}}},
							"AcceptVerbs": {
								{Name: "AcceptVerbs", Args: []testArg{
									{Value: "GET"},
									{Value: "POST"},
								}},
								{Name: "AcceptVerbs", Args: []testArg{
									{Value: "PUT"},
									{Name: "Route", Value: "/other"},
								}},
							},
						},
					},
				}),
			},
		},
	}
	runFindDeclarationsInFileTests(t, tests)
}

func runFindDeclarationsInFileTests(t *testing.T, tests []declarationTestCase) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			sourceFile, err := core.NewSourceFile("file.cs", strings.NewReader(tt.program), Language)
			if !assert.NoError(err) {
				return
			}

			declarations := FindDeclarationsInFile[Declarable](sourceFile)

			expected := make(map[string]testDeclarable)
			for _, d := range tt.expectedDeclarations {
				expected[d.AsTestDeclaration().QualifiedName] = d
			}
			actual := make(map[string]Declarable)
			for _, d := range declarations.Declarations() {
				actual[d.AsDeclaration().QualifiedName] = d
			}

			for qn, ed := range expected {
				ad, ok := actual[qn]
				if assert.Truef(ok, "actual declaration not found for declaration %s", ed.AsTestDeclaration().QualifiedName) {
					validateDeclarable(assert, ed, ad)
				}
			}
		})
	}
}

func validateDeclarable(assert *assert.Assertions, expected testDeclarable, actual Declarable) {
	if etd, ok := expected.(*testTypeDeclaration); ok && assert.IsType(&TypeDeclaration{}, actual) {
		validateTypeDeclaration(assert, etd, actual.(*TypeDeclaration))
	} else if emd, ok := expected.(*testMethodDeclaration); ok && assert.IsType(&MethodDeclaration{}, actual) {
		validateMethodDeclaration(assert, emd, actual.(*MethodDeclaration))
	} else if efd, ok := expected.(*testFieldDeclaration); ok && assert.IsType(&FieldDeclaration{}, actual) {
		validateFieldDeclaration(assert, efd, actual.(*FieldDeclaration))
	}
}

func validateDeclaration(assert *assert.Assertions, expected testDeclaration, actual Declaration) {
	assert.Equalf(expected.Name, actual.Name, "Name does not match for declaration %s", expected.QualifiedName)
	assert.Equalf(expected.QualifiedName, actual.QualifiedName, "QualifiedName does not match for declaration %s", expected.QualifiedName)
	assert.Equalf(expected.Namespace, actual.Namespace, "Namespace does not match for declaration %s", expected.QualifiedName)
	assert.Equalf(expected.Kind, actual.Kind, "Kind does not match for declaration %s", expected.QualifiedName)
	assert.Equalf(expected.IsNested, actual.IsNested, "IsNested does not match for declaration %s", expected.QualifiedName)
	assert.Equalf(expected.IsGeneric, actual.IsGeneric, "IsGeneric does not match for declaration %s", expected.QualifiedName)
	assert.Equalf(expected.IsSealed, actual.IsSealed(), "IsSealed() does not match for declaration %s", expected.QualifiedName)
	assert.Equalf(expected.Visibility, actual.Visibility, "Visibility does not match for declaration %s", expected.QualifiedName)
	assert.Equalf(expected.DeclaringClass, actual.DeclaringClass, "DeclaringClass does not match for declaration %s", expected.QualifiedName)
	if len(expected.HasModifiers) > 0 {
		assert.Truef(actual.HasModifiers(expected.HasModifiers[0], expected.HasModifiers...), "HasModifiers does not match for declaration %s", expected.QualifiedName)
	}

	attrs := actual.Attributes()
	for attrType, eattrs := range expected.Attributes {
		aattrs := attrs[attrType]
		if assert.Equalf(len(eattrs), len(aattrs), "len(Attributes) does not match for declaration %s", expected.QualifiedName) {
			for i, eattr := range eattrs {
				aattr := aattrs[i]
				assert.Equalf(eattr.Name, aattr.Name, "Attribute.Name does not match for declaration %s", expected.QualifiedName)
				aargs := aattr.Args()
				if assert.Equalf(len(eattr.Args), len(aattr.Args()), "len(Attribute.Args) does not match for declaration %s", expected.QualifiedName) {
					for j, earg := range eattr.Args {
						aarg := aargs[j]
						assert.Equalf(earg.Name, aarg.Name, "Attribute.Arg.Name does not match for declaration %s", expected.QualifiedName)
						assert.Equalf(earg.Value, aarg.Value, "Attribute.Arg.Value does not match for declaration %s", expected.QualifiedName)
					}
				}
			}
		}
	}
}

type (
	testDeclaration struct {
		HasModifiers   []string
		Namespace      string
		Name           string
		QualifiedName  string
		DeclaringFile  string
		Kind           DeclarationKind
		Visibility     Visibility
		IsGeneric      bool
		IsNested       bool
		DeclaringClass string
		IsSealed       bool
		Attributes     map[string][]testAttribute
	}

	testAttribute struct {
		Name string
		Args []testArg
	}

	testArg struct {
		Name  string
		Value string
	}

	testDeclarable interface {
		AsTestDeclaration() testDeclaration
	}
)

func (d testDeclaration) AsTestDeclaration() testDeclaration {
	return d
}

func asTestDeclarable[T testDeclarable](declarable T) testDeclarable {
	var td testDeclarable
	td = declarable
	return td
}
