package iac2

import (
	"bytes"
	"io/fs"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/graph"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/klothoplatform/klotho/pkg/provider/imports"
	"github.com/stretchr/testify/assert"
)

func TestOutputBody(t *testing.T) {
	fizz := &DummyFizz{Value: "my-hello"}
	buzz := DummyBuzz{}
	void := DummyVoid{}
	parent := &DummyBig{
		id:        "main",
		Fizz:      fizz,
		Buzz:      buzz,
		NestedDoc: &NestedResource{Fizz: fizz},
		NestedTemplate: &NestedTemplate{
			Str: "strVal",
			Arr: []string{"val1", "val2"},
		},
	}
	thingToImport := &DummyFizz{Value: "imported"}
	graph := core.NewResourceGraph()
	graph.AddResource(fizz)
	graph.AddResource(buzz)
	graph.AddResource(parent)
	graph.AddResource(void)
	graph.AddDependency(parent, fizz)
	graph.AddDependency(parent, buzz)
	graph.AddDependency(parent, void)
	graph.AddDependency(fizz, buzz)
	graph.AddDependency(void, fizz)
	graph.AddDependency(thingToImport, &imports.Imported{ID: "fizz-123"})

	compiler := CreateTemplatesCompiler(graph)
	compiler.templates = filesMapToFsMap(dummyTemplateFiles)

	t.Run("body", func(t *testing.T) {
		assert := assert.New(t)
		buf := bytes.Buffer{}
		err := compiler.RenderBody(&buf)
		if !assert.NoError(err) {
			return
		}
		expect := s(
			"const buzzShared = new aws.buzz.DummyResource();",
			"",
			"const fizzMyHello = new aws.fizz.DummyResource(`my-hello`);",
			"",
			"fs.ReadFile();",
			"",
			`const fizzImported = aws.fizz.DummyResource.get("fizz-imported", "fizz-123")`,
			"",
			"const bigMain = new DummyParent(",
			"				fizzMyHello,",
			"				{",
			"					buzz: buzzShared,",
			"					nestedDoc: {fizz: fizzMyHello,",
			"}",
			"					nestedTemplate: ",
			"		{",
			"			str: \"strVal\"",
			"			arr0: \"val1\"",
			"			arr1: \"val2\"",
			"		}",
			"					rawNestedTemplate: true",
			"				});",
		)
		assert.Equal(expect, buf.String())
	})
	t.Run("imports", func(t *testing.T) {
		assert := assert.New(t)
		buf := bytes.Buffer{}
		err := compiler.RenderImports(&buf)
		if !assert.NoError(err) {
			return
		}
		expect := strings.TrimLeft(`
import * as aws from '@pulumi/aws'
import * as inputs from '@pulumi/aws/types/input'
import {Whatever} from "@pulumi/aws/cool/service"
`, "\n")
		assert.Equal(expect, buf.String())
	})
}

func TestResolveStructInput(t *testing.T) {
	cases := []struct {
		name                   string
		parentResource         *core.Resource
		value                  any
		withVars               map[string]string
		useDoubleQuotedStrings bool
		want                   string
	}{
		{
			name:  "string",
			value: "hello, world",
			want:  "`hello, world`",
		},
		{
			name:  "bool",
			value: true,
			want:  `true`,
		},
		{
			name:  "int",
			value: 123,
			want:  `123`,
		},
		{
			name:  "float",
			value: 1234.5,
			want:  `1234.5`,
		},
		{
			name:     "struct",
			value:    DummyBuzz{},
			withVars: map[string]string{`buzz-shared`: `myVar`},
			want:     `myVar`,
		},
		{
			name:     "struct-pointer",
			value:    &DummyFizz{Value: `abc`},
			withVars: map[string]string{`fizz-abc`: `myVar`},
			want:     `myVar`,
		},
		{
			name:  "null",
			value: nil,
			want:  `null`,
		},
		{
			name:     "slice of resources",
			value:    []core.Resource{&DummyFizz{Value: `abc`}},
			withVars: map[string]string{`fizz-abc`: `myVar`},
			want:     `[myVar]`,
		},
		{
			name:     "slice of any",
			value:    []any{123, &DummyFizz{Value: `abc`}},
			withVars: map[string]string{`fizz-abc`: `myVar`},
			want:     `[123,myVar]`,
		},
		{
			name:     "array",
			value:    [2]any{123, &DummyFizz{Value: `abc`}},
			withVars: map[string]string{`fizz-abc`: `myVar`},
			want:     `[123,myVar]`,
		},
		{
			name: "map",
			value: map[string]any{
				"MyStruct": DummyBuzz{},
			},
			withVars:               map[string]string{`buzz-shared`: `myVar`},
			want:                   `{"MyStruct":myVar}`,
			useDoubleQuotedStrings: true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			tc := TemplatesCompiler{
				resourceVarNamesById: make(map[core.ResourceId]string),
			}
			for k, v := range tt.withVars {
				tc.resourceVarNamesById[core.ResourceId{Name: k}] = v
			}
			resourceVal := reflect.ValueOf(tt.parentResource)
			val := reflect.ValueOf(tt.value)
			actual, err := tc.resolveStructInput(&resourceVal, val, tt.useDoubleQuotedStrings, nil)
			assert.NoError(err)
			assert.Equal(tt.want, actual)
		})
	}
}

