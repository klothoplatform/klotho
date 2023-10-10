package solution_context

import (
	"errors"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
)

type RawAccessView struct {
	inner SolutionContext
}

func NewRawView(inner SolutionContext) RawAccessView {
	return RawAccessView{inner: inner}
}

func (view RawAccessView) Traits() *graph.Traits {
	return view.inner.DataflowGraph().Traits()
}

func (view RawAccessView) AddVertex(value *construct.Resource, options ...func(*graph.VertexProperties)) error {
	dfErr := view.inner.DataflowGraph().AddVertex(value, options...)
	deplErr := view.inner.DeploymentGraph().AddVertex(value, options...)
	if errors.Is(dfErr, graph.ErrVertexAlreadyExists) && errors.Is(deplErr, graph.ErrVertexAlreadyExists) {
		return graph.ErrVertexAlreadyExists
	}
	var err error
	if dfErr != nil && !errors.Is(dfErr, graph.ErrVertexAlreadyExists) {
		err = errors.Join(err, dfErr)
	}
	if deplErr != nil && !errors.Is(deplErr, graph.ErrVertexAlreadyExists) {
		err = errors.Join(err, deplErr)
	}
	if err != nil {
		return err
	}

	view.inner.RecordDecision(AddResourceDecision{Resource: value.ID})
	return nil
}

func (view RawAccessView) AddVerticesFrom(g construct.Graph) error {
	ordered, err := construct.ReverseTopologicalSort(g)
	if err != nil {
		return err
	}
	var errs error
	for _, rid := range ordered {
		//! This will cause issues when we solve multiple graphs
		// this should copy the vertex instead of using the same pointer
		res, err := g.Vertex(rid)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		err = view.AddVertex(res)
		//? should the vertex overwrite?
		if err != nil && !errors.Is(err, graph.ErrVertexAlreadyExists) {
			errs = errors.Join(errs, err)
		}
	}
	return errs
}

func (view RawAccessView) Vertex(hash construct.ResourceId) (*construct.Resource, error) {
	return view.inner.DataflowGraph().Vertex(hash)
}

func (view RawAccessView) VertexWithProperties(hash construct.ResourceId) (*construct.Resource, graph.VertexProperties, error) {
	return view.inner.DataflowGraph().VertexWithProperties(hash)
}

func (view RawAccessView) RemoveVertex(hash construct.ResourceId) error {
	err := errors.Join(
		view.inner.DataflowGraph().RemoveVertex(hash),
		view.inner.DeploymentGraph().RemoveVertex(hash),
	)
	if err != nil {
		return err
	}
	view.inner.RecordDecision(RemoveResourceDecision{Resource: hash})
	return nil
}

func (view RawAccessView) AddEdge(source, target construct.ResourceId, options ...func(*graph.EdgeProperties)) error {
	dfErr := view.inner.DataflowGraph().AddEdge(source, target, options...)

	var deplErr error
	et := view.inner.KnowledgeBase().GetEdgeTemplate(source, target)
	if et != nil && et.DeploymentOrderReversed {
		deplErr = view.inner.DeploymentGraph().AddEdge(target, source)
	} else {
		deplErr = view.inner.DeploymentGraph().AddEdge(source, target)
	}
	if errors.Is(dfErr, graph.ErrEdgeAlreadyExists) && errors.Is(deplErr, graph.ErrEdgeAlreadyExists) {
		return graph.ErrEdgeAlreadyExists
	}

	var err error
	if dfErr != nil && !errors.Is(dfErr, graph.ErrEdgeAlreadyExists) {
		err = errors.Join(err, dfErr)
	}
	if deplErr != nil && !errors.Is(deplErr, graph.ErrEdgeAlreadyExists) {
		err = errors.Join(err, deplErr)
	}
	if err != nil {
		return err
	}

	view.inner.RecordDecision(AddDependencyDecision{
		From: source,
		To:   target,
	})
	return nil
}

func (view RawAccessView) AddEdgesFrom(g construct.Graph) error {
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

func (view RawAccessView) Edge(source, target construct.ResourceId) (construct.ResourceEdge, error) {
	return view.inner.DataflowGraph().Edge(source, target)
}

func (view RawAccessView) Edges() ([]construct.Edge, error) {
	return view.inner.DataflowGraph().Edges()
}

func (view RawAccessView) UpdateEdge(source, target construct.ResourceId, options ...func(properties *graph.EdgeProperties)) error {
	dfErr := view.inner.DataflowGraph().UpdateEdge(source, target, options...)

	var deplErr error
	et := view.inner.KnowledgeBase().GetEdgeTemplate(source, target)
	if et != nil && et.DeploymentOrderReversed {
		deplErr = view.inner.DeploymentGraph().UpdateEdge(target, source, options...)
	} else {
		deplErr = view.inner.DeploymentGraph().UpdateEdge(source, target, options...)
	}
	return errors.Join(dfErr, deplErr)
}

func (view RawAccessView) RemoveEdge(source, target construct.ResourceId) error {
	dfErr := view.inner.DataflowGraph().RemoveEdge(source, target)

	var deplErr error
	et := view.inner.KnowledgeBase().GetEdgeTemplate(source, target)
	if et != nil && et.DeploymentOrderReversed {
		deplErr = view.inner.DeploymentGraph().RemoveEdge(target, source)
	} else {
		deplErr = view.inner.DeploymentGraph().RemoveEdge(source, target)
	}

	if err := errors.Join(dfErr, deplErr); err != nil {
		return err
	}

	view.inner.RecordDecision(RemoveDependencyDecision{
		From: source,
		To:   target,
	})
	return nil
}

func (view RawAccessView) AdjacencyMap() (map[construct.ResourceId]map[construct.ResourceId]construct.Edge, error) {
	return view.inner.DataflowGraph().AdjacencyMap()
}

func (view RawAccessView) PredecessorMap() (map[construct.ResourceId]map[construct.ResourceId]construct.Edge, error) {
	return view.inner.DataflowGraph().PredecessorMap()
}

func (view RawAccessView) Clone() (construct.Graph, error) {
	return nil, errors.New("cannot clone a view")
}

func (view RawAccessView) Order() (int, error) {
	return view.inner.DataflowGraph().Order()
}

func (view RawAccessView) Size() (int, error) {
	return view.inner.DataflowGraph().Size()
}
