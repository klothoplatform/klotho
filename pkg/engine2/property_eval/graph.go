package property_eval

import (
	"errors"
	"fmt"
	"strings"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	"github.com/klothoplatform/klotho/pkg/graph_addons"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

type Graph graph.Graph[EvaluationKey, EvaluationVertex]

func newGraph() Graph {
	return graph.NewWithStore(
		EvaluationVertex.Key,
		graph_addons.NewMemoryStore[EvaluationKey, EvaluationVertex](),
		graph.Directed(),
		graph.PreventCycles(),
	)
}

func (eval *PropertyEval) AddResources(rs ...*construct.Resource) error {
	vs := make(verticesAndDeps)
	var errs error
	for _, res := range rs {
		tmpl, err := eval.Solution.KnowledgeBase().GetResourceTemplate(res.ID)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		rvs, err := eval.resourceVertices(res, tmpl)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		vs.AddAll(rvs)
	}
	if errs != nil {
		return errs
	}
	return eval.enqueue(vs)
}

func (eval *PropertyEval) RemoveResource(res construct.ResourceId) error {
	// TODO
	return nil
}

func (eval *PropertyEval) AddEdges(es ...construct.Edge) error {
	vs := make(verticesAndDeps)
	var errs error
	for _, e := range es {
		tmpl := eval.Solution.KnowledgeBase().GetEdgeTemplate(e.Source, e.Target)
		if tmpl == nil {
			continue
		}
		evs, err := eval.edgeVertices(e, tmpl)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		vs.AddAll(evs)
	}
	if errs != nil {
		return errs
	}
	return eval.enqueue(vs)
}

func UpdateEdgeId(e construct.SimpleEdge, oldId, newId construct.ResourceId) construct.SimpleEdge {
	switch {
	case e.Source == oldId:
		e.Source = newId
	case e.Target == oldId:
		e.Target = newId
	}
	return e
}

func (eval *PropertyEval) resourceVertices(
	res *construct.Resource,
	tmpl *knowledgebase.ResourceTemplate,
) (verticesAndDeps, error) {
	vs := make(verticesAndDeps)
	var errs error

	cfgCtx := solution_context.DynamicCtx(eval.Solution)

	queue := []knowledgebase.Properties{tmpl.Properties}
	var props knowledgebase.Properties
	for len(queue) > 0 {
		props, queue = queue[0], queue[1:]
		for _, prop := range props {
			vertex := &propertyVertex{
				Ref:       construct.PropertyRef{Resource: res.ID, Property: prop.Path},
				Template:  prop,
				EdgeRules: make(map[construct.SimpleEdge][]knowledgebase.OperationalRule),
			}

			for _, constr := range eval.Solution.Constraints().Resources {
				if constr.Target == res.ID && constr.Property == prop.Path {
					vertex.Constraints = append(vertex.Constraints, constr)
				}
			}

			err := vs.AddDependencies(cfgCtx, vertex)
			if err != nil {
				errs = errors.Join(errs, err)
				continue
			}

			if strings.HasPrefix(prop.Type, "list") || strings.HasPrefix(prop.Type, "set") {
				// Because lists/sets will start as empty, do not recurse into their sub-properties
				continue
			}
			if prop.Properties != nil {
				queue = append(queue, prop.Properties)
			}
		}
	}
	return vs, errs
}

func (eval *PropertyEval) edgeVertices(
	edge construct.Edge,
	tmpl *knowledgebase.EdgeTemplate,
) (verticesAndDeps, error) {
	vs := make(verticesAndDeps)
	var errs error

	cfgCtx := solution_context.DynamicCtx(eval.Solution)
	data := knowledgebase.DynamicValueData{Edge: &edge}
	resEdge := construct.SimpleEdge{Source: edge.Source, Target: edge.Target}

	opVertex := &edgeVertex{Edge: resEdge}

	vertices := make(map[construct.PropertyRef]*propertyVertex)
	for i, rule := range tmpl.OperationalRules {
		if len(rule.Steps) > 0 {
			opVertex.Rules = append(opVertex.Rules, knowledgebase.OperationalRule{
				// Split out only the steps, the configuration goes to the property it references
				If: rule.If, Steps: rule.Steps,
			})
		}

		for j, config := range rule.ConfigurationRules {
			var ref construct.PropertyRef
			err := cfgCtx.ExecuteDecode(config.Resource, data, &ref.Resource)
			if err != nil {
				errs = errors.Join(errs,
					fmt.Errorf("could not execute resource template for edge rule [%d][%d]: %w", i, j, err),
				)
			}
			err = cfgCtx.ExecuteDecode(config.Config.Field, data, &ref.Property)
			if err != nil {
				errs = errors.Join(errs,
					fmt.Errorf("could not execute config.field template for edge rule [%d][%d]: %w", i, j, err),
				)
			}

			vertex, ok := vertices[ref]
			if !ok {
				existing, err := eval.graph.Vertex(EvaluationKey{Ref: ref})
				switch {
				case errors.Is(err, graph.ErrVertexNotFound):
					vertex = &propertyVertex{Ref: ref, EdgeRules: make(map[construct.SimpleEdge][]knowledgebase.OperationalRule)}
				case err != nil:
					errs = errors.Join(errs, fmt.Errorf("could not attempt to get existing vertex for %s: %w", ref, err))
					continue
				default:
					if v, ok := existing.(*propertyVertex); ok {
						vertex = v
					} else {
						errs = errors.Join(errs, fmt.Errorf("existing vertex for %s is not a property vertex", ref))
						continue
					}
				}
				vertices[ref] = vertex
			}

			vertex.EdgeRules[resEdge] = append(vertex.EdgeRules[resEdge], knowledgebase.OperationalRule{
				If:                 rule.If,
				ConfigurationRules: []knowledgebase.ConfigurationRule{config},
			})
		}
	}
	if errs != nil {
		return nil, errs
	}

	if len(opVertex.Rules) > 0 {
		errs = errors.Join(errs, vs.AddDependencies(cfgCtx, opVertex))
	}

	// do this in a second pass so that edges config that reference the same property (rare, but possible)
	// will get batched before calling [depsForProp].
	for _, vertex := range vertices {
		errs = errors.Join(errs, vs.AddDependencies(cfgCtx, vertex))
	}

	return vs, errs
}
