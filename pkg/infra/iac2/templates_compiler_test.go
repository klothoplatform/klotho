package iac2

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

func TestOutputBody(t *testing.T) {

	compiler := CreateTemplatesCompiler()
	compiler.templates = filesMapToFsMap(dummyTemplateFiles)
	compiler.AddResource(
		DummyResourceParent{
			Fizz: DummyResourceLeafFizz{
				Value: "my-hello",
			},
			Buzz: DummyResourceLeafBuzz{},
		})

	t.Run("body", func(t *testing.T) {
		assert := assert.New(t)
		buf := bytes.Buffer{}
		err := compiler.RenderBody(&buf)
		if !assert.NoError(err) {
			return
		}
		expect := strings.TrimLeft(`
const dummyResourceLeafFizz_1 = new aws.fizz.DummyResource("my-hello");
const dummyResourceLeafBuzz_2 = new aws.buzz.DummyResource();
const dummyResourceParent_0 = new DummyParent(
				dummyResourceLeafFizz_1,
				{buzz: dummyResourceLeafBuzz_2});
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
	DummyResourceLeafFizz struct {
		Value string
	}

	DummyResourceLeafBuzz struct {
		// nothing
	}

	DummyResourceParent struct {
		Fizz DummyResourceLeafFizz
		Buzz DummyResourceLeafBuzz
	}
)

var dummyTemplateFiles = map[string]string{
	`dummy_resource_leaf_fizz/factory.ts`: `
		import * as aws from '@pulumi/aws'

		interface Args {
			Value: string,
		}

		function create(args: Args): aws.fizz.DummyResource {
			return new aws.fizz.DummyResource(args.Value);
		}`,

	`dummy_resource_leaf_buzz/factory.ts`: `
		import * as aws from '@pulumi/aws'
		import {Whatever} from "@pulumi/aws/cool/service"; // Note the trailing semicolon. It'll get removed.

		interface Args {}

		function create(args: Args): aws.buzz.DummyResource {
			return new aws.buzz.DummyResource();
		}`,

	`dummy_resource_parent/factory.ts`: `
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
