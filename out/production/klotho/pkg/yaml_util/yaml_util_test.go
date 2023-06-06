package yaml_util

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSetValue(t *testing.T) {
	cases := []struct {
		name      string
		input     string
		set       string
		to        string
		expect    string
		expectErr string
	}{
		{
			name:   "happy path",
			input:  "hello: world",
			set:    "goodbye",
			to:     "farewell",
			expect: "hello: world\ngoodbye: farewell\n",
		},
		{
			name:   "input is empty",
			input:  "",
			set:    "hello",
			to:     "world",
			expect: "hello: world\n",
		},
		{
			name:  "new deep option",
			input: "",
			set:   "one.two.three",
			to:    "123",
			// note: the expected text uses string indentation, not tabs
			expect: `one:
    two:
        three: 123
`,
		},
		{
			name:   "existing deep option",
			input:  "one:\n  two: hello",
			set:    "one.two",
			to:     "goodbye",
			expect: "one:\n    two: goodbye\n",
		},
		{
			name:   "comments get preserved",
			input:  "#top-comment\nhello: world # this is my comment",
			set:    "hello",
			to:     "earth",
			expect: "#top-comment\nhello: earth # this is my comment\n",
		},
		{
			name:      "doc is a scalar",
			input:     "123",
			set:       "hello",
			to:        "world",
			expectErr: `can't set the path "hello"'`,
		},
		{
			name:   "overwrite scalar",
			input:  "hello: world",
			set:    "hello",
			to:     "goodbye",
			expect: "hello: goodbye\n",
		},
		{
			name:   "overwrite scalar with different-typed scalar",
			input:  "hello: 123",
			set:    "hello",
			to:     "world",
			expect: "hello: world\n",
		},
		{
			name:      "overwrite map with scalar",
			input:     "hello:\n  greet_target: world",
			set:       "hello",
			to:        "world",
			expectErr: "\"hello\" cannot be a scalar",
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			actual, err := SetValue([]byte(tt.input), tt.set, tt.to)
			assert.Equal(tt.expect, string(actual))
			var errStr string
			if err != nil {
				errStr = err.Error()
			}
			assert.Equal(tt.expectErr, errStr)
		})
	}
}

func TestCheckValid(t *testing.T) {
	// all cases validate against the dummyData struct
	cases := []struct {
		name           string
		yaml           string
		strict         bool
		successLenient bool
		successStrict  bool
	}{
		{
			name:           "happy path",
			yaml:           "str_value: hello\nint_value: 123",
			successLenient: true,
			successStrict:  true,
		},
		{
			name:           "missing fields",
			yaml:           "str_value: hello",
			successLenient: true,
			successStrict:  true,
		},
		{
			name:           "extra fields",
			yaml:           "bogus_value: hello",
			successLenient: true,
			successStrict:  false,
		},
		{
			name:           "wrongly typed fields",
			yaml:           "int_value: not a number",
			successLenient: false,
			successStrict:  false,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			assert.Equal(tt.successLenient, CheckValid[dummyData]([]byte(tt.yaml), Lenient) == nil, "lenient check")
			assert.Equal(tt.successStrict, CheckValid[dummyData]([]byte(tt.yaml), Strict) == nil, "strict check")
		})
	}
}

type dummyData struct {
	StrValue string `yaml:"str_value"`
	IntValue int    `yaml:"int_value"`
}
