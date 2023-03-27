package iac2

import (
	"bytes"
	"embed"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"reflect"
	"sort"
	"strings"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/graph"
	"github.com/klothoplatform/klotho/pkg/lang/javascript"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type (
	// TemplatesCompiler renders a graph of [core.Resource] nodes by combining each one with its corresponding
	// ResourceCreationTemplate
	TemplatesCompiler struct {
		// templates is the fs.FS where we read all of our `<struct>/factory.ts` files
		templates fs.FS
		// resourceGraph is the graph of resources to render
		resourceGraph *graph.Directed[core.Resource] // TODO make this be a core.ResourceGraph, and un-expose that struct's Underlying
		// templatesByStructName is a cache from struct name (e.g. "CloudwatchLogs") to the template for that struct.
		templatesByStructName map[string]ResourceCreationTemplate
		// resourceVarNames is a set of all variable names
		resourceVarNames map[string]struct{}
		// resourceVarNamesById is a map from resource id to the variable name for that resource
		resourceVarNamesById map[string]string
	}
)

var (
	//go:embed templates/*/factory.ts templates/*/package.json
	standardTemplates embed.FS
)

func CreateTemplatesCompiler(resources *graph.Directed[core.Resource]) *TemplatesCompiler {
	subTemplates, err := fs.Sub(standardTemplates, "templates")
	if err != nil {
		panic(err) // unexpected, since standardTemplates is statically built into klotho
	}
	return &TemplatesCompiler{
		templates:             subTemplates,
		resourceGraph:         resources,
		templatesByStructName: make(map[string]ResourceCreationTemplate),
		resourceVarNames:      make(map[string]struct{}),
		resourceVarNamesById:  make(map[string]string),
	}
}

func (tc TemplatesCompiler) RenderBody(out io.Writer) error {
	errs := multierr.Error{}
	vertexIds, err := tc.resourceGraph.VertexIdsInTopologicalOrder()
	if err != nil {
		return err
	}
	for i, id := range vertexIds {
		resource := tc.resourceGraph.GetVertex(id)
		err := tc.renderResource(out, resource)
		errs.Append(err)
		if i < len(vertexIds)-1 {
			_, err = out.Write([]byte("\n\n"))
			if err != nil {
				return err
			}
		}

	}
	return errs.ErrOrNil()
}

func (tc TemplatesCompiler) RenderImports(out io.Writer) error {
	errs := multierr.Error{}

	allImports := make(map[string]struct{})
	for _, res := range tc.resourceGraph.GetAllVertices() {
		tmpl, err := tc.GetTemplate(res)
		if err != nil {
			errs.Append(err)
			continue
		}
		for statement := range tmpl.Imports {
			allImports[statement] = struct{}{}
		}
	}
	if err := errs.ErrOrNil(); err != nil {
		return err
	}

	sortedImports := make([]string, 0, len(allImports))
	for statement := range allImports {
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

func (tc TemplatesCompiler) RenderPackageJSON() (*javascript.NodePackageJson, error) {
	errs := multierr.Error{}
	mainPJson := javascript.NodePackageJson{}
	for _, res := range tc.resourceGraph.GetAllVertices() {
		pJson, err := tc.GetPackageJSON(res)
		if err != nil {
			errs.Append(err)
			continue
		}
		mainPJson.Merge(&pJson)
	}
	if err := errs.ErrOrNil(); err != nil {
		return &mainPJson, err
	}
	return &mainPJson, nil
}

func (tc TemplatesCompiler) renderResource(out io.Writer, resource core.Resource) error {

	tmpl, err := tc.GetTemplate(resource)
	if err != nil {
		return err
	}

	errs := multierr.Error{}

	resourceVal := reflect.ValueOf(resource)
	for resourceVal.Kind() == reflect.Pointer {
		resourceVal = resourceVal.Elem()
	}
	inputArgs := make(map[string]string)
	var zeroValue reflect.Value
	for fieldName := range tmpl.InputTypes {
		childVal := resourceVal.FieldByName(fieldName)
		if childVal == zeroValue {
			zap.S().Warnf(
				`Klotho compiler error: no field %s.%s while rendering typescript template`,
				resourceVal.Type().Name(),
				fieldName)
			continue
		}
		resolvedValue := tc.resolveStructInput(childVal)
		if resolvedValue == "" {
			errs.Append(errors.Errorf(`child struct of %v is not of a known type: %v`, resource, childVal.Interface()))
		} else {
			inputArgs[fieldName] = resolvedValue
		}

	}

	varName := tc.getVarName(resource)

	fmt.Fprintf(out, `const %s = `, varName)
	errs.Append(tmpl.RenderCreate(out, inputArgs))
	_, err = out.Write([]byte(";"))
	if err != nil {
		return err
	}

	return errs.ErrOrNil()
}

// resolveStructInput translates a value to a form suitable to inject into the typescript as an input to a function.
func (tc TemplatesCompiler) resolveStructInput(childVal reflect.Value) string {
	switch childVal.Kind() {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%v", childVal.Interface())
	case reflect.String:
		return quoteTsString(childVal.Interface().(string))
	case reflect.Struct, reflect.Pointer:
		if childVal.Kind() == reflect.Pointer && childVal.IsNil() {
			return "null"
		}
		if typedChild, ok := childVal.Interface().(graph.Identifiable); ok {
			return tc.getVarName(typedChild)
		} else {
			return ""
		}
	case reflect.Array, reflect.Slice:
		sliceLen := childVal.Len()

		buf := strings.Builder{}
		buf.WriteRune('[')
		for i := 0; i < sliceLen; i++ {
			buf.WriteString(tc.resolveStructInput(childVal.Index(i)))
			if i < (sliceLen - 1) {
				buf.WriteRune(',')
			}
		}
		buf.WriteRune(']')
		return buf.String()
	case reflect.Map:
		mapLen := childVal.Len()

		buf := strings.Builder{}
		buf.WriteRune('{')
		for i, key := range childVal.MapKeys() {
			buf.WriteString(tc.resolveStructInput(key))
			buf.WriteRune(':')
			buf.WriteString(tc.resolveStructInput(childVal.MapIndex(key)))
			if i < (mapLen - 1) {
				buf.WriteRune(',')
			}
		}
		buf.WriteRune('}')

		return buf.String()
	}
	return ""
}

// getVarName gets a unique but nice-looking variable for the given item.
//
// It does this by first calculating an ideal variable name, which is a camel-cased ${structName}${Id}. For example, if
// you had an object CoolResource{id: "foo-bar"}, the ideal variable name is coolResourceFooBar.
//
// If that ideal variable name hasn't been used yet, this function returns it. If it has been used, we append `_${i}` to
// it, where ${i} is the lowest positive integer that would give us a new, unique variable name. This isn't expected
// to happen often, if at all, since ids are globally unique.
func (tc TemplatesCompiler) getVarName(v graph.Identifiable) string {
	if name, alreadyResolved := tc.resourceVarNamesById[v.Id()]; alreadyResolved {
		return name
	}

	// Generate something like "lambdaFoo", where Lambda is the name of the struct and "foo" is the id
	desiredName := lowercaseFirst(toUpperCamel(v.Id()))
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

func (tc TemplatesCompiler) GetTemplate(v graph.Identifiable) (ResourceCreationTemplate, error) {
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
	template := ParseResourceCreationTemplate(typeName, contents)
	tc.templatesByStructName[typeName] = template
	return template, nil
}

func (tc TemplatesCompiler) GetPackageJSON(v graph.Identifiable) (javascript.NodePackageJson, error) {
	packageContent := javascript.NodePackageJson{}
	typeName := structName(v)
	templateName := camelToSnake(typeName)
	contents, err := fs.ReadFile(tc.templates, templateName+`/package.json`)
	if err != nil {
		return *packageContent.Clone(), err
	}
	err = json.NewDecoder(bytes.NewReader(contents)).Decode(&packageContent)
	if err != nil {
		return *packageContent.Clone(), err
	}
	return *packageContent.Clone(), nil
}
