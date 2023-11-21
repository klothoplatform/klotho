package iac3

import (
	"errors"
	"fmt"
	"io"
	"path"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/iancoleman/strcase"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/set"
)

type templateInputArgs map[string]any

var validIdentifierPattern = regexp.MustCompile(`^[a-zA-Z_$][a-zA-Z_$0-9]*$`)

func (tc *TemplatesCompiler) RenderResource(out io.Writer, rid construct.ResourceId) error {
	resTmpl, err := tc.ResourceTemplate(rid)
	if err != nil {
		return err
	}
	r, err := tc.graph.Vertex(rid)
	if err != nil {
		return err
	}
	inputs, err := tc.getInputArgs(r, resTmpl)
	if err != nil {
		return err
	}

	if resTmpl.OutputType != "void" {
		_, err = fmt.Fprintf(out, "const %s = ", tc.vars[rid])
		if err != nil {
			return err
		}
	}
	err = resTmpl.Template.Execute(out, inputs)
	if err != nil {
		return fmt.Errorf("could not render resource %s: %w", rid, err)
	}

	exportData := PropertyTemplateData{
		Resource: rid,
		Object:   tc.vars[rid],
		Input:    inputs,
	}
	var errs error
	for export, tmpl := range resTmpl.Exports {
		_, err = fmt.Fprintf(out, "\nexport const %s_%s = ", tc.vars[rid], export)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("could not render export name %s: %w", export, err))
			continue
		}
		err = tmpl.Execute(out, exportData)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("could not render export value %s: %w", export, err))
			continue
		}
	}
	if errs != nil {
		return errs
	}

	return nil
}

func (tc *TemplatesCompiler) convertArg(arg any, templateArg *Arg) (any, error) {

	switch arg := arg.(type) {
	case construct.ResourceId:
		return tc.vars[arg], nil

	case construct.PropertyRef:
		return tc.PropertyRefValue(arg)

	case string:
		// use templateString to quote the string value
		return templateString(arg), nil

	case bool, int, float64:
		// safe to use as-is
		return arg, nil

	case nil:
		// don't add to inputs
		return nil, nil

	default:
		switch val := reflect.ValueOf(arg); val.Kind() {
		case reflect.Slice, reflect.Array:
			list := make(TsList, 0, val.Len())
			for i := 0; i < val.Len(); i++ {
				if !val.Index(i).IsValid() || val.Index(i).IsNil() {
					continue
				}
				output, err := tc.convertArg(val.Index(i).Interface(), templateArg)
				if err != nil {
					return "", err
				}
				list = append(list, output)
			}
			return list, nil
		case reflect.Map:
			TsMap := make(TsMap, val.Len())
			for _, key := range val.MapKeys() {
				if !val.MapIndex(key).IsValid() || val.MapIndex(key).IsZero() {
					continue
				}
				keyStr, found := key.Interface().(string)
				if !found {
					return "", fmt.Errorf("map key is not a string")
				}
				keyResult := strcase.ToLowerCamel(keyStr)
				if templateArg != nil && templateArg.Wrapper == string(CamelCaseWrapper) {
					keyResult = strcase.ToCamel(keyStr)
				} else if templateArg != nil && templateArg.Wrapper == string(ModelCaseWrapper) {
					if validIdentifierPattern.MatchString(keyStr) {
						keyResult = keyStr
					} else {
						keyResult = fmt.Sprintf(`"%s"`, keyStr)
					}
				}
				output, err := tc.convertArg(val.MapIndex(key).Interface(), templateArg)
				if err != nil {
					return "", err
				}
				TsMap[keyResult] = output
			}
			return TsMap, nil
		case reflect.Struct:
			if hashset, ok := val.Interface().(set.HashedSet[string, any]); ok {
				return tc.convertArg(hashset.ToSlice(), templateArg)
			}
			fallthrough
		default:
			return jsonValue{Raw: arg}, nil
		}
	}
}

