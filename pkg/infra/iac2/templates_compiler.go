package iac2

import (
	"embed"
	_ "embed"
	"fmt"
	"github.com/klothoplatform/klotho/pkg/graph"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"github.com/mitchellh/hashstructure/v2"
	"github.com/pkg/errors"
	"io"
	"io/fs"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

type (
	varNamer interface {
		VariableName() string
	}

	resource struct {
		hash    string
		element any
	}

	templatesCompiler struct {
		templates             fs.FS
		resourceGraph         *graph.Directed[resource]
		resources             map[any]resource
		templatesByStructName map[string]ResourceCreationTemplate
	}
)

var (
	//go:embed templates/*/factory.ts
	standardTemplates embed.FS

	nonIdentifierChars = regexp.MustCompile(`\W`)
)

func CreateTemplatesCompiler() *templatesCompiler {
	subTemplates, err := fs.Sub(standardTemplates, "templates")
	if err != nil {
		panic(err) // unexpected, since standardTemplates is statically built into klotho
	}
	return &templatesCompiler{
		templates:             subTemplates,
		resourceGraph:         graph.NewDirected[resource](),
		resources:             make(map[any]resource),
		templatesByStructName: make(map[string]ResourceCreationTemplate),
	}
}

func (tc templatesCompiler) AddResource(v any) {
	if _, exists := tc.resources[v]; exists {
		return

	}
	res := resource{
		hash:    fmt.Sprintf(`%x`, len(tc.resources)),
		element: v,
	}
	tc.resources[v] = res
	tc.resourceGraph.AddVertex(res)
	for _, child := range getStructValues(v) {
		if reflect.TypeOf(child).Kind() == reflect.Struct {
			tc.AddResource(child)
			childRes := tc.getResource(child)
			tc.resourceGraph.AddEdge(res.Id(), childRes.Id())
		}
	}
}

func (tc templatesCompiler) getResource(v any) resource {
	childRes, childExists := tc.resources[v]
	if !childExists {
		panic(fmt.Sprintf(`compiler has inconsistent state: no resource for %v`, v))
	}
	return childRes
}

func (tc templatesCompiler) RenderBody(out io.Writer) error {
	// TODO: for now, assume a nice little tree
	// TODO: need a stable sorting of outputs!

	errs := multierr.Error{}
	for _, res := range tc.resourceGraph.Roots() {
		err := tc.renderResource(out, res.element)
		errs.Append(err)
	}
	return errs.ErrOrNil()
}

func (tc templatesCompiler) RenderImports(out io.Writer) error {
	// TODO: for now, assume a nice little tree

	allImports := make(map[string]struct{})
	for _, res := range tc.resources {
		tmpl := tc.GetTemplate(res.element)
		for statement, _ := range tmpl.imports {
			allImports[statement] = struct{}{}
		}
	}

	sortedImports := make([]string, 0, len(allImports))
	for statement, _ := range allImports {
		sortedImports = append(sortedImports, statement)
	}

	sort.Strings(sortedImports)
	for _, statement := range sortedImports {
		if _, err := out.Write([]byte(statement)); err != nil {
			return err
		}
		if _, err := out.Write([]byte("\n")); err != nil {
			return err
		}
	}

	return nil
}

func (tc templatesCompiler) renderResource(out io.Writer, res any) error {
	// TODO: for now, assume a nice little tree
	errs := multierr.Error{}

	inputArgs := make(map[string]string)
	for fieldName, child := range getStructValues(res) {
		childType := reflect.TypeOf(child) // todo cache in the resource?
		switch childType.Kind() {
		case reflect.String:
			inputArgs[fieldName] = quoteTsString(child.(string))
		case reflect.Struct:
			errs.Append(tc.renderResource(out, child))
			inputArgs[fieldName] = tc.getResource(child).VariableName()
		default:
			errs.Append(errors.Errorf(`unrecognized input type for %v [%s]: %v`, res, fieldName, child))
		}
	}

	varName := tc.getResource(res).VariableName()
	fmt.Fprintf(out, `const %s = `, varName)
	errs.Append(tc.GetTemplate(res).RenderCreate(out, inputArgs))
	out.Write([]byte(";\n"))

	return errs.ErrOrNil()

}

func (tc templatesCompiler) GetTemplate(v any) ResourceCreationTemplate {
	// TODO cache into the resource
	vType := reflect.TypeOf(v)
	typeName := vType.Name()
	existing, ok := tc.templatesByStructName[typeName]
	if ok {
		return existing
	}
	templateName := camelToSnake(typeName)
	contents, err := fs.ReadFile(tc.templates, templateName+`/factory.ts`)
	if err != nil {
		// Shouldn't ever happen; would mean an error in how we set up our structs
		panic(err)
	}
	template := ParseResourceCreationTemplate(contents)
	tc.templatesByStructName[typeName] = template
	return template
}

func (r resource) Id() string {
	if r.hash == "" {
		h, err := hashstructure.Hash(r.element, hashstructure.FormatV2, nil)
		if err != nil {
			// Shouldn't ever happen; would mean an error in how we set up our structs
			panic(err)
		}
		r.hash = fmt.Sprintf("%x", h)
	}
	return r.hash
}

func (r resource) VariableName() string {
	name := fmt.Sprintf(`%s_%s`, reflect.TypeOf(r.element).Name(), r.Id())
	firstChar := name[:1]
	rest := name[1:]
	return strings.ToLower(firstChar) + rest
}
