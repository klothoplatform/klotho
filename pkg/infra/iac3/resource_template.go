package iac3

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/klothoplatform/klotho/pkg/query"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
)

type (
	ResourceTemplate struct {
		Name              string
		Imports           []string
		OutputType        string
		Template          *template.Template
		PropertyTemplates map[string]*template.Template
		Args              map[string]Arg
		Path              string
	}

	PropertyTemplateData struct {
		Object string
		Input  map[string]any
	}

	Arg struct {
		Name    string
		Type    string
		Wrapper string
	}

	WrapperType string
)

const (
	TemplateWrapper       WrapperType = "TemplateWrapper"
	CamelCaseWrapper      WrapperType = "CamelCaseWrapper"
	LowerCamelCaseWrapper WrapperType = "LowerCamelCaseWrapper"
	ModelCaseWrapper      WrapperType = "ModelCaseWrapper"
)

var (
	//go:embed find_create_func.scm
	findCreateFuncQuery string

	//go:embed find_imports.scm
	findImportsQuery string

	//go:embed find_return.scm
	findReturn string

	//go:embed find_props_func.scm
	findPropsFuncQuery string

	//go:embed find_property.scm
	findPropertyQuery string

	//go:embed find_args.scm
	findArgs string

	curlyEscapes     = regexp.MustCompile(`~~{{`)
	templateComments = regexp.MustCompile(`//*TMPL\s+`)
)

func ParseTemplate(name string, r io.Reader) (*ResourceTemplate, error) {
	rt := &ResourceTemplate{Name: name}

	node, err := parseFile(r)
	if err != nil {
		return nil, err
	}

	rt.Template, rt.OutputType, err = createNodeToTemplate(node, name)
	if err != nil {
		return nil, err
	}
	rt.PropertyTemplates, err = propertiesNodeToTemplate(node, name)
	if err != nil {
		return nil, err
	}
	rt.Imports, err = importsFromTemplate(node)
	if err != nil {
		return nil, err
	}
	rt.Args, err = parseArgs(node, name)
	if err != nil {
		return nil, err
	}

	return rt, nil
}

func parseArgs(node *sitter.Node, name string) (map[string]Arg, error) {
	argsFunc := doQuery(node, findArgs)
	args := map[string]Arg{}
	for {
		argMatches, found := argsFunc()
		if !found {
			break
		}
		interfaceName := argMatches["name"].Content()
		if interfaceName != "Args" {
			continue
		}
		argName := argMatches["property_name"].Content()
		argType := argMatches["property_type"].Content()
		argWrapper := argMatches["nested"]
		if argWrapper == nil {
			args[argName] = Arg{Name: argName, Type: argType}
			continue
		}
		args[argName] = Arg{Name: argName, Type: argType, Wrapper: argWrapper.Content()}
	}

	return args, nil
}

func createNodeToTemplate(node *sitter.Node, name string) (*template.Template, string, error) {
	createFunc := doQuery(node, findCreateFuncQuery)
	create, found := createFunc()
	if !found {
		return nil, "", fmt.Errorf("no create function found in %s", name)
	}
	outputType := create["return_type"].Content()
	var expressionBody string
	if outputType == "void" {
		expressionBody = bodyContents(create["body"])
	} else {
		body, found := doQuery(create["body"], findReturn)()
		if !found {
			return nil, "", fmt.Errorf("no 'return' found in %s body:```\n%s\n```", name, create["body"].Content())
		}
		expressionBody = body["return_body"].Content()
	}
	expressionBody = parameterizeArgs(expressionBody, "")
	expressionBody = templateComments.ReplaceAllString(expressionBody, "")

	// transform escaped double curly brace literals e.g. ~~{{ .ID }} -> {{ `{{` }} .ID }}
	expressionBody = curlyEscapes.ReplaceAllString(expressionBody, "{{ `{{` }}")
	tmpl, err := template.New(name).Parse(expressionBody)

	return tmpl, outputType, err
}