func Test_renderGlueVars(t *testing.T) {
	vpc := &resources.Vpc{Name: "vpc"}
	cases := []struct {
		name                 string
		subResource          core.Resource
		nodes                []core.Resource
		edges                []graph.Edge[core.Resource]
		resourceVarNamesById map[core.ResourceId]string
		want                 string
	}{
		{
			name:        "routeTableAssociation",
			subResource: &resources.RouteTable{Name: "rt1"},
			nodes: []core.Resource{
				vpc,
				&resources.RouteTable{Name: "rt1"},
				&resources.Subnet{Name: "s1", Vpc: vpc},
			},
			edges: []graph.Edge[core.Resource]{
				{
					Source:      &resources.RouteTable{Name: "rt1"},
					Destination: &resources.Subnet{Name: "s1", Vpc: vpc},
				},
			},
			resourceVarNamesById: map[core.ResourceId]string{
				{Provider: "aws", Type: "route_table", Name: "rt1"}: "testRouteTable",
				{Provider: "aws", Type: "vpc_subnet", Name: "s1"}:   "subnet1",
			},
			want: "\n\nconst routeTableAssociationS1 = new aws.ec2.RouteTableAssociation(`s1`, {\n\t\t\t\tsubnetId: vpcSubnetS1.id,\n\t\t\trouteTableId: testRouteTable.id,\n\t\t\t});\n\n",
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			tc := CreateTemplatesCompiler(core.NewResourceGraph())
			tc.resourceVarNamesById = tt.resourceVarNamesById
			tc.templates = filesMapToFsMap(subResourceTemplateFiles)
			for _, res := range tt.nodes {
				tc.resourceGraph.AddResource(res)
			}
			for _, edge := range tt.edges {
				tc.resourceGraph.AddDependency(edge.Source, edge.Destination)
			}
			buf := &bytes.Buffer{}

			err := tc.renderGlueVars(buf, tt.subResource)
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.want, buf.String())
		})
	}
}

type (
	DummyFizz struct {
		Value string
	}

	DummyBuzz struct {
		// nothing
	}

	DummyBig struct {
		id             string
		Fizz           *DummyFizz
		Buzz           DummyBuzz
		NestedDoc      *NestedResource
		NestedTemplate *NestedTemplate
	}

	DummyVoid struct {
		// nothing
	}

	NestedResource struct {
		Fizz *DummyFizz
	}

	NestedTemplate struct {
		Str string
		Arr []string
	}
)

func (f *DummyFizz) Id() core.ResourceId                      { return core.ResourceId{Name: "fizz-" + f.Value} }
func (f *DummyFizz) BaseConstructRefs() core.BaseConstructSet { return nil }
func (f *DummyFizz) DeleteContext() core.DeleteContext {
	return core.DeleteContext{}
}
func (b DummyBuzz) Id() core.ResourceId                      { return core.ResourceId{Name: "buzz-shared"} }
func (f DummyBuzz) BaseConstructRefs() core.BaseConstructSet { return nil }
func (f DummyBuzz) DeleteContext() core.DeleteContext {
	return core.DeleteContext{}
}
func (p *DummyBig) Id() core.ResourceId                      { return core.ResourceId{Name: "big-" + p.id} }
func (f *DummyBig) BaseConstructRefs() core.BaseConstructSet { return nil }
func (f *DummyBig) DeleteContext() core.DeleteContext {
	return core.DeleteContext{}
}
func (p DummyVoid) Id() core.ResourceId                      { return core.ResourceId{Name: "void"} }
func (f DummyVoid) BaseConstructRefs() core.BaseConstructSet { return nil }
func (f DummyVoid) DeleteContext() core.DeleteContext {
	return core.DeleteContext{}
}

