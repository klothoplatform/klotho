package iac2

import (
	"fmt"
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

func Test_appliedOutputsToString(t *testing.T) {
	cases := []struct {
		name  string
		want  string
		input []AppliedOutput
	}{
		{
			name: "simple test",
			input: []AppliedOutput{
				{
					appliedName: fmt.Sprintf("%s.openIdConnectIssuerUrl", "awsEksClusterTestAppCluster1"),
					varName:     "cluster_oidc_url",
				},
				{
					appliedName: fmt.Sprintf("%s.arn", "awsEksClusterTestAppCluster1"),
					varName:     "cluster_arn",
				},
			},
			want: "pulumi.all([awsEksClusterTestAppCluster1.openIdConnectIssuerUrl, awsEksClusterTestAppCluster1.arn]).apply(([cluster_oidc_url, cluster_arn]) => { return ",
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			result := appliedOutputsToString(tt.input)
			assert := assert.New(t)
			assert.Equal(tt.want, result)
		})
	}

}

func Test_dedupeAppliedOutputs(t *testing.T) {
	cases := []struct {
		name    string
		want    []AppliedOutput
		input   []AppliedOutput
		wantErr bool
	}{
		{
			name: "simple test",
			input: []AppliedOutput{
				{
					appliedName: fmt.Sprintf("%s.openIdConnectIssuerUrl", "awsEksClusterTestAppCluster1"),
					varName:     "cluster_oidc_url",
				},
				{
					appliedName: fmt.Sprintf("%s.openIdConnectIssuerUrl", "awsEksClusterTestAppCluster1"),
					varName:     "cluster_oidc_url",
				},
			},
			want: []AppliedOutput{
				{
					appliedName: fmt.Sprintf("%s.openIdConnectIssuerUrl", "awsEksClusterTestAppCluster1"),
					varName:     "cluster_oidc_url",
				},
			},
		},
		{
			name: "var name conflict ",
			input: []AppliedOutput{
				{
					appliedName: fmt.Sprintf("%s.should differ", "awsEksClusterTestAppCluster1"),
					varName:     "cluster_oidc_url",
				},
				{
					appliedName: fmt.Sprintf("%s.openIdConnectIssuerUrl", "awsEksClusterTestAppCluster1"),
					varName:     "cluster_oidc_url",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			result, err := deduplicateAppliedOutputs(tt.input)

			if tt.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}

			assert.Equal(tt.want, result)
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
