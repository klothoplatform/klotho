package iac3

import (
	"context"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/stretchr/testify/assert"
)

func Test_parseArgs(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    map[string]Arg
	}{
		{
			name: "basic args",
			content: `
			interface Args {
				foo: string
				bar: number
			}
			`,
			want: map[string]Arg{
				"foo": {
					Name:    "foo",
					Type:    "string",
					Wrapper: "",
				},
				"bar": {
					Name:    "bar",
					Type:    "number",
					Wrapper: "",
				},
			},
		},
		{
			name: "with wrapper args",
			content: `
			interface Args {
				foo: WrappedClass<string>
			}
			`,
			want: map[string]Arg{
				"foo": {
					Name:    "foo",
					Type:    "WrappedClass<string>",
					Wrapper: "WrappedClass",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			parser := sitter.NewParser()
			parser.SetLanguage(templateTSLang.Sitter)
			tree, err := parser.ParseCtx(context.TODO(), nil, []byte(tt.content))
			if err != nil {
				t.Fatal(err)
			}

			got, err := parseArgs(tree.RootNode(), tt.name)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(got, tt.want)
		})
	}
}
