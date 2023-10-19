package engine2

import (
	"errors"
	"fmt"
	"strings"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/constraints"
	"github.com/klothoplatform/klotho/pkg/engine2/path_selection"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	"go.uber.org/zap"
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
	return view.MakeResourcesOperational([]*construct.Resource{value})
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

	var data path_selection.EdgeData
	for _, constr := range view.constraints.Edges {
		if !constr.Target.Source.Matches(source) || !constr.Target.Target.Matches(target) {
			continue
		}

		switch constr.Operator {
		case constraints.MustContainConstraintOperator:
			data.Constraint.NodeMustExist = append(data.Constraint.NodeMustExist, constr.Node)

		case constraints.MustNotContainConstraintOperator:
			data.Constraint.NodeMustNotExist = append(data.Constraint.NodeMustNotExist, constr.Node)
		}
	}

	validPath, err := path_selection.SelectPath(solutionContext(view), dep, data)
	if err != nil {
		return err
	}
	path, err := path_selection.ExpandEdge(solutionContext(view), dep, validPath)
	if err != nil {
		return err
	}
	// Once the path is selected & expanded, first add all the resources to the graph
	resources := make([]*construct.Resource, len(path))
	for i, pathId := range path {
		res, err := view.Vertex(pathId)
		switch {
		case errors.Is(err, graph.ErrVertexNotFound):
			res = construct.CreateResource(pathId)
			errs = errors.Join(errs, view.raw().AddVertex(res))

		case err != nil:
			errs = errors.Join(errs, err)
		}
		resources[i] = res
	}
	if errs != nil {
		return errs
	}

	// After all the resources, then add all the dependencies
	pstr := make([]string, len(path))
	edges := make([]construct.Edge, len(path)-1)
	for i, pathId := range path {
		pstr[i] = pathId.String()
		if i == 0 {
			continue
		}
		edge := construct.Edge{Source: path[i-1], Target: pathId}
		errs = errors.Join(errs, view.raw().AddEdge(edge.Source, edge.Target))
		edges[i-1] = edge
	}
	if len(pstr) > 2 {
		zap.S().Debugf("Expanded %s -> %s to %s", source, target, strings.Join(pstr, " -> "))
	}
	if errs != nil {
		return errs
	}

	if err := view.MakeResourcesOperational(resources); err != nil {
		return err
	}

	return view.MakeEdgesOperational(edges)
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
	return view.raw().RemoveEdge(source, target)
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
