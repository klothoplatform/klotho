package iac2

import (
	"context"
	_ "embed"
	"fmt"
	"strings"
	"testing"
	"text/template"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/stretchr/testify/assert"
)

func TestParseTemplate(t *testing.T) {
	t.Run("basic creation", func(t *testing.T) {
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
	})

	t.Run("bad return panic", func(t *testing.T) {
		assert := assert.New(t)
		defer func() {
			r := recover()
			assert.NotNil(r)
		}()

		ParseResourceCreationTemplate("test", []byte(badReturnTemplateBody))
	})
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
		{
			given: `//TMPL test`,
			want:  `test`,
		},
		{
			given:  `//TMPL {{ .Param }}`,
			want:   `{{ .Param }}`,
			input:  map[string]any{"Param": "value"},
			result: `value`,
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

const badReturnTemplateBody = `
import * as aws from '@pulumi/aws'
import {Role} from "@pulumi/aws/iam";

interface Args {
  	input1: string,
	input2: pulumi.Output<string>,
}

function create(args: Args): aws.lambda.Function {
	const a = 1;
	return a;
}
`

func Test_bodyContents(t *testing.T) {
	statementQuery, err := sitter.NewQuery([]byte("(statement_block) @v"), javascript.GetLanguage())
	if err != nil {
		t.Fail()
		return
	}
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name: "simple",
			content: `{
	fs.ReadFile();
}`,
			want: "fs.ReadFile()",
		},
		{
			name: "multiline",
			content: `{
	fs.ReadFile();
	fs.WriteFile();
}`,
			want: `fs.ReadFile();
fs.WriteFile()`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			parser := sitter.NewParser()
			parser.SetLanguage(javascript.GetLanguage())
			js, err := parser.ParseCtx(context.Background(), nil, []byte(tt.content))
			if !assert.NoError(err) {
				return
			}

			cursor := sitter.NewQueryCursor()
			cursor.Exec(statementQuery, js.RootNode())

			match, ok := cursor.NextMatch()
			if !assert.True(ok) || !assert.Len(match.Captures, 1) {
				return
			}

			got := bodyContents(match.Captures[0].Node)

			assert.Equal(tt.want, got)
		})
	}
}
