package solution_context

import (
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

func Downstream(
	sol SolutionContext,
	rid construct.ResourceId,
	layer knowledgebase.DependencyLayer,
) ([]construct.ResourceId, error) {
	return knowledgebase.Downstream(sol.DataflowGraph(), sol.KnowledgeBase(), rid, layer)
}

func DownstreamFunctional(sol SolutionContext, resource construct.ResourceId) ([]construct.ResourceId, error) {
	return knowledgebase.DownstreamFunctional(sol.DataflowGraph(), sol.KnowledgeBase(), resource)
}

func Upstream(
	sol SolutionContext,
	resource construct.ResourceId,
	layer knowledgebase.DependencyLayer,
) ([]construct.ResourceId, error) {
	return knowledgebase.Upstream(sol.DataflowGraph(), sol.KnowledgeBase(), resource, layer)
}

func UpstreamFunctional(sol SolutionContext, resource construct.ResourceId) ([]construct.ResourceId, error) {
	return knowledgebase.UpstreamFunctional(sol.DataflowGraph(), sol.KnowledgeBase(), resource)
}

func IsOperationalResourceSideEffect(sol SolutionContext, rid, sideEffect construct.ResourceId) (bool, error) {
	return knowledgebase.IsOperationalResourceSideEffect(sol.DataflowGraph(), sol.KnowledgeBase(), rid, sideEffect)
}
