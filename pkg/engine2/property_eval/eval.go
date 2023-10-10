package property_eval

import (
	"errors"
	"fmt"
	"sort"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/operational_rule"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

func SetupResources(solCtx solution_context.SolutionContext, resources []*construct.Resource) error {
	return setupInner(solCtx, newGraph(), resources, nil)
}

// setupInner is a recursive function that resolves properties in the graph.
func setupInner(
	solCtx solution_context.SolutionContext,
	g Graph,
	resources []*construct.Resource,
	edges []construct.Edge,
) error {
	err := AddResources(g, solCtx, resources)
	if err != nil {
		return err
	}

	for {
		prop, err := popProperty(g)
		if err != nil {
			return err
		}
		if prop == nil {
			break
		}
		ctx := solCtx.With("resource", prop.Ref.Resource).With("property", prop.Ref)
		cfgData := knowledgebase.DynamicValueData{Resource: prop.Ref.Resource}

		res, err := ctx.RawView().Vertex(prop.Ref.Resource)
		if err != nil {
			return err
		}

		if prop.Constraint != nil {
			err = solution_context.ApplyConfigureConstraint(ctx, res, *prop.Constraint)
		} else if prop.Template.DefaultValue != nil {
			err = solution_context.ConfigureResource(
				ctx,
				res,
				knowledgebase.Configuration{Field: prop.Ref.Property, Value: prop.Template.DefaultValue},
				cfgData,
				"set",
			)
		}
		if err != nil {
			return fmt.Errorf("could not configure %s: %w", prop.Ref, err)
		}

		if prop.Template.OperationalRule == nil {
			continue
		}

		opCtx := operational_rule.OperationalRuleContext{
			Solution: ctx,
			Property: &prop.Template,
			Data:     cfgData,
		}
		result, err := opCtx.HandleOperationalRule(*prop.Template.OperationalRule)
		if err != nil {
			return fmt.Errorf("could not handle operational rule for %s: %w", prop.Ref, err)
		}

		err = AddResources(g, ctx, result.CreatedResources)
		if err != nil {
			return fmt.Errorf("could not add operational resources for %s: %w", prop.Ref, err)
		}
		edges = append(edges, result.AddedDependencies...)
	}
	if len(edges) == 0 {
		return nil
	}

	var nextResources []*construct.Resource
	var nextEdges []construct.Edge
	var errs error
	for _, edge := range edges {
		addRes, addDep, err := solCtx.OperationalView().MakeEdgeOperational(edge.Source, edge.Target)
		errs = errors.Join(errs, err)
		nextResources = append(nextResources, addRes...)
		nextEdges = append(nextEdges, addDep...)
	}
	if errs != nil {
		return errs
	}

	return setupInner(solCtx, g, nextResources, nextEdges)
}

func popProperty(g Graph) (*PropertyVertex, error) {
	// Use adjacency map so we can easily find properties with no dependencies.
	// This is effectively the same as a reverse topological sort (through sequential calls to [popProperty]).
	adj, err := g.AdjacencyMap()
	if err != nil {
		return nil, err
	}
	if len(adj) == 0 {
		return nil, nil
	}

	var candidates []construct.PropertyRef
	for ref, edges := range adj {
		if len(edges) == 0 {
			candidates = append(candidates, ref)
		}
	}

	if len(candidates) == 0 {
		// This should never happen unless the graph contains cycles, which should never happen...
		return nil, fmt.Errorf("no candidates with no unresolved dependencies found in graph (cycles detected)")
	}

	// Make the process deterministic
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].String() < candidates[j].String()
	})

	ref := candidates[0]
	prop, err := g.Vertex(ref)
	if err != nil {
		return nil, fmt.Errorf("could not get vertex %s from graph: %w", ref, err)
	}

	// Remove predecessor edges, since this property is about to be resolved
	// and it is also necessary before RemoveVertex.
	pred, err := g.PredecessorMap()
	if err != nil {
		return nil, err
	}
	var errs error
	for _, edge := range pred[ref] {
		errs = errors.Join(errs, g.RemoveEdge(edge.Source, edge.Target))
	}
	if errs != nil {
		return nil, errs
	}
	err = g.RemoveVertex(ref)
	if err != nil {
		return nil, err
	}
	return prop, nil
}
