package csharp

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func validateFieldDeclaration(assert *assert.Assertions, expected *testFieldDeclaration, actual *FieldDeclaration) {
	validateDeclaration(assert, expected.testDeclaration, actual.Declaration)
	assert.Equalf(expected.HasInitialValue, actual.HasInitialValue, "HasInitialValue does not match for declaration %s", expected.QualifiedName)
}

func TestFindFieldDeclarationsInFile(t *testing.T) {
	tests := []declarationTestCase[*FieldDeclaration]{
		{
			name: "parses attributes on field declarations",
			program: `
			
			class c1 {
				[Attr1]
				int f1;
			}
			`,
			expectedDeclarations: []testDeclarable{
				testFieldDeclaration{
					testDeclaration: testDeclaration{
						Name:          "f1",
						Kind:          DeclarationKindField,
						Visibility:    VisibilityInternal,
						QualifiedName: "c1.f1",
						Attributes: map[string][]testAttribute{
							"Attr1": {{Name: "Attr1", Args: []testArg{{Name: "Arg", Value: "value"}}}},
						},
					},
				},
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
				testFieldDeclaration{
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
				},
				testFieldDeclaration{
					testDeclaration: testDeclaration{
						Name:           "f1",
						Kind:           DeclarationKindField,
						Visibility:     VisibilityInternal,
						QualifiedName:  "ns1.c1.f1",
						Namespace:      "ns1",
						DeclaringClass: "ns1.c1",
					},
					Type: "int",
				},
				testFieldDeclaration{
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
				},
				testFieldDeclaration{
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
				},
				testFieldDeclaration{
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
				},
				testFieldDeclaration{
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
				},
				testFieldDeclaration{
					testDeclaration: testDeclaration{
						Name:           "e1",
						Kind:           DeclarationKindEvent,
						Visibility:     VisibilityPublic,
						QualifiedName:  "ns1.c1.e1",
						Namespace:      "ns1",
						DeclaringClass: "ns1.c1",
					},
					Type: "SomeDelegate",
				},
			},
		},
	}
	runFindDeclarationsInFileTests(t, tests)
}

type testFieldDeclaration struct {
	testDeclaration
	HasInitialValue bool
	Type            string
}
