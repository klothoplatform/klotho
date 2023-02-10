package csharp

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
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
			sourceFile, err := core.NewSourceFile("file.cs", strings.NewReader(tt.program), Language)
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

func TestImports_IsValidTypeName(t *testing.T) {
	tests := []struct {
		name           string
		typeQuery      string
		namespace      string
		typeName       string
		program        string
		expectedOutput bool
	}{
		{
			name:           "uses qualified type name",
			program:        `class C1 : Fully.Qualified.Name {}`,
			typeQuery:      "(base_list .(_) @type)",
			namespace:      "Fully.Qualified",
			typeName:       "Name",
			expectedOutput: true,
		},
		{
			name: "uses class declared in the same file + namespace",
			program: `
			namespace ns1 {
				class C1 : C2 {} 
				class C2 {} 
			}`,
			typeQuery:      "(base_list .(_) @type)",
			namespace:      "ns1",
			typeName:       "C2",
			expectedOutput: true,
		},
		{
			name: "uses class declared in the same file + different namespace",
			program: `
			namespace ns1 {
				class C1 : C2 {} 
			}
			namespace ns2 {
				class C2 {} 
			}`,
			typeQuery:      "(base_list .(_) @type)",
			namespace:      "ns1",
			typeName:       "C2",
			expectedOutput: false,
		},
		{
			name: "uses type imported into a different namespace in the same file",
			program: `
			namespace ns1 {
				class C1 : ClassFromOtherNamespace {} 
			}
			namespace ns2 {
				using Other.Namespace;
			}`,
			typeQuery:      "(base_list .(_) @type)",
			namespace:      "Other.Namespace",
			typeName:       "ClassFromOtherNamespace",
			expectedOutput: false,
		},
		{
			name: "uses aliased type imported into a different namespace in the same file",
			program: `
			namespace ns1 {
				class C1 : AliasedType {} 
			}
			namespace ns2 {
				using AliasedType = Other.Namespace.ClassFromOtherNamespace;
			}`,
			typeQuery:      "(base_list .(_) @type)",
			namespace:      "Other.Namespace",
			typeName:       "ClassFromOtherNamespace",
			expectedOutput: false,
		},
		{
			name: "uses type name from imported namespace",
			program: `
			using Other.Namespace;
			class C1 : ImportedName {}
			`,
			typeQuery:      "(base_list .(_) @type)",
			namespace:      "Other.Namespace",
			typeName:       "ImportedName",
			expectedOutput: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			file, err := core.NewSourceFile("program.cs", strings.NewReader(tt.program), Language)
			if !assert.NoError(err) {
				return
			}
			match, found := DoQuery(file.Tree().RootNode(), tt.typeQuery)()
			if !assert.True(found) {
				return
			}
			typeNode := match["type"]
			if !assert.NotNil(typeNode) {
				return
			}
			assert.Equalf(tt.expectedOutput, IsValidTypeName(typeNode, tt.namespace, tt.typeName), "IsValidTypeName(%v, %v, %v)", typeNode.Content(), tt.namespace, tt.typeName)
		})
	}
}
