package iac2

import (
	"context"
	_ "embed"
	"io"
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
		inputTypes map[string]string
		outputType string
	}

	ResourceCreationTemplate struct {
		ResourceCreationSignature
		name               string
		imports            map[string]struct{}
		expressionTemplate string
	}
)

var (
	tsLanguage = core.SourceLanguage{
		ID:     core.LanguageId("ts"),
		Sitter: typescript.GetLanguage(),
	}

	parameterizeArgsRegex = regexp.MustCompile(`args(\.\w+)`)
	curlyEscapes          = regexp.MustCompile(`({+)(args\.)`)

	//go:embed find_args.scm
	findArgsQuery string

	//go:embed find_create_func.scm
	findCreateFuncQuery string

	//go:embed find_imports.scm
	findImportsQuery string
)

func ParseResourceCreationTemplate(name string, contents []byte) ResourceCreationTemplate {
	node := parseFile(contents)

	result := ResourceCreationTemplate{name: name}

	// inputs
	result.inputTypes = make(map[string]string)
	nextInput := doQuery(node, findArgsQuery)
	for {
		match, found := nextInput()
		if !found {
			break
		}
		inputName, inputType := match["property_name"].Content(), match["property_type"].Content()
		result.inputTypes[inputName] = inputType
	}

	// return type and expression
	createFunc := doQuery(node, findCreateFuncQuery)
	create, found := createFunc()
	if !found {
		// unexpected, since all inputs are from resources in the klotho binary
		panic("couldn't find valid create() function")
	}
	result.outputType = create["return_type"].Content()
	result.expressionTemplate = parameterizeArgs(create["return_body"].Content())

	// imports
	result.imports = make(map[string]struct{})
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
		result.imports[importLine] = struct{}{}
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
	contents = parameterizeArgsRegex.ReplaceAllString(contents, `{{$1}}`)
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

func (t ResourceCreationTemplate) RenderCreate(out io.Writer, inputs map[string]string) error {
	tmpl, err := template.New("template").Parse(t.expressionTemplate)
	if err != nil {
		return errors.Wrapf(err, `while writing template for %s`, t.name)
	}
	return tmpl.Execute(out, inputs)
}