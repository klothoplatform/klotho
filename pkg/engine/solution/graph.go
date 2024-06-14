package solution

import (
	construct "github.com/klothoplatform/klotho/pkg/construct"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledgebase"
)

func Downstream(
	sol Solution,
	rid construct.ResourceId,
	layer knowledgebase.DependencyLayer,
) ([]construct.ResourceId, error) {
	return knowledgebase.Downstream(sol.DataflowGraph(), sol.KnowledgeBase(), rid, layer)
}

func DownstreamFunctional(sol Solution, resource construct.ResourceId) ([]construct.ResourceId, error) {
	return knowledgebase.DownstreamFunctional(sol.DataflowGraph(), sol.KnowledgeBase(), resource)
}

func Upstream(
	sol Solution,
	resource construct.ResourceId,
	layer knowledgebase.DependencyLayer,
) ([]construct.ResourceId, error) {
	return knowledgebase.Upstream(sol.DataflowGraph(), sol.KnowledgeBase(), resource, layer)
}

func UpstreamFunctional(sol Solution, resource construct.ResourceId) ([]construct.ResourceId, error) {
	return knowledgebase.UpstreamFunctional(sol.DataflowGraph(), sol.KnowledgeBase(), resource)
}

func IsOperationalResourceSideEffect(sol Solution, rid, sideEffect construct.ResourceId) (bool, error) {
	return knowledgebase.IsOperationalResourceSideEffect(sol.DataflowGraph(), sol.KnowledgeBase(), rid, sideEffect)
}
