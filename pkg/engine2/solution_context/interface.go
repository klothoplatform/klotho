package solution_context

import (
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/constraints"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

//go:generate mockgen -source=./interface.go --destination=../operational_eval/solution_context_mock_test.go --package=operational_eval

type (
	SolutionContext interface {
		// With returns a new context with a new key/value pair pushed onto the context stack.
		// Implementations must not mutate the original context.
		With(key string, value interface{}) SolutionContext
		KnowledgeBase() knowledgebase.TemplateKB
		Constraints() *constraints.Constraints
		RecordDecision(d SolveDecision)
		GetDecisions() DecisionRecords

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
	}

	OperationalView interface {
		construct.Graph

		MakeResourcesOperational(resources []*construct.Resource) error
		UpdateResourceID(oldId, newId construct.ResourceId) error
		MakeEdgesOperational(edges []construct.Edge) error
	}
)

func DynamicCtx(sol SolutionContext) knowledgebase.DynamicValueContext {
	return knowledgebase.DynamicValueContext{Graph: sol.DataflowGraph(), KnowledgeBase: sol.KnowledgeBase()}
}
