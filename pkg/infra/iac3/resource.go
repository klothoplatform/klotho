package iac3

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
)

type templateInputArgs map[string]any

func (tc *TemplatesCompiler) RenderResource(out io.Writer, rid construct.ResourceId) error {
	resTmpl, err := tc.templates.ResourceTemplate(rid)
	if err != nil {
		return err
	}
	r, err := tc.graph.Vertex(rid)
	if err != nil {
		return err
	}
	inputs, err := tc.getInputArgs(r)
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

	return nil
}

func (tc *TemplatesCompiler) convertArg(arg any) (any, error) {
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
			// convert each element
			vars := make([]any, val.Len())
			for i := 0; i < val.Len(); i++ {
				v, err := tc.convertArg(val.Index(i).Interface())
				if err != nil {
					return nil, err
				}
				if jsonV, ok := v.(jsonValue); ok {
					vars[i] = jsonV.Raw
				} else {
					vars[i] = v
				}
			}
			return vars, nil
		case reflect.Map:
			// convert each element
			vars := make(map[string]any, val.Len())
			for _, k := range val.MapKeys() {
				v, err := tc.convertArg(val.MapIndex(k).Interface())
				if err != nil {
					return nil, err
				}
				if jsonV, ok := v.(jsonValue); ok {
					vars[k.String()] = jsonV.Raw
				} else {
					vars[k.String()] = v
				}
			}
			return jsonValue{Raw: vars}, nil
		default:
			return jsonValue{Raw: arg}, nil
		}
	}

}

func (tc *TemplatesCompiler) getInputArgs(r *construct.Resource) (templateInputArgs, error) {
	var errs error
	inputs := make(map[string]any, len(r.Properties)+len(globalVariables)+2) // +2 for Name and dependsOn

	for name, value := range r.Properties {
		argValue, err := tc.convertArg(value)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("could not convert arg %q: %w", name, err))
			continue
		}
		if argValue != nil {
			inputs[name] = argValue
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

		case "kubernetes:helm_chart":
			ao := tc.NewAppliedOutput(construct.PropertyRef{
				Resource: dep,
				// ready: pulumi.Output<pulumi.CustomResource[]>
				Property: "ready",
			}, "")
			applied = append(applied, ao)
			dependsOn = append(dependsOn, "..."+ao.Name)

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
			return json.NewEncoder(w).Encode(dependsOn)
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
