package csharp

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFindMethodDeclarationsInFile(t *testing.T) {
	tests := []declarationTestCase[*MethodDeclaration]{
		{
			name: "parses attributes on types",
			program: `
			class c1 {
				[Route("/path"), AcceptVerbs("GET", "POST")]
				[AcceptVerbs("PUT", Route="/other")]
				void m1(){}
			}
			`,
			expectedDeclarations: []testDeclarable{
				testMethodDeclaration{
					testDeclaration: testDeclaration{
						Name:          "m1",
						Kind:          DeclarationKindMethod,
						Visibility:    VisibilityInternal,
						QualifiedName: "c1.m1",
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
				},
			},
		},
		{
			name: "parses method declarations",
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
				testMethodDeclaration{
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
				},
				testMethodDeclaration{
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
					Parameters: []testParameter{
						{Type: "T1", Name: "p1"},
						{Type: "T2", Name: "p2"},
					},
				},
				testMethodDeclaration{
					testDeclaration: testDeclaration{
						Name:           "am1",
						Kind:           DeclarationKindMethod,
						Visibility:     VisibilityInternal,
						QualifiedName:  "ns1.c1.am1",
						Namespace:      "ns1",
						DeclaringClass: "ns1.c1",
					},
					ReturnType: "SomeType",
				},
				testMethodDeclaration{
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
				},
				testMethodDeclaration{
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
				},
			},
		},
	}
	runFindDeclarationsInFileTests(t, tests)
}

func validateMethodDeclaration(assert *assert.Assertions, expected *testMethodDeclaration, actual *MethodDeclaration) {
	validateDeclaration(assert, expected.testDeclaration, actual.Declaration)
	assert.Equalf(expected.ReturnType, actual.ReturnType, "ReturnType does not match for declaration %s", expected.QualifiedName)
	assert.Equalf(len(expected.Parameters), len(actual.Parameters), "len(Parameters) does not match for declaration %s", expected.QualifiedName)
	for i, ep := range expected.Parameters {
		assert.Equalf(ep.Name, actual.Parameters[i].Name, "Parameters[%d].Name does not match for declaration %s", i, expected.QualifiedName)
		assert.Equalf(ep.Type, actual.Parameters[i].TypeNode.Content(), "Parameters[%d].Type does not match for declaration %s", i, expected.QualifiedName)
		assert.NotNilf(actual.Parameters[i].TypeNode, "TypeNode not set for declaration %s", expected.QualifiedName)
	}
}

type (
	testMethodDeclaration struct {
		testDeclaration
		Parameters []testParameter
		ReturnType string
	}

	testParameter struct {
		Type string
		Name string
	}
)
