package engine

import (
	"errors"
	"fmt"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/solution_context"
)

type MakeOperationalView solutionContext

func (view MakeOperationalView) Traits() *graph.Traits {
	return view.Dataflow.Traits()
}

func (view MakeOperationalView) AddVertex(value *construct.Resource, options ...func(*graph.VertexProperties)) error {
	err := view.raw().AddVertex(value, options...)
	if err != nil {
		return err
	}
	err = view.MakeResourcesOperational([]*construct.Resource{value})
	if err != nil {
		return err
	}

	// Look for any global edge constraints to the type of resource we are adding and enforce them
	for _, edgeConstraint := range view.constraints.Edges {
		if edgeConstraint.Target.Source.Name == "" &&
			edgeConstraint.Target.Source.QualifiedTypeName() == value.ID.QualifiedTypeName() {
			err = view.AddEdge(value.ID, edgeConstraint.Target.Target)
			if err != nil {
				return err
			}
		} else if edgeConstraint.Target.Target.Name == "" &&
			edgeConstraint.Target.Target.QualifiedTypeName() == value.ID.QualifiedTypeName() {
			err = view.AddEdge(edgeConstraint.Target.Source, value.ID)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (view MakeOperationalView) AddVerticesFrom(g construct.Graph) error {
	ordered, err := construct.ReverseTopologicalSort(g)
	if err != nil {
		return err
	}

	raw := view.raw()
	var errs error
	var resources []*construct.Resource

	add := func(id construct.ResourceId) {
		res, err := g.Vertex(id)
		if err != nil {
			errs = errors.Join(errs, err)
			return
		}
		errs = errors.Join(errs, raw.AddVertex(res))
		resources = append(resources, res)
	}

	for _, rid := range ordered {
		add(rid)
	}
	if errs != nil {
		return errs
	}
	return view.MakeResourcesOperational(resources)
}

func (view MakeOperationalView) raw() solution_context.RawAccessView {
	return solution_context.NewRawView(solutionContext(view))
}

func (view MakeOperationalView) MakeResourcesOperational(resources []*construct.Resource) error {
	return view.propertyEval.AddResources(resources...)
}

func (view MakeOperationalView) UpdateResourceID(oldId, newId construct.ResourceId) error {
	return view.propertyEval.UpdateId(oldId, newId)
}

func (view MakeOperationalView) Vertex(hash construct.ResourceId) (*construct.Resource, error) {
	return view.raw().Vertex(hash)
}

func (view MakeOperationalView) VertexWithProperties(hash construct.ResourceId) (*construct.Resource, graph.VertexProperties, error) {
	return view.raw().VertexWithProperties(hash)
}

func (view MakeOperationalView) RemoveVertex(hash construct.ResourceId) error {
	return errors.Join(
		view.raw().RemoveVertex(hash),
		view.propertyEval.RemoveResource(hash),
	)
}

func (view MakeOperationalView) AddEdge(source, target construct.ResourceId, options ...func(*graph.EdgeProperties)) (err error) {
	var dep construct.ResourceEdge
	var errs error
	dep.Source, err = view.Vertex(source)
	if err != nil {
		errs = errors.Join(errs, fmt.Errorf("no source found: %w", err))
	}
	dep.Target, err = view.Vertex(target)
	if err != nil {
		errs = errors.Join(errs, fmt.Errorf("no target found: %w", err))
	}
	if errs != nil {
		return fmt.Errorf("cannot add edge %s -> %s: %w", source, target, errs)
	}
	// Only add to view if there is an edge template, otherwise theres the potential to cause circular dependencies since path
	// solving will not have been run on the edge yet for intermediate resource
	if view.KB.GetEdgeTemplate(source, target) != nil {
		err = view.raw().AddEdge(source, target)
		if err != nil {
			return fmt.Errorf("cannot add edge %s -> %s: %w", source, target, err)
		}
	}
	// If both resources are imported we dont need to evaluate the edge vertex since we cannot modify the resources properties
	if dep.Source.Imported && dep.Target.Imported {
		return nil
	}
	return view.propertyEval.AddEdges(graph.Edge[construct.ResourceId]{Source: source, Target: target})
}

func (view MakeOperationalView) MakeEdgesOperational(edges []construct.Edge) error {
	return view.propertyEval.AddEdges(edges...)
}

func (view MakeOperationalView) AddEdgesFrom(g construct.Graph) error {
	edges, err := g.Edges()
	if err != nil {
		return err
	}
	var errs error
	for _, edge := range edges {
		errs = errors.Join(errs, view.AddEdge(edge.Source, edge.Target))
	}
	return errs
}

func (view MakeOperationalView) Edge(source, target construct.ResourceId) (construct.ResourceEdge, error) {
	return view.Dataflow.Edge(source, target)
}

func (view MakeOperationalView) Edges() ([]construct.Edge, error) {
	return view.Dataflow.Edges()
}

func (view MakeOperationalView) UpdateEdge(source, target construct.ResourceId, options ...func(properties *graph.EdgeProperties)) error {
	return view.raw().UpdateEdge(source, target, options...)
}

func (view MakeOperationalView) RemoveEdge(source, target construct.ResourceId) error {
	return errors.Join(
		view.raw().RemoveEdge(source, target),
		view.propertyEval.RemoveEdge(source, target),
	)
}

func (view MakeOperationalView) AdjacencyMap() (map[construct.ResourceId]map[construct.ResourceId]construct.Edge, error) {
	return view.Dataflow.AdjacencyMap()
}

func (view MakeOperationalView) PredecessorMap() (map[construct.ResourceId]map[construct.ResourceId]construct.Edge, error) {
	return view.Dataflow.PredecessorMap()
}

func (view MakeOperationalView) Clone() (construct.Graph, error) {
	return nil, errors.New("cannot clone an operational view")
}

func (view MakeOperationalView) Order() (int, error) {
	return view.Dataflow.Order()
}

func (view MakeOperationalView) Size() (int, error) {
	return view.Dataflow.Size()
}
