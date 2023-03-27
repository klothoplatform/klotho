package iac2

import (
	"github.com/stretchr/testify/assert"
	"testing"
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
	assert.Equal("new Function({{.blah}})", parsed.ExpressionTemplate)
	assert.Equal(
		map[string]struct{}{
			`import * as aws from '@pulumi/aws'`:   {},
			`import {Role} from "@pulumi/aws/iam"`: {},
		},
		parsed.Imports,
	)
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
