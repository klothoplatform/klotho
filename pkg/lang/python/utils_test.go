package python

import (
	"strings"
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/stretchr/testify/assert"
)

func Test_stringLiteralContent(t *testing.T) {

	tests := []struct {
		name      string
		inputStr  string
		want      string
		wantError bool
	}{
		{
			name:     "strips single quote",
			inputStr: `'"input"'`,
			want:     `"input"`,
		},
		{
			name:     "strips double quote",
			inputStr: `"\"input\""`,
			want:     `\"input\"`,
		},
		{
			name:     "strips 3x single quote",
			inputStr: `'''input'''`,
			want:     "input",
		},
		{
			name:     "strips 3x double quote",
			inputStr: `"""input"""`,
			want:     "input",
		},
		{
			name:      "returns error on b-string",
			inputStr:  `b"input"`,
			wantError: true,
		},
		{
			name:      "returns error on f-string",
			inputStr:  `f"input"`,
			wantError: true,
		},
		{
			name:      "returns error on non-string input node",
			inputStr:  `x="input"`,
			wantError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			f, _ := core.NewSourceFile("", strings.NewReader(tt.inputStr), Language)
			got, err := stringLiteralContent(f.Tree().RootNode().Child(0).Child(0), f.Program())

			if tt.wantError {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.want, got)
		})
	}
}
