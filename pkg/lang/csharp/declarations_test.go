package csharp

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/stretchr/testify/assert"
	"sort"
	"strings"
	"testing"
)

func TestFindDeclarationsInFile(t *testing.T) {

	type testCase struct {
		name                 string
		program              string
		expectedDeclarations []testDeclarable
	}
	tests := []testCase{
		{
			name: "Parses type declarations",
			program: `
			public class c1 {}
			class gc1<T> {}
			static class stc1 {}
			private class pc1 : Base1, Base2 {}
			interface i1 {}
			struct s1 {}
			record r1 {}
			abstract class ac1 {}
			sealed class slc1 {}
			partial class pc1 {}
			protected internal class pic1 {}

			namespace ns1 {
				class c2 {
					class nc1 {}
				}
				namespace ns2 {
					class c3 {}
				}
			}
			`,
			expectedDeclarations: []testDeclarable{
				asTestDeclarable(&testTypeDeclaration{
					testDeclaration: testDeclaration{
						Name:          "c1",
						Kind:          DeclarationKindClass,
						Visibility:    VisibilityPublic,
						QualifiedName: "c1",
					},
				}),
				asTestDeclarable(&testTypeDeclaration{
					testDeclaration: testDeclaration{
						Name:          "gc1",
						Kind:          DeclarationKindClass,
						Visibility:    VisibilityInternal,
						QualifiedName: "gc1",
						IsGeneric:     true,
					},
				}),
				asTestDeclarable(&testTypeDeclaration{
					testDeclaration: testDeclaration{
						Name:          "stc1",
						Kind:          DeclarationKindClass,
						Visibility:    VisibilityInternal,
						QualifiedName: "stc1",
						HasModifiers:  []string{"static"},
						IsSealed:      true,
					},
				}),
				asTestDeclarable(&testTypeDeclaration{
					testDeclaration: testDeclaration{
						Name:          "pc1",
						Kind:          DeclarationKindClass,
						Visibility:    VisibilityPrivate,
						QualifiedName: "pc1",
					},
					Bases: []string{"Base1", "Base1"},
				}),
				asTestDeclarable(&testTypeDeclaration{
					testDeclaration: testDeclaration{
						Name:          "i1",
						Kind:          DeclarationKindInterface,
						Visibility:    VisibilityInternal,
						QualifiedName: "i1",
					},
				}),
				asTestDeclarable(&testTypeDeclaration{
					testDeclaration: testDeclaration{
						Name:          "s1",
						Kind:          DeclarationKindStruct,
						Visibility:    VisibilityInternal,
						QualifiedName: "s1",
					},
				}),
				asTestDeclarable(&testTypeDeclaration{
					testDeclaration: testDeclaration{
						Name:          "r1",
						Kind:          DeclarationKindRecord,
						Visibility:    VisibilityInternal,
						QualifiedName: "r1",
					},
				}),
				asTestDeclarable(&testTypeDeclaration{
					testDeclaration: testDeclaration{
						Name:          "ac1",
						Kind:          DeclarationKindClass,
						Visibility:    VisibilityInternal,
						QualifiedName: "ac1",
						HasModifiers:  []string{"abstract"},
					},
				}),
				asTestDeclarable(&testTypeDeclaration{
					testDeclaration: testDeclaration{
						Name:          "slc1",
						Kind:          DeclarationKindClass,
						Visibility:    VisibilityInternal,
						QualifiedName: "slc1",
						HasModifiers:  []string{"sealed"},
						IsSealed:      true,
					},
				}),
				asTestDeclarable(&testTypeDeclaration{
					testDeclaration: testDeclaration{
						Name:          "pc1",
						Kind:          DeclarationKindClass,
						Visibility:    VisibilityInternal,
						QualifiedName: "pc1",
						HasModifiers:  []string{"partial"},
					},
				}),
				asTestDeclarable(&testTypeDeclaration{
					testDeclaration: testDeclaration{
						Name:          "pic1",
						Kind:          DeclarationKindClass,
						Visibility:    VisibilityProtectedInternal,
						QualifiedName: "pic1",
					},
				}),
				asTestDeclarable(&testTypeDeclaration{
					testDeclaration: testDeclaration{
						Name:          "c2",
						Kind:          DeclarationKindClass,
						Visibility:    VisibilityInternal,
						QualifiedName: "ns1.c2",
						Namespace:     "ns1",
					},
				}),
				asTestDeclarable(&testTypeDeclaration{
					testDeclaration: testDeclaration{
						Name:           "nc1",
						Kind:           DeclarationKindClass,
						Visibility:     VisibilityPrivate,
						QualifiedName:  "ns1.c2.nc1",
						Namespace:      "ns1",
						DeclaringClass: "ns1.c2",
						IsNested:       true,
					},
				}),
				asTestDeclarable(&testTypeDeclaration{
					testDeclaration: testDeclaration{
						Name:          "c3",
						Kind:          DeclarationKindClass,
						Visibility:    VisibilityInternal,
						QualifiedName: "ns1.ns2.c3",
						Namespace:     "ns1.ns2",
					},
				}),
			},
		},
		{
			name: "Parses method declarations",
			program: `
			namespace ns1 {
				class c1 {
					class nc1 {
						void pm1() {} // nested class members are private by default
					}
					public virtual T m1<T>(T1 p1, T2 p2) {}
					abstract SomeType am1();
					static void stm1() {}
					sealed void slm1() {}
				}
			}
			`,
			expectedDeclarations: []testDeclarable{
				asTestDeclarable(&testMethodDeclaration{
					testDeclaration: testDeclaration{
						Name:           "pm1",
						Kind:           DeclarationKindMethod,
						Visibility:     VisibilityPrivate,
						QualifiedName:  "ns1.c1.nc1.pm1",
						Namespace:      "ns1",
						DeclaringClass: "ns1.c1.nc1",
						IsNested:       true,
					},
					ReturnType: "void",
				}),
				asTestDeclarable(&testMethodDeclaration{
					testDeclaration: testDeclaration{
						Name:           "m1",
						Kind:           DeclarationKindMethod,
						Visibility:     VisibilityPublic,
						QualifiedName:  "ns1.c1.m1",
						Namespace:      "ns1",
						DeclaringClass: "ns1.c1",
						HasModifiers:   []string{"virtual"},
						IsGeneric:      true,
					},
					ReturnType: "T",
					Parameters: []Parameter{
						{Type: "T1", Name: "p1"},
						{Type: "T2", Name: "p2"},
					},
				}),
				asTestDeclarable(&testMethodDeclaration{
					testDeclaration: testDeclaration{
						Name:           "am1",
						Kind:           DeclarationKindMethod,
						Visibility:     VisibilityInternal,
						QualifiedName:  "ns1.c1.am1",
						Namespace:      "ns1",
						DeclaringClass: "ns1.c1",
					},
					ReturnType: "SomeType",
				}),
				asTestDeclarable(&testMethodDeclaration{
					testDeclaration: testDeclaration{
						Name:           "stm1",
						Kind:           DeclarationKindMethod,
						Visibility:     VisibilityInternal,
						QualifiedName:  "ns1.c1.stm1",
						Namespace:      "ns1",
						DeclaringClass: "ns1.c1",
						HasModifiers:   []string{"static"},
						IsSealed:       true,
					},
					ReturnType: "void",
				}),
				asTestDeclarable(&testMethodDeclaration{
					testDeclaration: testDeclaration{
						Name:           "slm1",
						Kind:           DeclarationKindMethod,
						Visibility:     VisibilityInternal,
						QualifiedName:  "ns1.c1.slm1",
						Namespace:      "ns1",
						DeclaringClass: "ns1.c1",
						HasModifiers:   []string{"sealed"},
						IsSealed:       true,
					},
					ReturnType: "void",
				}),
			},
		},
		{
			name: "Parses field declarations",
			program: `
			namespace ns1 {
				class c1 {
					class nc1 {
						int nf1 = 0; // nested class members are private by default
					}
					int f1;
					Dictionary<T> f2 = new Dictionary<>();
					public static int f3 = 1;
					public int f4 = 1, f5 = 2;
					public event SomeDelegate e1;
				}
			}
			`,
			expectedDeclarations: []testDeclarable{
				asTestDeclarable(&testFieldDeclaration{
					testDeclaration: testDeclaration{
						Name:           "nf1",
						Kind:           DeclarationKindField,
						Visibility:     VisibilityPrivate,
						QualifiedName:  "ns1.c1.nc1.nf1",
						Namespace:      "ns1",
						DeclaringClass: "ns1.c1.nc1",
						IsNested:       true,
					},
					HasInitialValue: true,
					Type:            "int",
				}),
				asTestDeclarable(&testFieldDeclaration{
					testDeclaration: testDeclaration{
						Name:           "f1",
						Kind:           DeclarationKindField,
						Visibility:     VisibilityInternal,
						QualifiedName:  "ns1.c1.f1",
						Namespace:      "ns1",
						DeclaringClass: "ns1.c1",
					},
					Type: "int",
				}),
				asTestDeclarable(&testFieldDeclaration{
					testDeclaration: testDeclaration{
						Name:           "f2",
						Kind:           DeclarationKindField,
						Visibility:     VisibilityInternal,
						QualifiedName:  "ns1.c1.f2",
						Namespace:      "ns1",
						DeclaringClass: "ns1.c1",
						IsGeneric:      true,
					},
					HasInitialValue: true,
					Type:            "Dictionary<T>",
				}),
				asTestDeclarable(&testFieldDeclaration{
					testDeclaration: testDeclaration{
						Name:           "f3",
						Kind:           DeclarationKindField,
						Visibility:     VisibilityPublic,
						QualifiedName:  "ns1.c1.f3",
						Namespace:      "ns1",
						DeclaringClass: "ns1.c1",
						HasModifiers:   []string{"static"},
						IsSealed:       true, // not relevant on field declarations
					},
					HasInitialValue: true,
					Type:            "int",
				}),
				asTestDeclarable(&testFieldDeclaration{
					testDeclaration: testDeclaration{
						Name:           "f4",
						Kind:           DeclarationKindField,
						Visibility:     VisibilityPublic,
						QualifiedName:  "ns1.c1.f4",
						Namespace:      "ns1",
						DeclaringClass: "ns1.c1",
					},
					HasInitialValue: true,
					Type:            "int",
				}),
				asTestDeclarable(&testFieldDeclaration{
					testDeclaration: testDeclaration{
						Name:           "f5",
						Kind:           DeclarationKindField,
						Visibility:     VisibilityPublic,
						QualifiedName:  "ns1.c1.f5",
						Namespace:      "ns1",
						DeclaringClass: "ns1.c1",
					},
					HasInitialValue: true,
					Type:            "int",
				}),
				asTestDeclarable(&testFieldDeclaration{
					testDeclaration: testDeclaration{
						Name:           "e1",
						Kind:           DeclarationKindEvent,
						Visibility:     VisibilityPublic,
						QualifiedName:  "ns1.c1.e1",
						Namespace:      "ns1",
						DeclaringClass: "ns1.c1",
					},
					Type: "SomeDelegate",
				}),
			},
		},
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

func validateFieldDeclaration(assert *assert.Assertions, expected *testFieldDeclaration, actual *FieldDeclaration) {
	validateDeclaration(assert, expected.testDeclaration, actual.Declaration)
	assert.Equalf(expected.HasInitialValue, actual.HasInitialValue, "HasInitialValue does not match for declaration %s", expected.QualifiedName)
}

func validateMethodDeclaration(assert *assert.Assertions, expected *testMethodDeclaration, actual *MethodDeclaration) {
	validateDeclaration(assert, expected.testDeclaration, actual.Declaration)
	assert.Equalf(expected.ReturnType, actual.ReturnType, "ReturnType does not match for declaration %s", expected.QualifiedName)
	assert.Equalf(len(expected.Parameters), len(actual.Parameters), "len(Parameters) does not match for declaration %s", expected.QualifiedName)
	for i, ep := range expected.Parameters {
		assert.Equalf(ep.Name, actual.Parameters[i].Name, "Parameters[%d].Name does not match for declaration %s", i, expected.QualifiedName)
		assert.Equalf(ep.Type, actual.Parameters[i].Type, "Parameters[%d].Type does not match for declaration %s", i, expected.QualifiedName)
		assert.NotNilf(actual.Parameters[i].TypeNode, "TypeNode not set for declaration %s", expected.QualifiedName)
	}
}

func validateTypeDeclaration(assert *assert.Assertions, expected *testTypeDeclaration, actual *TypeDeclaration) {
	validateDeclaration(assert, expected.testDeclaration, actual.Declaration)
	sort.Strings(expected.Bases)
	var aBases []string
	for b := range actual.Bases {
		aBases = append(aBases, b)
	}
	assert.Equalf(expected.Bases, aBases, "Bases do not match for declaration %s", expected.QualifiedName)
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

	testTypeDeclaration struct {
		testDeclaration
		Bases []string
	}

	testMethodDeclaration struct {
		testDeclaration
		Parameters []Parameter
		ReturnType string
	}

	testFieldDeclaration struct {
		testDeclaration
		HasInitialValue bool
		Type            string
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
