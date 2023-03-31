package iac2

import (
	"strings"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func TestParseTemplate(t *testing.T) {
	assert := assert.New(t)
	parsed := ParseResourceCreationTemplate("dummy", []byte(simpleTemplateBody))

	assert.Equal(
		map[string]string{
			"input1": "string",
			"input2": "pulumi.Output<string>",
		},
		parsed.InputTypes)
	assert.Equal("aws.lambda.Function", parsed.OutputType)
	assert.Equal("new Function({{parseTS .blah}})", parsed.ExpressionTemplate)
	assert.Equal(
		map[string]struct{}{
			`import * as aws from '@pulumi/aws'`:   {},
			`import {Role} from "@pulumi/aws/iam"`: {},
		},
		parsed.Imports,
	)
}

func TestParameterizeArgs(t *testing.T) {
	cases := []struct {
		given  string
		want   string
		input  map[string]any
		result string
	}{
		{
			given:  `new Foo(args.Bar)`,
			want:   `new Foo({{parseTS .Bar}})`,
			input:  map[string]any{"Bar": `"HELLO"`},
			result: `new Foo("HELLO")`,
		},
		{
			given:  `new Foo({args.Bar})`,
			want:   "new Foo({{`{`}}{{parseTS .Bar}}})",
			input:  map[string]any{"Bar": `"HELLO"`},
			result: `new Foo({"HELLO"})`,
		},
		{
			given:  `new Foo({{args.Bar}})`, // two curlies
			want:   "new Foo({{`{{`}}{{parseTS .Bar}}}})",
			input:  map[string]any{"Bar": `"HELLO"`},
			result: `new Foo({{"HELLO"}})`,
		},
		{
			given: `new Foo(argsFoo)`,
			want:  `new Foo(argsFoo)`,
		},
		{
			given: `new Foo(myargs.Foo)`,
			want:  `new Foo(myargs.Foo)`,
		},
	}
	for _, tt := range cases {
		t.Run(tt.given, func(t *testing.T) {
			tmplStr := parameterizeArgs(tt.given)
			assert := assert.New(t)
			assert.Equal(tt.want, tmplStr, `template create`)

			if tt.input != nil {
				t.Run("template use", func(t *testing.T) {
					tmpl, err := template.New("template").Funcs(template.FuncMap{"parseTS": func(s string) string { return s }}).Parse(tmplStr)
					if assert.NoError(err) {
						buf := strings.Builder{}
						err := tmpl.Execute(&buf, tt.input)
						if assert.NoError(err) {
							assert.Equal(tt.result, buf.String(), `template use`)
						}
					}
				})
			}
		})
	}

}

const simpleTemplateBody = `
import * as aws from '@pulumi/aws'
import {Role} from "@pulumi/aws/iam";

interface Args {
  	input1: string,
	input2: pulumi.Output<string>,
}

function create(args: Args): aws.lambda.Function {
	return new Function(args.blah);
}
`
