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
	"github.com/klothoplatform/klotho/pkg/graph_store"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/klothoplatform/klotho/pkg/set"
	"go.uber.org/zap"
)

type (
	PropertyVertex struct {
		Ref        construct.PropertyRef
		Path       construct.PropertyPath
		Template   knowledgebase.Property
		Constraint *constraints.ResourceConstraint
		Resource   *construct.Resource
	}

	Graph struct {
		graph.Graph[construct.PropertyRef, *PropertyVertex]

		done set.Set[construct.PropertyRef]
	}
)

func newGraph() Graph {
	return Graph{
		Graph: graph.NewWithStore(
			func(p *PropertyVertex) construct.PropertyRef { return p.Ref },
			graph_store.NewMemoryStore[construct.PropertyRef, *PropertyVertex](),
			graph.Directed(),
			graph.PreventCycles(),
		),
		done: make(set.Set[construct.PropertyRef]),
	}
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
		zap.S().Debugf("%s", source)
		for _, target := range targets {
			if target.String() == "aws:api_resource:test_api:test_api_lambda_test_app#PathPart" {
				zap.S().Debugf("configuring %s", target)
			}

			if _, err := g.Vertex(target); errors.Is(err, graph.ErrVertexNotFound) {
				tmpl, err := ctx.KnowledgeBase().GetResourceTemplate(target.Resource)
				if err != nil {
					return err
				}
				_, err = addResource(g, ctx, &construct.Resource{ID: target.Resource}, tmpl)
				if err != nil {
					return err
				}
			}

			zap.S().Debugf("  -> %s", target)
			if !g.done.Contains(target) {
				errs = errors.Join(errs, g.AddEdge(source, target))
			}
		}
	}
	if errs != nil {
		return fmt.Errorf("could not add edges to property eval graph: %w", errs)
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
			ref := construct.PropertyRef{Resource: res.ID, Property: prop.Path}
			vertex, err := g.Vertex(ref)
			if errors.Is(err, graph.ErrVertexNotFound) {
				vertex = &PropertyVertex{
					Ref:      construct.PropertyRef{Resource: res.ID, Property: prop.Path},
					Template: prop,
					Resource: res,
				}
			} else if err != nil {
				return nil, fmt.Errorf("could not get vertex for %s: %w", ref, err)
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
			errs = errors.Join(errs, construct.IgnoreExists(g.AddVertex(vertex)))
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
