package csharp

import (
	"strings"
	"testing"

	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/stretchr/testify/assert"
)

func TestFindImportsInFile(t *testing.T) {

	tests := []struct {
		name            string
		program         string
		expectedImports Imports
	}{
		{
			name:    "compilation_unit::using",
			program: `using ns1;`,
			expectedImports: Imports{"ns1": []Import{
				{
					Name:  "ns1",
					Scope: ImportScopeCompilationUnit,
					Type:  ImportTypeUsing,
				},
			}},
		},
		{
			name:    "compilation_unit::using_static",
			program: `using static ns1;`,
			expectedImports: Imports{"ns1": []Import{
				{
					Name:  "ns1",
					Scope: ImportScopeCompilationUnit,
					Type:  ImportTypeUsingStatic,
				},
			}},
		},
		{
			name:    "compilation_unit::using_global",
			program: `global using ns1;`,
			expectedImports: Imports{"ns1": []Import{
				{
					Name:  "ns1",
					Scope: ImportScopeGlobal,
					Type:  ImportTypeUsing,
				},
			}},
		},
		{
			name: "compilation_unit::using_alias",
			program: `
			using ns1;
			using a1 = ns1
			`,
			expectedImports: Imports{"ns1": []Import{
				{
					Name:  "ns1",
					Scope: ImportScopeCompilationUnit,
					Type:  ImportTypeUsing,
				},
				{
					Name:  "ns1",
					Alias: "a1",
					Scope: ImportScopeCompilationUnit,
					Type:  ImportTypeUsingAlias,
				},
			}},
		},
		{
			name: "namespace::using",
			program: `
			namespace ns1 {
				using ns2;
			}
			`,
			expectedImports: Imports{"ns2": []Import{
				{
					Name:      "ns2",
					Scope:     ImportScopeNamespace,
					Type:      ImportTypeUsing,
					Namespace: "ns1",
				},
			}},
		},
		{
			name: "all::multiple imports",
			program: `
			global using ns1;
			using ns2;
			using static ns3;
			using a1 = ns2;
			namespace lns1 {
				using ns2;
			}
			namespace lns2 {
				using ns3;
			}
			`,
			expectedImports: Imports{
				"ns1": []Import{
					{
						Name:  "ns1",
						Scope: ImportScopeGlobal,
						Type:  ImportTypeUsing,
					},
				},
				"ns2": []Import{
					{
						Name:  "ns2",
						Scope: ImportScopeCompilationUnit,
						Type:  ImportTypeUsing,
					},
					{
						Name:  "ns2",
						Scope: ImportScopeCompilationUnit,
						Type:  ImportTypeUsingAlias,
						Alias: "a1",
					},
					{
						Name:      "ns2",
						Scope:     ImportScopeNamespace,
						Type:      ImportTypeUsing,
						Namespace: "lns1",
					},
				},
				"ns3": []Import{
					{
						Name:  "ns3",
						Scope: ImportScopeCompilationUnit,
						Type:  ImportTypeUsingStatic,
					},
					{
						Name:      "ns3",
						Scope:     ImportScopeNamespace,
						Type:      ImportTypeUsing,
						Namespace: "lns2",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		assert := assert.New(t)
		t.Run(tt.name, func(t *testing.T) {
			sourceFile, err := types.NewSourceFile("file.cs", strings.NewReader(tt.program), Language)
			if !assert.NoError(err) {
				return
			}

			verifyImports(assert, tt.expectedImports, FindImportsInFile(sourceFile))
		})
	}
}

func verifyImports(assert *assert.Assertions, expected Imports, actual Imports) {
	if !assert.Equal(len(expected), len(actual)) {
		return
	}
	for name, expectedImports := range expected {
		if actualImports, ok := actual[name]; assert.True(ok) {
			if !assert.Equal(len(expectedImports), len(actualImports)) {
				return
			}
			for i, expectedImport := range expectedImports {
				verifyImport(assert, expectedImport, actualImports[i])
			}
		}
	}
}

// Verifies that all import fields aside from Node are equal
func verifyImport(assert *assert.Assertions, expected Import, actual Import) {
	assert.Equal(expected.Name, actual.Name)
	assert.Equal(expected.Type, actual.Type)
	assert.Equal(expected.Alias, actual.Alias)
	assert.Equal(expected.Namespace, actual.Namespace)
	assert.Equal(expected.Scope, actual.Scope)
	assert.NotNil(actual.Node)
}