var dummyTemplateFiles = map[string]string{
	`dummy_fizz/factory.ts`: `
		import * as aws from '@pulumi/aws'

		interface Args {
			Value: string,
		}

		function create(args: Args): aws.fizz.DummyResource {
			return new aws.fizz.DummyResource(args.Value);
		}`,

	`dummy_buzz/factory.ts`: `
		import * as aws from '@pulumi/aws'
		import {Whatever} from "@pulumi/aws/cool/service"; // Note the trailing semicolon. It'll get removed.

		interface Args {}

		function create(args: Args): aws.buzz.DummyResource {
			return new aws.buzz.DummyResource();
		}`,

	`dummy_void/factory.ts`: `
		import * as aws from '@pulumi/aws'
		import {Whatever} from "@pulumi/aws/cool/service"; // Note the trailing semicolon. It'll get removed.

		interface Args {}

		function create(args: Args): void {
			fs.ReadFile();
		}`,

	`dummy_big/nested_template.ts.tmpl`: `
		{
			str: "{{.Str}}"
			{{- range $index, $val := .Arr }}
			arr{{$index}}: "{{$val}}"
			{{- end}}
		}`,
	`dummy_big/factory.ts`: `
		import * as aws from '@pulumi/aws'
		import * as inputs from '@pulumi/aws/types/input'

		interface Args {
			Fizz: aws.fizz.DummyResource,
			Buzz: aws.buzz.DummyResource,
			NestedDoc: aws.nest.DummyResource
			NestedTemplate: pulumi.Input<inputs.nest.NestedInput>
		}

		function create(args: Args): aws.foobar.DummyParent {
			return new DummyParent(
				args.Fizz,
				{
					buzz: args.Buzz,
					nestedDoc: args.NestedDoc
					nestedTemplate: args.NestedTemplate
					//TMPL {{- if eq .NestedTemplate.Raw.Str "strVal"}}
					rawNestedTemplate: true
					//TMPL {{- end}}
				});
		}`,
}

var subResourceTemplateFiles = map[string]string{
	"role_policy_attachment/factory.ts": `import * as aws from '@pulumi/aws'
				import * as pulumi from '@pulumi/pulumi'
				
				interface Args {
					Name: string
					Policy: aws.iam.Policy
					Role: aws.iam.Role
				}
				
				// noinspection JSUnusedLocalSymbols
				function create(args: Args): aws.iam.RolePolicyAttachment {
					return new aws.iam.RolePolicyAttachment(args.Name, {
						policyArn: args.Policy.arn,
						role: args.Role
					})
				}`,
	"route_table_association/factory.ts": `import * as aws from '@pulumi/aws'

		interface Args {
			Name: string
			Subnet: aws.ec2.Subnet
			RouteTable: aws.ec2.RouteTable
		}
		
		// noinspection JSUnusedLocalSymbols
		function create(args: Args): aws.ec2.RouteTableAssociation {
			return new aws.ec2.RouteTableAssociation(args.Name, {
				subnetId: args.Subnet.id,
			routeTableId: args.RouteTable.id,
			})
		}`,
}

func filesMapToFsMap(files map[string]string) fs.FS {
	mockFs := make(fstest.MapFS)
	for path, contents := range files {
		mockFs[path] = &fstest.MapFile{
			Data:    []byte(contents),
			Mode:    0700,
			ModTime: time.Now(),
			Sys:     struct{}{},
		}
	}
	return mockFs
}

// s joins all the inputs via newline. We use it because the output might itself contain backticks, which makes the
// built-in go multiline strings unuseful.
func s(lines ...string) string {
	return strings.Join(lines, "\n")
}
