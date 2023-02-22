package csharp

import (
	"github.com/stretchr/testify/assert"
	"sort"
	"testing"
)

func TestFindTypeDeclarationsInFile(t *testing.T) {
	tests := []declarationTestCase{
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
	}
	runFindDeclarationsInFileTests(t, tests)
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

type testTypeDeclaration struct {
	testDeclaration
	Bases []string
}
