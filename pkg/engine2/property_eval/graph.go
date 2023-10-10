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
			errs = errors.Join(errs, g.AddEdge(source, target))
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
	err := ctx.RawView().AddVertex(res)
	switch {
	case errors.Is(err, graph.ErrVertexAlreadyExists):
		// ignore
	case err != nil:
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
			if prop.Properties != nil {
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
	if ref, ok := prop.Template.DefaultValue.(construct.PropertyRef); ok {
		return []construct.PropertyRef{ref}, nil
	}
	vStr, ok := prop.Template.DefaultValue.(string)
	if !ok || !strings.Contains(vStr, "fieldValue") {
		// if it's not a PropertyRef or a template string that could resolve to a PropertyRef, it has no dependencies
		return nil, nil
	}
	propCtx := &fauxConfigContext{inner: cfgCtx, kb: cfgCtx.KB}
	tmpl, err := template.New(prop.Ref.String()).Funcs(propCtx.TemplateFunctions()).Parse(vStr)
	if err != nil {
		return nil, fmt.Errorf("could not parse template for property %s: %w", prop.Template.Path, err)
	}
	err = tmpl.Execute(
		io.Discard, // we don't care about the results, just the side effect of appending to propCtx.refs
		knowledgebase.DynamicValueData{Resource: prop.Ref.Resource},
	)
	if err != nil {
		return nil, fmt.Errorf("could not execute template for property %s: %w", prop.Template.Path, err)
	}
	return propCtx.refs, nil
}

// fauxConfigContext acts like a [knowledgebase.DynamicValueContext] but replaces the [FieldValue] function
// with one that just returns the zero value of the property type and records the property reference.
type fauxConfigContext struct {
	inner knowledgebase.DynamicValueContext
	kb    knowledgebase.TemplateKB
	refs  []construct.PropertyRef
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

	tmpl, err := ctx.kb.GetResourceTemplate(resId)
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
