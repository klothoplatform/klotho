package javascript

import (
	"context"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/stretchr/testify/assert"
)

func TestSpecificExportQuery(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		queryName   string
		matchString string
	}{
		{
			name:        "wrong name",
			source:      "exports.b = 1;",
			queryName:   "a",
			matchString: "",
		},
		{
			name:        "no name",
			source:      "exports.b = 1;",
			queryName:   "",
			matchString: "1",
		},
		{
			name:        "match number",
			source:      "exports.a = 2;",
			queryName:   "a",
			matchString: "2",
		},
		{
			name:        "match string",
			source:      "exports.a = 'a';",
			queryName:   "a",
			matchString: "'a'",
		},
		{
			name:        "match func",
			source:      "exports.func = func;",
			queryName:   "func",
			matchString: "func",
		},
		{
			name:        "match object",
			source:      "exports.a = {b: 2};",
			queryName:   "a",
			matchString: "{b: 2}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			parser := sitter.NewParser()
			parser.SetLanguage(javascript.GetLanguage())
			tree, err := parser.ParseCtx(context.Background(), nil, []byte(tt.source))
			if !assert.NoError(err) {
				return
			}

			node := SpecificExportQuery(tree.RootNode(), tt.queryName)
			if tt.matchString != "" && assert.NotNil(node) {
				str := node.Content()
				assert.Equal(tt.matchString, str)
			} else {
				assert.Nil(node)
			}
		})
	}
}
