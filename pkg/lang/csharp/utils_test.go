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
