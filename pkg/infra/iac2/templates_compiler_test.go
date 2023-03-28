package iac2

import (
	"bytes"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/stretchr/testify/assert"
	"io/fs"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	graph2 "github.com/klothoplatform/klotho/pkg/graph"
)

func TestOutputBody(t *testing.T) {
	fizz := &DummyFizz{Value: "my-hello"}
	buzz := DummyBuzz{}
	parent := &DummyBig{
		id:   "main",
		Fizz: fizz,
		Buzz: buzz,
	}
	graph := graph2.NewDirected[core.Resource]()
	graph.AddVertex(fizz)
	graph.AddVertex(buzz)
	graph.AddVertex(parent)
	graph.AddEdge(fizz.Id(), parent.Id())
	graph.AddEdge(buzz.Id(), parent.Id())
	graph.AddEdge(buzz.Id(), fizz.Id())

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
			"const bigMain = new DummyParent(",
			"				fizzMyHello,",
			"				{buzz: buzzShared});")
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
import {Whatever} from "@pulumi/aws/cool/service"
`, "\n")
		assert.Equal(expect, buf.String())
	})
}

func TestResolveStructInput(t *testing.T) {
	cases := []struct {
		name     string
		value    any
		withVars map[string]string
		want     string
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
			withVars: map[string]string{`buzz-shared`: `myVar`},
			want:     "{`MyStruct`:myVar}",
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			tc := TemplatesCompiler{
				resourceVarNamesById: tt.withVars,
			}
			val := reflect.ValueOf(tt.value)
			actual := tc.resolveStructInput(val)
			assert.Equal(tt.want, actual)
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
		id   string
		Fizz *DummyFizz
		Buzz DummyBuzz
	}
)

func (f *DummyFizz) Id() string                               { return "fizz-" + f.Value }
func (f *DummyFizz) Provider() string                         { return "DummyProvider" }
func (f *DummyFizz) KlothoConstructRef() []core.AnnotationKey { return nil }

func (b DummyBuzz) Id() string                               { return "buzz-shared" }
func (f DummyBuzz) Provider() string                         { return "DummyProvider" }
func (f DummyBuzz) KlothoConstructRef() []core.AnnotationKey { return nil }

func (p *DummyBig) Id() string                               { return "big-" + p.id }
func (f *DummyBig) Provider() string                         { return "DummyProvider" }
func (f *DummyBig) KlothoConstructRef() []core.AnnotationKey { return nil }

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

	`dummy_big/factory.ts`: `
		import * as aws from '@pulumi/aws'

		interface Args {
			Fizz: aws.fizz.DummyResource,
			Buzz: aws.buzz.DummyResource,
		}

		function create(args: Args): aws.foobar.DummyParent {
			return new DummyParent(
				args.Fizz,
				{buzz: args.Buzz});
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
