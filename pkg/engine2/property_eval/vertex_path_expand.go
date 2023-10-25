//go:build ignore

package property_eval

import (
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/path_selection"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/klothoplatform/klotho/pkg/set"
)

type pathExpandVertex struct {
	Edge         construct.SimpleEdge
	selectedPath []construct.ResourceId
}

func (v *pathExpandVertex) Key() EvaluationKey {
	return EvaluationKey{Edge: v.Edge}
}

func (v *pathExpandVertex) Evaluate(eval *PropertyEval) error {
	expanded, err := path_selection.ExpandEdge(eval.Solution, v.Edge, v.selectedPath)
	if err != nil {
		return err
	}
	// rest the same as Operational.AddEdge, add resources, edges then make them operational
	return nil
}

func (v *pathExpandVertex) UpdateFrom(other EvaluationVertex) {
	panic("not implemented") // TODO: Implement
}

func (v *pathExpandVertex) Dependencies(
	cfgCtx knowledgebase.DynamicValueContext,
) (set.Set[construct.PropertyRef], graphStates, error) {
	deps := make(set.Set[construct.PropertyRef])

	for _, res := range []construct.ResourceId{v.Edge.Source, v.Edge.Target} {
		tmpl, err := cfgCtx.KB().GetResourceTemplate(res)
		for k := range tmpl.Properties {
			pt, err := cfgCtx.KB().GetResourcePropertyType(res, k)
			resType, ok := pt.(knowledgebase.ResourcePropertyType)
			if !ok {
				continue
			}
			for _, elem := range v.selectedPath[1 : len(v.selectedPath)-1] {
				if resType.Value.Matches(elem) {
					deps.Add(construct.PropertyRef{Resource: res, Property: k})
					break
				}
			}
		}
	}

	return deps, nil, nil
}

func (eval *PropertyEval) AddPath(source, target construct.ResourceId) (err error) {
	edge := construct.SimpleEdge{Source: source, Target: target}
	vertex := &pathExpandVertex{Edge: edge}
	vertex.selectedPath, err = path_selection.SelectPath(eval.Solution, edge, path_selection.EdgeData{})
	if len(vertex.selectedPath) <= 2 {
		return eval.AddEdges(edge)
	}
	vs := make(verticesAndDeps)
	vs.AddDependencies(solution_context.DynamicCtx(eval.Solution), vertex)
	return eval.enqueue(vs)
}
