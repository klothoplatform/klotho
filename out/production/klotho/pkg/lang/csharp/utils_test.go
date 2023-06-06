package csharp

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func Test_IsValidTypeName(t *testing.T) {
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
		{
			name: "works for method argument types",
			program: `
			using Other.Namespace;
			class C1 { void M1(ImportedName p1){} }
			`,
			typeQuery:      "(parameter type: (_) @type)",
			namespace:      "Other.Namespace",
			typeName:       "ImportedName",
			expectedOutput: true,
		},
		{
			name: "works for field types",
			program: `
			using Other.Namespace;
			class C1 { ImportedName F1 = new ImportedName(); }
			`,
			typeQuery:      "(variable_declaration type: (_) @type)",
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

func Test_normalizedStringContent(t *testing.T) {
	tests := []struct {
		name       string
		stringNode string
		want       string
	}{
		{
			name:       "string literal: content is extracted",
			stringNode: `"H\"ello\\"`,
			want:       `H\"ello\\`,
		},
		{
			name:       `verbatim string literal: 2 double quotes are converted to 1 escaped double quote ("" -> \")`,
			stringNode: `@"Some ""quoted"" text"`,
			want:       `Some \"quoted\" text`,
		},
		{
			name:       `verbatim string literal: 1 back slash converted to 2 backslashes (\ -> \\)`,
			stringNode: `@"Some \text\"`,
			want:       `Some \\text\\`,
		},
		{
			name:       `raw string literal: double quotes are escaped (" -> \")`,
			stringNode: `"""Some "quoted" text"""`,
			want:       "", // TODO: replace this want with the commented out alternative once raw string literals are supported (tree-sitter-c-sharp >= 0.21.0)
			//want:       `Some \"quoted\" text`,
		},
		{
			name:       `raw string literal: 1 back slash converted to 2 backslashes (\ -> \\)`,
			stringNode: `"""Some \text\"""`,
			want:       "", // TODO: replace this want with the commented out alternative once raw string literals are supported (tree-sitter-c-sharp >= 0.21.0)
			//want:       `Some \\text\\`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			f, _ := core.NewSourceFile("file.cs", strings.NewReader(tt.stringNode), Language)
			assert.Equalf(tt.want, normalizedStringContent(f.Tree().RootNode().Child(0)), "normalizedStringContent(%v)", tt.stringNode)
		})
	}
}
