package engine2

import (
	"errors"
	"fmt"
	"strings"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/constraints"
	"github.com/klothoplatform/klotho/pkg/engine2/operational_rule"
	"github.com/klothoplatform/klotho/pkg/engine2/path_selection"
	property_eval "github.com/klothoplatform/klotho/pkg/engine2/property_eval"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
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
	return property_eval.SetupResources(solutionContext(view), resources)
}

func (view MakeOperationalView) Vertex(hash construct.ResourceId) (*construct.Resource, error) {
	return view.raw().Vertex(hash)
}

func (view MakeOperationalView) VertexWithProperties(hash construct.ResourceId) (*construct.Resource, graph.VertexProperties, error) {
	return view.raw().VertexWithProperties(hash)
}

func (view MakeOperationalView) RemoveVertex(hash construct.ResourceId) error {
	return view.raw().RemoveVertex(hash)
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

	path, err := path_selection.SelectPath(solutionContext(view), dep, data)
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
	if len(path) == 2 {
		zap.S().Infof("Adding edge %s -> %s (for %s -> %s)", path[0], path[1], source, target)
		errs = view.raw().AddEdge(path[0], path[1], options...)
	} else {
		pstr := make([]string, len(path))
		for i, pathId := range path {
			pstr[i] = pathId.String()
			if i == 0 {
				continue
			}
			errs = errors.Join(errs, view.raw().AddEdge(path[i-1], pathId))
		}
		zap.S().Infof("Expanded %s -> %s to %s", source, target, strings.Join(pstr, " -> "))
	}
	if errs != nil {
		return errs
	}

	// After the graph is set up, first make all the resources operational
	err = property_eval.SetupResources(solutionContext(view), resources)
	if err != nil {
		return err
	}

	// Finally, after the graph and resources are operational, make all the edges operational
	var result operational_rule.Result
	for i, pathId := range path {
		if i == 0 {
			continue
		}
		addRes, addDep, err := view.MakeEdgeOperational(path[i-1], pathId)
		errs = errors.Join(errs, err)
		result.CreatedResources = append(result.CreatedResources, addRes...)
		result.AddedDependencies = append(result.AddedDependencies, addDep...)
	}
	if errs != nil {
		return errs
	}

	for len(result.CreatedResources) > 0 || len(result.AddedDependencies) > 0 {
		err = property_eval.SetupResources(solutionContext(view), result.CreatedResources)
		if err != nil {
			return err
		}
		var nextResult operational_rule.Result
		for _, edge := range result.AddedDependencies {
			addRes, addDep, err := view.MakeEdgeOperational(edge.Source, edge.Target)
			errs = errors.Join(errs, err)
			nextResult.CreatedResources = append(nextResult.CreatedResources, addRes...)
			nextResult.AddedDependencies = append(nextResult.AddedDependencies, addDep...)
		}
		if errs != nil {
			return errs
		}
		result = nextResult
	}

	return nil
}

func (view MakeOperationalView) MakeEdgeOperational(
	source, target construct.ResourceId,
) ([]*construct.Resource, []construct.Edge, error) {
	tmpl := view.KB.GetEdgeTemplate(source, target)
	if tmpl == nil {
		return nil, nil, nil
	}

	edge := construct.Edge{Source: source, Target: target}
	view.stack = append(view.stack, solution_context.KV{Key: "edge", Value: edge})

	opCtx := operational_rule.OperationalRuleContext{
		Data:     knowledgebase.DynamicValueData{Edge: &edge},
		Solution: solutionContext(view),
	}

	var result operational_rule.Result
	var errs error
	for _, rule := range tmpl.OperationalRules {
		ruleResult, err := opCtx.HandleOperationalRule(rule)
		errs = errors.Join(errs, err)
		result.Append(ruleResult)
	}
	return result.CreatedResources, result.AddedDependencies, errs
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
	clone, err := solutionContext(view).Clone(true)
	if err != nil {
		return nil, err
	}
	return MakeOperationalView(clone), nil
}

func (view MakeOperationalView) Order() (int, error) {
	return view.Dataflow.Order()
}

func (view MakeOperationalView) Size() (int, error) {
	return view.Dataflow.Size()
}