func (tc *TemplatesCompiler) getInputArgs(r *construct.Resource, template *ResourceTemplate) (templateInputArgs, error) {
	var errs error
	inputs := make(map[string]any, len(r.Properties)+len(globalVariables)+2) // +2 for Name and dependsOn
	selfReferences := make(map[string]construct.PropertyRef)

	for name, value := range r.Properties {
		templateArg := template.Args[name]
		var argValue any
		var err error
		if templateArg.Wrapper == string(TemplateWrapper) {
			argValue, err = tc.useNestedTemplate(template, value, templateArg)
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf("could not use nested template for arg %q: %w", name, err))
				continue
			}
		} else if ref, ok := value.(construct.PropertyRef); ok && ref.Resource == r.ID {
			selfReferences[name] = ref
		} else {
			argValue, err = tc.convertArg(value, &templateArg)
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf("could not convert arg %q: %w", name, err))
				continue
			}
		}

		if argValue != nil {
			inputs[name] = argValue
		}
	}

	for name, value := range selfReferences {
		if mapping, ok := template.PropertyTemplates[value.Property]; ok {
			data := PropertyTemplateData{
				Resource: r.ID,
				Object:   tc.vars[r.ID],
				Input:    inputs,
			}
			result, err := executeToString(mapping, data)
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf("could not execute self-reference %q: %w", name, err))
				continue
			}
			inputs[name] = result
		} else {
			errs = errors.Join(errs, fmt.Errorf("could not find mapping for self-reference %q", name))
		}
	}

	if errs != nil {
		return templateInputArgs{}, errs
	}

	downstream, err := construct.DirectDownstreamDependencies(tc.graph, r.ID)
	if err != nil {
		return templateInputArgs{}, err
	}
	var dependsOn []string
	var applied appliedOutputs
	for _, dep := range downstream {
		switch dep.QualifiedTypeName() {
		case "aws:region", "aws:availability_zone", "aws:account_id":
			continue

		case "kubernetes:manifest", "kubernetes:kustomize_directory":
			ao := tc.NewAppliedOutput(construct.PropertyRef{
				Resource: dep,
				// resources: pulumi.Output<{
				// 		[key: string]: pulumi.CustomResource;
				// }>
				Property: "resources",
			}, "")
			applied = append(applied, ao)
			dependsOn = append(dependsOn, fmt.Sprintf("...Object.values(%s)", ao.Name))

		default:
			dependsOn = append(dependsOn, tc.vars[dep])
		}
	}
	sort.Strings(dependsOn)
	if len(applied) > 0 {
		buf := getBuffer()
		defer releaseBuffer(buf)
		err = applied.Render(buf, func(w io.Writer) error {
			_, err := w.Write([]byte("["))
			if err != nil {
				return err
			}
			for i, dep := range dependsOn {
				_, err = w.Write([]byte(dep))
				if err != nil {
					return err
				}
				if i < len(dependsOn)-1 {
					_, err = w.Write([]byte(", "))
					if err != nil {
						return err
					}
				}
			}
			_, err = w.Write([]byte("]"))
			if err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return templateInputArgs{}, err
		}
		inputs["dependsOn"] = buf.String()
	} else {
		inputs["dependsOn"] = "[" + strings.Join(dependsOn, ", ") + "]"
	}

	inputs["Name"] = templateString(r.ID.Name)

	for g := range globalVariables {
		inputs[g] = g
	}

	return inputs, nil
}

func (tc *TemplatesCompiler) useNestedTemplate(resTmpl *ResourceTemplate, val any, arg Arg) (string, error) {

	var contents []byte
	var err error

	nestedTemplatePath := path.Join(resTmpl.Path, strcase.ToSnake(arg.Name)+".ts.tmpl")

	f, err := tc.templates.fs.Open(nestedTemplatePath)
	if err != nil {
		return "", fmt.Errorf("could not find template for %s: %w", nestedTemplatePath, err)
	}
	contents, err = io.ReadAll(f)
	if err != nil {
		return "", fmt.Errorf("could not read template for %s: %w", nestedTemplatePath, err)
	}
	if len(contents) == 0 {
		return "", fmt.Errorf("no contents in template for %s: %w", nestedTemplatePath, err)
	}

	tmpl, err := template.New(nestedTemplatePath).Funcs(template.FuncMap{
		"modelCase":      tc.modelCase,
		"lowerCamelCase": tc.lowerCamelCase,
		"camelCase":      tc.camelCase,
		"getVar": func(id construct.ResourceId) string {
			return tc.vars[id]
		},
	}).Parse(string(contents))
	if err != nil {
		return "", fmt.Errorf("could not parse template for %s: %w", nestedTemplatePath, err)
	}
	result := getBuffer()
	err = tmpl.Execute(result, val)
	if err != nil {
		return "", fmt.Errorf("could not execute template for %s: %w", nestedTemplatePath, err)
	}
	return result.String(), nil
}

func (tc *TemplatesCompiler) modelCase(val any) (any, error) {
	return tc.convertArg(val, &Arg{Wrapper: string(ModelCaseWrapper)})
}

func (tc *TemplatesCompiler) lowerCamelCase(val any) (any, error) {
	return tc.convertArg(val, &Arg{Wrapper: string(LowerCamelCaseWrapper)})
}

func (tc *TemplatesCompiler) camelCase(val any) (any, error) {
	return tc.convertArg(val, &Arg{Wrapper: string(CamelCaseWrapper)})
}
