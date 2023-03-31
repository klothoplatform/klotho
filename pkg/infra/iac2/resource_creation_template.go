package iac2

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"strings"
	"text/template"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/query"
	"github.com/pkg/errors"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
)

type (
	ResourceCreationSignature struct {
		// InputTypes is the inputs that this template requires: a Go-struct reflection of the `interface Args` in
		// the TypeScript source. The keys are the field names in the `Args` interface, and the values are their types.
		InputTypes map[string]string
		// OutputType is the type that the template's `create(...)` function returns
		OutputType string
	}

	// ResourceCreationTemplate contains information about a parsed template
	ResourceCreationTemplate struct {
		ResourceCreationSignature
		// name is the template's name, which is only used for error reporting
		name string
		// Imports is a set of import statements that the template needs
		Imports map[string]struct{}
		// ExpressionTemplate is a Go-[text/template] for a TypeScript expression to generate a piece of infrastructure.
		ExpressionTemplate string
	}
)

var (
	tsLanguage = core.SourceLanguage{
		ID:     core.LanguageId("ts"),
		Sitter: typescript.GetLanguage(),
	}

	parameterizeArgsRegex = regexp.MustCompile(`\bargs(\.\w+)`)
	curlyEscapes          = regexp.MustCompile(`({+)(args\.)`)
	templateComments      = regexp.MustCompile(`//*TMPL\s+(.*)`)

	//go:embed find_args.scm
	findArgsQuery string

	//go:embed find_create_func.scm
	findCreateFuncQuery string

	//go:embed find_imports.scm
	findImportsQuery string
)

// ParseResourceCreationTemplate parses TypeScript file into a ResourceCreationTemplate, which TemplatesCompiler
// can then use. It looks for three things within the TypeScript source:
//
//  1. an imports section, which become the ResourceCreationTemplate's `imports` field.
//  2. an `interface Args`, which contains the inputs this template expects. Those turn into the
//     ResourceCreationSignature's `inputTypes` map.
//  3. a `function create(args: Args)`, which is expected to contain only a single `return` statement.
//
// The `create` function gets used in two ways:
//
//  1. Its return value becomes the ResourceCreationSignature `outputType`
//  2. Its `return` expression becomes the ResourceCreationTemplate `expressionTemplate`. As part of this, any usage of
//     `arg.Foo` will get turned into `{{.Foo}}` for use in a Go template.
func ParseResourceCreationTemplate(name string, contents []byte) ResourceCreationTemplate {
	node := parseFile(contents)

	result := ResourceCreationTemplate{name: name}

	// inputs
	result.InputTypes = make(map[string]string)
	nextInput := doQuery(node, findArgsQuery)
	for {
		match, found := nextInput()
		if !found {
			break
		}
		inputName, inputType := match["property_name"].Content(), match["property_type"].Content()
		result.InputTypes[inputName] = inputType
	}

	// return type and expression
	createFunc := doQuery(node, findCreateFuncQuery)
	create, found := createFunc()
	if !found {
		// unexpected, since all inputs are from resources in the klotho binary
		panic("couldn't find valid create() function")
	}
	result.OutputType = create["return_type"].Content()
	result.ExpressionTemplate = parameterizeArgs(create["return_body"].Content())

	// imports
	result.Imports = make(map[string]struct{})
	importsQuery := doQuery(node, findImportsQuery)
	for {
		match, found := importsQuery()
		if !found {
			break
		}
		importLine := match["import"].Content()
		// Trim any trailing semicolons. This helps normalize imports, so that we don't include them twice if one file
		// includes the semicolon and the other doesn't.
		importLine = strings.TrimRight(importLine, ";")
		result.Imports[importLine] = struct{}{}
	}

	return result
}

// doQuery is a thin wrapper around `query.Exec` to use typescript as the Language.
func doQuery(c *sitter.Node, q string) query.NextMatchFunc {
	return query.Exec(tsLanguage, c, q)
}

// parameterizeArgs turns "args.foo" into {{.Foo}}. It's very simplistic and just works off regex
func parameterizeArgs(contents string) string {
	// If the source has "{args.Foo}", then just turning "args.Foo" -> "{{.Foo}}" would result in "{{{.Foo}}}", which is
	// invalid go-template. So, we first turn "{args." into "{{`{`}}args.", which will eventually result in
	// "{{`{`}}{{.Foo}}" — which, while ugly, will result in the correct template execution.
	contents = curlyEscapes.ReplaceAllString(contents, "{{`$1`}}$2")
	contents = templateComments.ReplaceAllString(contents, "$1")
	contents = parameterizeArgsRegex.ReplaceAllString(contents, `{{parseTS $1}}`)
	return contents
}

func parseFile(contents []byte) *sitter.Node {
	parser := sitter.NewParser()
	parser.SetLanguage(tsLanguage.Sitter)
	tree, err := parser.ParseCtx(context.TODO(), nil, contents)
	if err != nil {
		panic(err) // unexpected, since all inputs are from resources in the klotho binary
	}
	return tree.RootNode()
}

func (t ResourceCreationTemplate) RenderCreate(out io.Writer, inputs map[string]templateValue) error {
	tmpl, err := template.New(t.name).Funcs(template.FuncMap{
		"parseTS": parseTS,
	}).Parse(t.ExpressionTemplate)
	if err != nil {
		return errors.Wrapf(err, `while writing template for %s`, t.name)
	}
	return tmpl.Execute(out, inputs)
}

// parseTS returns the parsed value of val if val implements templateValue or val's string representation otherwise
func parseTS(val reflect.Value) (string, error) {
	if templateVal, ok := val.Interface().(templateValue); ok {
		out, err := templateVal.Parse()
		if err != nil {
			return "", core.WrapErrf(err, "template value parsing failed")
		}
		return out, nil
	}
	return "", fmt.Errorf("invalid template value detected: %s: %v: template values must implement the templateValue interface", val.Kind().String(), val.Interface())
}