func propertiesNodeToTemplate(node *sitter.Node, name string) (map[string]*template.Template, error) {
	propsFunc := doQuery(node, findPropsFuncQuery)
	propsNode, found := propsFunc()
	if !found {
		return nil, nil
	}

	propTemplates := make(map[string]*template.Template)
	var errs error
	nextProp := doQuery(propsNode["body"], findPropertyQuery)
	for {
		propMatches, found := nextProp()
		if !found {
			break
		}
		propName := propMatches["key"].Content()
		valueBase := propMatches["value"].Content()
		valueBase = parameterizeArgs(valueBase, ".Input")
		valueBase = parameterizeObject(valueBase)
		t, err := template.New(propName).Parse(valueBase)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("error parsing property %q: %w", propName, err))
			continue
		}
		propTemplates[propName] = t
	}
	return propTemplates, errs
}

var templateTSLang = types.SourceLanguage{
	ID:     types.LanguageId("ts"),
	Sitter: typescript.GetLanguage(),
}

func doQuery(c *sitter.Node, q string) query.NextMatchFunc {
	return query.Exec(templateTSLang, c, q)
}

func parseFile(r io.Reader) (*sitter.Node, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	parser := sitter.NewParser()
	parser.SetLanguage(templateTSLang.Sitter)
	tree, err := parser.ParseCtx(context.TODO(), nil, content)
	if err != nil {
		return nil, err
	}
	return tree.RootNode(), nil
}

// bodyContents returns the contents of a 'statement_block' with the surrounding {}
// and indentation removed so that the contents of a void function
// can be inlined with the rest of the index.ts.
func bodyContents(node *sitter.Node) string {
	if node.ChildCount() == 0 || node.Child(0).Content() != "{" {
		return node.Content()
	}
	var buf strings.Builder
	buf.Grow(len(node.Content()))
	for i := 0; i < int(node.NamedChildCount()); i++ {
		if i > 0 {
			buf.WriteRune('\n')
		}
		buf.WriteString(node.NamedChild(i).Content())
	}
	return strings.TrimSuffix(buf.String(), ";") // Remove any trailing ';' since one is added later to prevent ';;'
}

var (
	curlyArgsEscapes      = regexp.MustCompile(`({+)(args\.)`)
	parameterizeArgsRegex = regexp.MustCompile(`\bargs(\.\w+)`)
)

// parameterizeArgs turns "args.foo" into {{.Foo}}. It's very simplistic and just works off regex
// If the source has "{args.Foo}", then just turning "args.Foo" -> "{{.Foo}}" would result in "{{{.Foo}}}", which is
// invalid go-template. So, we first turn "{args." into "{{`{`}}args.", which will eventually result in
// "{{`{`}}{{.Foo}}" — which, while ugly, will result in the correct template execution.
func parameterizeArgs(contents string, argPrefix string) string {
	contents = curlyArgsEscapes.ReplaceAllString(contents, "{{`$1`}}$2")
	contents = parameterizeArgsRegex.ReplaceAllString(contents, fmt.Sprintf(`{{%s$1}}`, argPrefix))
	return contents
}

var (
	curlyObjectEscapes      = regexp.MustCompile(`({+)(object\.)`)
	parameterizeObjectRegex = regexp.MustCompile(`\bobject(\.\w+)`)
)

// parameterizeObject is like [parameterizeArgs], but for "object.foo" -> "{{.Object}}.foo".
func parameterizeObject(contents string) string {
	contents = curlyObjectEscapes.ReplaceAllString(contents, "{{`$1`}}$2")
	contents = parameterizeObjectRegex.ReplaceAllString(contents, `{{.Object}}$1`)
	return contents
}

func importsFromTemplate(node *sitter.Node) ([]string, error) {
	imports := make(map[string]struct{})
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
		imports[importLine] = struct{}{}
	}
	list := make([]string, 0, len(imports))
	for imp := range imports {
		list = append(list, imp)
	}
	sort.Strings(list)
	return list, nil
}
