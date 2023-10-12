package property_eval

import (
	"errors"
	"fmt"
	"sort"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/operational_rule"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/klothoplatform/klotho/pkg/set"
	"go.uber.org/zap"
)

func SetupResources(
	solCtx solution_context.SolutionContext,
	resources []*construct.Resource,
) (construct.ResourceIdChangeResults, error) {
	return setupInner(solCtx, newGraph(), resources, nil)
}

// setupInner is a recursive function that resolves properties in the graph.
func setupInner(
	solCtx solution_context.SolutionContext,
	g Graph,
	resources []*construct.Resource,
	edges []construct.Edge,
) (construct.ResourceIdChangeResults, error) {
	idChanges := make(construct.ResourceIdChangeResults)
	return setupInnerRecurse(solCtx, g, resources, edges, idChanges)
}

func setupInnerRecurse(
	solCtx solution_context.SolutionContext,
	g Graph,
	resources []*construct.Resource,
	edges []construct.Edge,
	idChanges construct.ResourceIdChangeResults,
) (construct.ResourceIdChangeResults, error) {
	err := AddResources(g, solCtx, resources)
	if err != nil {
		return idChanges, err
	}

	for {
		prop, err := popProperty(g)
		if err != nil {
			return idChanges, err
		}
		if prop == nil {
			break
		}
		zap.S().Debugf("configuring %s", prop.Ref)
		g.done.Add(prop.Ref)
		ctx := solCtx.With("resource", prop.Ref.Resource).With("property", prop.Ref)
		cfgData := knowledgebase.DynamicValueData{Resource: prop.Ref.Resource}
		res, err := ctx.RawView().Vertex(prop.Ref.Resource)
		if err != nil {
			return idChanges, fmt.Errorf("could not get resource %s, while setting up inner properties: %w", prop.Ref.Resource, err)
		}

		if prop.Constraint != nil {
			err = solution_context.ApplyConfigureConstraint(ctx, res, *prop.Constraint)
		} else if property, _ := prop.Resource.GetProperty(prop.Template.Path); property == nil && prop.Template.DefaultValue != nil {
			err = solution_context.ConfigureResource(
				ctx,
				res,
				knowledgebase.Configuration{Field: prop.Ref.Property, Value: prop.Template.DefaultValue},
				cfgData,
				"set",
			)
		}
		// set this here in case defaults set a value which changes the namespace
		idChanges[prop.Ref.Resource] = res.ID
		if err != nil {
			return idChanges, fmt.Errorf("could not configure %s: %w", prop.Ref, err)
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
			return idChanges, fmt.Errorf("could not handle operational rule for %s: %w", prop.Ref, err)
		}
		// Add our id changes if the property ref resource is in the initial resources list
		idChanges[prop.Ref.Resource] = res.ID
		err = replaceResourceId(g, prop.Ref.Resource, prop.Resource.ID)
		if err != nil {
			return idChanges, fmt.Errorf("could not replace resource id %s with %s when namespace changed: %w", prop.Ref.Resource, prop.Resource.ID, err)
		}
		err = AddResources(g, ctx, result.CreatedResources)
		if err != nil {
			return idChanges, fmt.Errorf("could not add operational resources for %s: %w", prop.Ref, err)
		}
		edges = append(edges, result.AddedDependencies...)
	}
	if len(edges) == 0 {
		return idChanges, nil
	}

	var nextResources []*construct.Resource
	var nextEdges []construct.Edge
	var errs error
	for _, edge := range edges {
		if src, found := idChanges[edge.Source]; found {
			edge.Source = src
		}
		if tgt, found := idChanges[edge.Target]; found {
			edge.Target = tgt
		}
		addRes, addDep, err := solCtx.OperationalView().MakeEdgeOperational(edge.Source, edge.Target)
		errs = errors.Join(errs, err)
		nextResources = append(nextResources, addRes...)
		nextEdges = append(nextEdges, addDep...)
	}
	if errs != nil {
		return idChanges, errs
	}

	return setupInnerRecurse(solCtx, g, nextResources, nextEdges, idChanges)
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

// ReplaceResource replaces the resources identified by `oldId` with `newRes` in the graph and in any property
// references (as [ResourceId] or [PropertyRef]) of the old ID to the new ID in any resource that depends on or is
// depended on by the resource.
func replaceResourceId(g Graph, oldId, newRes construct.ResourceId) error {
	// Short circuit if the resource ID hasn't changed.
	if newRes.Matches(oldId) {
		return nil
	}
	refs, err := graph.TopologicalSort(g)
	if err != nil {
		return err
	}
	for _, ref := range refs {
		r, props, err := g.VertexWithProperties(ref)
		if err != nil {
			return err
		}
		// Make sure that the constraint is substituted no matter what if it is for the old id
		if r.Constraint != nil && r.Constraint.Target.Matches(oldId) {
			r.Constraint.Target = newRes
		}
		if ref.Resource.Matches(oldId) {
			r.Ref.Resource = newRes
			err = g.AddVertex(r, construct.CopyVertexProps(props))
			if err != nil {
				return err
			}

			neighbors := make(set.Set[construct.PropertyRef])
			adj, err := g.AdjacencyMap()
			if err != nil {
				return err
			}
			for _, edge := range adj[ref] {
				err = errors.Join(
					err,
					g.AddEdge(r.Ref, edge.Target, construct.CopyEdgeProps(edge.Properties)),
					g.RemoveEdge(edge.Source, edge.Target),
				)
				neighbors.Add(edge.Target)
			}
			if err != nil {
				return err
			}

			pred, err := g.PredecessorMap()
			if err != nil {
				return err
			}
			for _, edge := range pred[ref] {
				err = errors.Join(
					err,
					g.AddEdge(edge.Source, r.Ref, construct.CopyEdgeProps(edge.Properties)),
					g.RemoveEdge(edge.Source, edge.Target),
				)
				neighbors.Add(edge.Source)
			}
			if err != nil {
				return err
			}

			if err := g.RemoveVertex(ref); err != nil {
				return err
			}

		}
	}
	return nil
}
