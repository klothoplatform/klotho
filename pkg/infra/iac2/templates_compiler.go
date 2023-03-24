package iac2

import (
	"embed"
	_ "embed"
	"fmt"
	"github.com/klothoplatform/klotho/pkg/graph"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"github.com/pkg/errors"
	"io"
	"io/fs"
	"reflect"
	"regexp"
	"sort"
)

type (
	varNamer interface {
		VariableName() string
	}

	templatesCompiler struct {
		// templates is the fs.FS where we read all of our `<struct>/factory.ts` files
		templates fs.FS
		// resourceGraph is the graph of resources to render
		resourceGraph *graph.Directed[graph.Identifiable]
		// templatesByStructName is a cache from struct name (e.g. "CloudwatchLogs") to the template for that struct.
		templatesByStructName map[string]ResourceCreationTemplate
		// resourceVarNames is a set of all variable names
		resourceVarNames map[string]struct{}
		// resourceVarNamesById is a map from resource id to the variable name for that resource
		resourceVarNamesById map[string]string
	}
)

var (
	//go:embed templates/*/factory.ts
	standardTemplates embed.FS

	nonIdentifierChars = regexp.MustCompile(`\W`)
)

func CreateTemplatesCompiler(resources *graph.Directed[graph.Identifiable]) *templatesCompiler {
	subTemplates, err := fs.Sub(standardTemplates, "templates")
	if err != nil {
		panic(err) // unexpected, since standardTemplates is statically built into klotho
	}
	return &templatesCompiler{
		templates:             subTemplates,
		resourceGraph:         resources,
		templatesByStructName: make(map[string]ResourceCreationTemplate),
		resourceVarNames:      make(map[string]struct{}),
		resourceVarNamesById:  make(map[string]string),
	}
}

func (tc templatesCompiler) RenderBody(out io.Writer) error {
	errs := multierr.Error{}
	vertexIds, err := tc.resourceGraph.VertexIdsInTopologicalOrder()
	if err != nil {
		return err
	}
	reverseInPlace(vertexIds)
	for _, id := range vertexIds {
		resource := tc.resourceGraph.GetVertex(id)
		err := tc.renderResource(out, resource)
		errs.Append(err)
	}
	return errs.ErrOrNil()
}

func (tc templatesCompiler) RenderImports(out io.Writer) error {
	errs := multierr.Error{}

	allImports := make(map[string]struct{})
	for _, res := range tc.resourceGraph.GetAllVertices() {
		tmpl, err := tc.GetTemplate(res)
		if err != nil {
			errs.Append(err)
			continue
		}
		for statement, _ := range tmpl.imports {
			allImports[statement] = struct{}{}
		}
	}
	if err := errs.ErrOrNil(); err != nil {
		return err
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

func (tc templatesCompiler) renderResource(out io.Writer, resource graph.Identifiable) error {
	// TODO: for now, assume a nice little tree
	errs := multierr.Error{}

	inputArgs := make(map[string]string)
	for fieldName, child := range getStructValues(resource) {
		childType := reflect.TypeOf(child)
		switch childType.Kind() {
		case reflect.String:
			inputArgs[fieldName] = quoteTsString(child.(string))
		case reflect.Struct, reflect.Pointer:
			if child, ok := child.(graph.Identifiable); ok {
				inputArgs[fieldName] = tc.getVarName(child)
			} else {
				errs.Append(errors.Errorf(`child struct of %v is not of a known type: %v`, resource, child))
			}
		default:
			errs.Append(errors.Errorf(`unrecognized input type for %v [%s]: %v`, resource, fieldName, child))
		}
	}

	varName := tc.getVarName(resource)
	tmpl, err := tc.GetTemplate(resource)
	if err != nil {
		return err
	}

	fmt.Fprintf(out, `const %s = `, varName)
	errs.Append(tmpl.RenderCreate(out, inputArgs))
	out.Write([]byte(";\n"))

	return errs.ErrOrNil()
}

func (tc templatesCompiler) getVarName(v graph.Identifiable) string {
	if name, alreadyResolved := tc.resourceVarNamesById[v.Id()]; alreadyResolved {
		return name
	}

	// Generate something like "lambdaFoo", where Lambda is the name of the struct and "foo" is the id
	desiredName := lowercaseFirst(structName(v)) + toUpperCamel(v.Id())
	resolvedName := desiredName
	for i := 1; ; i++ {
		_, varNameTaken := tc.resourceVarNames[resolvedName]
		if varNameTaken {
			resolvedName = fmt.Sprintf("%s_%d", desiredName, i)
		} else {
			break
		}
	}
	tc.resourceVarNames[resolvedName] = struct{}{}
	tc.resourceVarNamesById[v.Id()] = resolvedName
	return resolvedName
}

func (tc templatesCompiler) GetTemplate(v graph.Identifiable) (ResourceCreationTemplate, error) {
	typeName := structName(v)
	existing, ok := tc.templatesByStructName[typeName]
	if ok {
		return existing, nil
	}
	templateName := camelToSnake(typeName)
	contents, err := fs.ReadFile(tc.templates, templateName+`/factory.ts`)
	if err != nil {
		return ResourceCreationTemplate{}, err
	}
	template := ParseResourceCreationTemplate(contents)
	tc.templatesByStructName[typeName] = template
	return template, nil
}
