package iac2

import (
	"bytes"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	graph2 "github.com/klothoplatform/klotho/pkg/graph"
	"github.com/stretchr/testify/assert"
)

func TestOutputBody(t *testing.T) {
	fizz := &DummyFizz{Value: "my-hello"}
	buzz := DummyBuzz{}
	parent := &DummyBig{
		id:   "main",
		Fizz: fizz,
		Buzz: buzz,
	}
	graph := graph2.NewDirected[graph2.Identifiable]()
	graph.AddVertex(fizz)
	graph.AddVertex(buzz)
	graph.AddVertex(parent)
	graph.AddEdge(parent.Id(), fizz.Id())
	graph.AddEdge(parent.Id(), buzz.Id())
	graph.AddEdge(fizz.Id(), buzz.Id())

	compiler := CreateTemplatesCompiler(graph)
	compiler.templates = filesMapToFsMap(dummyTemplateFiles)

	t.Run("body", func(t *testing.T) {
		assert := assert.New(t)
		buf := bytes.Buffer{}
		err := compiler.RenderBody(&buf)
		if !assert.NoError(err) {
			return
		}
		expect := strings.TrimLeft(`
const dummyBuzzShared = new aws.buzz.DummyResource();
const dummyFizzMyHello = new aws.fizz.DummyResource("my-hello");
const dummyBigMain = new DummyParent(
				dummyFizzMyHello,
				{buzz: dummyBuzzShared});
`, "\n")
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

func (f *DummyFizz) Id() string {
	return f.Value
}

func (b DummyBuzz) Id() string {
	return "shared"
}

func (p *DummyBig) Id() string {
	return p.id
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
