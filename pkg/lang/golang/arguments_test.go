package golang

import (
	"strings"
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/stretchr/testify/assert"
)

func Test_GetArguements(t *testing.T) {
	tests := []struct {
		name      string
		source    string
		want      []Argument
		wantFound bool
	}{
		{
			name: "finds next function Name and args",
			source: `
			x = s.my_func("val")
			y = s.other_func("something_else)
			`,
			want: []Argument{
				{Content: `"val"`, Type: "interpreted_string_literal"},
			},
			wantFound: true,
		},
		{
			name:      "args not required",
			source:    `v, err := s.someFunc()`,
			wantFound: false,
		},
		{
			name:   "a call containing other function calls as args",
			source: `v, err := runtimevar.OpenVariable(context.TODO(), fmt.Sprintf("file://%s?decoder=string", path))`,
			want: []Argument{
				{Content: "context.TODO()", Type: "call_expression"},
				{Content: `fmt.Sprintf("file://%s?decoder=string", path)`, Type: "call_expression"},
			},
			wantFound: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			f, err := core.NewSourceFile("", strings.NewReader(tt.source), Language)
			if !assert.NoError(err) {
				return
			}
			args, found := getArguements(f.Tree().RootNode())

			assert.ElementsMatch(tt.want, args)
			assert.Equal(tt.wantFound, found)
		})
	}
}
