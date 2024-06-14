package solution

import (
	construct "github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledgebase"
)

type (
	Solution interface {
		KnowledgeBase() knowledgebase.TemplateKB
		Constraints() *constraints.Constraints

		RecordDecision(d SolveDecision)
		GetDecisions() []SolveDecision

		DataflowGraph() construct.Graph
		DeploymentGraph() construct.Graph

		// OperationalView returns a graph that makes any resources or edges added operational as part of the operation.
		// Read operations come from the Dataflow graph.
		// Write operations will update both the Dataflow and Deployment graphs.
		OperationalView() OperationalView
		// RawView returns a graph that makes no changes beyond explicitly requested operations.
		// Read operations come from the Dataflow graph.
		// Write operations will update both the Dataflow and Deployment graphs.
		RawView() construct.Graph
		// GlobalTag returns the global tag for the solution context
		GlobalTag() string
		Outputs() map[string]construct.Output
	}

	OperationalView interface {
		construct.Graph

		MakeResourcesOperational(resources []*construct.Resource) error
		UpdateResourceID(oldId, newId construct.ResourceId) error
		MakeEdgesOperational(edges []construct.Edge) error
	}
)

func DynamicCtx(sol Solution) knowledgebase.DynamicValueContext {
	return knowledgebase.DynamicValueContext{Graph: sol.DataflowGraph(), KnowledgeBase: sol.KnowledgeBase()}
}
