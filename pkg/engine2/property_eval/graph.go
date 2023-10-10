package property_eval

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"strings"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/constraints"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

type (
	PropertyVertex struct {
		Ref        construct.PropertyRef
		Path       construct.PropertyPath
		Template   knowledgebase.Property
		Constraint *constraints.ResourceConstraint
	}

	Graph = graph.Graph[construct.PropertyRef, *PropertyVertex]
)

func newGraph() Graph {
	return graph.New(
		func(p *PropertyVertex) construct.PropertyRef { return p.Ref },
		graph.Directed(),
		graph.Acyclic(),
		graph.PreventCycles(),
	)
}

func AddResources(
	g Graph,
	ctx solution_context.SolutionContext,
	resources []*construct.Resource,
) error {
	deps := make(map[construct.PropertyRef][]construct.PropertyRef)
	var errs error
	for _, res := range resources {
		tmpl, err := ctx.KnowledgeBase().GetResourceTemplate(res.ID)
		if err != nil {
			return err
		}
		rdeps, err := addResource(g, ctx, res, tmpl)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		for k, v := range rdeps {
			deps[k] = v
		}
	}
	if errs != nil {
		return errs
	}

	for source, targets := range deps {
		for _, target := range targets {
			errs = errors.Join(errs, construct.IgnoreExists(g.AddEdge(source, target)))
		}
	}
	if errs != nil {
		return errs
	}
	return nil
}

// addResource adds the resources' properties to the graph, returning all the dependencies. The caller must
// add the edges to the graph when they are ready - this is so that in [InitGraph], we can add all the
// resources to the graph before adding the edges.
func addResource(
	g Graph,
	ctx solution_context.SolutionContext,
	res *construct.Resource,
	tmpl *knowledgebase.ResourceTemplate,
) (map[construct.PropertyRef][]construct.PropertyRef, error) {
	err := construct.IgnoreExists(ctx.RawView().AddVertex(res))
	if err != nil {
		return nil, fmt.Errorf("could not add resource %s to graph: %w", res.ID, err)
	}

	cfgCtx := solution_context.DynamicCtx(ctx)
	queue := []knowledgebase.Properties{tmpl.Properties}
	var props knowledgebase.Properties
	var errs error
	deps := make(map[construct.PropertyRef][]construct.PropertyRef)
	for len(queue) > 0 {
		props, queue = queue[0], queue[1:]

		for _, prop := range props {
			vertex := &PropertyVertex{
				Ref:      construct.PropertyRef{Resource: res.ID, Property: prop.Path},
				Template: prop,
			}
			if _, err := g.Vertex(vertex.Ref); err == nil {
				continue
			}
			path, err := res.PropertyPath(prop.Path)
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf("could not get property path %s: %w", prop.Path, err))
				continue
			}
			vertex.Path = path
			for _, constr := range ctx.Constraints().Resources {
				if constr.Target == res.ID && constr.Property == prop.Path {
					vertex.Constraint = &constr
					break
				}
			}
			errs = errors.Join(errs, g.AddVertex(vertex))
			if prop.Properties != nil && !strings.HasPrefix(prop.Type, "list") {
				// Because lists will start as empty, do not recurse into their sub-properties
				queue = append(queue, prop.Properties)
			}
			vdeps, err := depsForProp(cfgCtx, vertex)
			if err != nil {
				errs = errors.Join(errs, err)
				continue
			}
			deps[vertex.Ref] = vdeps
		}
	}
	return deps, errs
}

func depsForProp(cfgCtx knowledgebase.DynamicValueContext, prop *PropertyVertex) ([]construct.PropertyRef, error) {
	if prop.Constraint != nil {
		// for now, constraints can't be templated so won't have dependencies
		return nil, nil
	}
	propCtx := &fauxConfigContext{inner: cfgCtx}

	if err := propCtx.Execute(prop.Template.DefaultValue, prop.Ref); err != nil {
		return nil, fmt.Errorf("could not execute template for %s: %w", prop.Ref, err)
	}

	if opRule := prop.Template.OperationalRule; opRule != nil {
		var errs error
		errs = errors.Join(errs, propCtx.Execute(opRule.If, prop.Ref))
		for _, rule := range opRule.ConfigurationRules {
			errs = errors.Join(errs, propCtx.Execute(rule.Config.Value, prop.Ref))
		}
		for _, step := range opRule.Steps {
			for _, stepRes := range step.Resources {
				errs = errors.Join(errs, propCtx.Execute(stepRes, prop.Ref))
			}
		}
		if errs != nil {
			return nil, fmt.Errorf("could not execute templates for %s: %w", prop.Ref, errs)
		}
	}

	return propCtx.refs, nil
}

// fauxConfigContext acts like a [knowledgebase.DynamicValueContext] but replaces the [FieldValue] function
// with one that just returns the zero value of the property type and records the property reference.
type fauxConfigContext struct {
	inner knowledgebase.DynamicValueContext
	refs  []construct.PropertyRef
}

func (ctx *fauxConfigContext) Execute(v any, ref construct.PropertyRef) error {
	if ref, ok := v.(construct.PropertyRef); ok {
		ctx.refs = append(ctx.refs, ref)
	}
	vStr, ok := v.(string)
	if !ok || !strings.Contains(vStr, "fieldValue") {
		return nil
	}
	tmpl, err := template.New(ref.String()).Funcs(ctx.TemplateFunctions()).Parse(vStr)
	if err != nil {
		return fmt.Errorf("could not parse template %w", err)
	}

	// Ignore execution errors for when the zero value is invalid due to other assumptions
	// if there is an error with the template, this will be caught later when actually processing it.
	_ = tmpl.Execute(
		io.Discard, // we don't care about the results, just the side effect of appending to propCtx.refs
		knowledgebase.DynamicValueData{Resource: ref.Resource},
	)

	return nil
}

func (ctx *fauxConfigContext) TemplateFunctions() template.FuncMap {
	funcs := ctx.inner.TemplateFunctions()
	funcs["fieldValue"] = ctx.FieldValue
	return funcs
}

func (ctx *fauxConfigContext) FieldValue(field string, resource any) (any, error) {
	resId, err := knowledgebase.TemplateArgToRID(resource)
	if err != nil {
		return "", err
	}
	ctx.refs = append(ctx.refs, construct.PropertyRef{Resource: resId, Property: field})

	tmpl, err := ctx.inner.KB.GetResourceTemplate(resId)
	if err != nil {
		return "", err
	}

	return emptyValue(tmpl, field)
}

func emptyValue(tmpl *knowledgebase.ResourceTemplate, property string) (any, error) {
	prop := tmpl.GetProperty(property)
	if prop == nil {
		return nil, fmt.Errorf("could not find property %s on template %s", property, tmpl.Id())
	}
	ptype, err := prop.PropertyType()
	if err != nil {
		return nil, fmt.Errorf("could not get property type for property %s on template %s: %w", property, tmpl.Id(), err)
	}
	return ptype.ZeroValue(), nil
}
