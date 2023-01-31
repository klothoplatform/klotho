package python

import (
	"strings"
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/stretchr/testify/assert"
)

func Test_getNextCallDetails(t *testing.T) {
	tests := []struct {
		name            string
		source          string
		wantCallDetails FunctionCallDetails
		wantFound       bool
	}{
		{
			name: "finds next function Name and args",
			source: `
			x = my_func("val", key1=val1, key2=2)
			y = other_func()
			`,
			wantCallDetails: FunctionCallDetails{
				Name: "my_func",
				Arguments: []FunctionArg{
					{
						Name:  "",
						Value: `"val"`,
					}, {
						Name:  "key1",
						Value: `val1`,
					}, {
						Name:  "key2",
						Value: `2`,
					},
				},
			},
			wantFound: true,
		},
		{
			name:   "args not required",
			source: `x = my_func()`,
			wantCallDetails: FunctionCallDetails{
				Name:      "my_func",
				Arguments: []FunctionArg{},
			},
			wantFound: true,
		},
		{
			name:   "finds function call on attribute",
			source: `x = y.my_func()`,
			wantCallDetails: FunctionCallDetails{
				Name:      "y.my_func",
				Arguments: []FunctionArg{},
			},
			wantFound: true,
		},
		{
			name:            "found is false when no next call is found",
			source:          `import module`,
			wantCallDetails: FunctionCallDetails{Arguments: []FunctionArg{}},
			wantFound:       false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			f, err := core.NewSourceFile("", strings.NewReader(tt.source), Language)
			if !assert.NoError(err) {
				return
			}

			details, found := getNextCallDetails(f.Tree().RootNode())

			assert.Equal(tt.wantCallDetails, details)
			assert.Equal(tt.wantFound, found)
		})
	}
}
