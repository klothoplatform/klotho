package enginetesting

import (
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/construct2/graphtest"
	"github.com/klothoplatform/klotho/pkg/engine2/constraints"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/stretchr/testify/mock"
)

type TestSolution struct {
	mock.Mock

	KB     MockKB
	Constr constraints.Constraints

	dataflow, deployment construct.Graph
}

func NewTestSolution() *TestSolution {
	sol := &TestSolution{
		dataflow:   construct.NewGraph(),
		deployment: construct.NewAcyclicGraph(),
	}
	return sol
}

func (sol *TestSolution) LoadState(t *testing.T, initGraph ...any) {
	graphtest.MakeGraph(t, sol.RawView(), initGraph...)
}

func (sol *TestSolution) With(key string, value interface{}) solution_context.SolutionContext {
	return sol
}

func (sol *TestSolution) KnowledgeBase() knowledgebase.TemplateKB {
	return &sol.KB
}

func (sol *TestSolution) Constraints() *constraints.Constraints {
	return &sol.Constr
}

func (sol *TestSolution) RecordDecision(d solution_context.SolveDecision) {}

func (sol *TestSolution) DataflowGraph() construct.Graph {
	return sol.dataflow
}

func (sol *TestSolution) DeploymentGraph() construct.Graph {
	return sol.deployment
}

func (sol *TestSolution) OperationalView() solution_context.OperationalView {
	return testOperationalView{Graph: sol.RawView(), Mock: &sol.Mock}
}

func (sol *TestSolution) RawView() construct.Graph {
	return solution_context.NewRawView(sol)
}

type testOperationalView struct {
	construct.Graph
	Mock *mock.Mock
}

func (view testOperationalView) MakeResourcesOperational(resources []*construct.Resource) (construct.ResourceIdChangeResults, error) {
	args := view.Mock.Called(resources)
	return args.Get(0).(construct.ResourceIdChangeResults), args.Error(1)
}

func (view testOperationalView) MakeEdgeOperational(
	source, target construct.ResourceId,
) ([]*construct.Resource, []construct.Edge, error) {
	args := view.Mock.Called(source, target)
	return args.Get(0).([]*construct.Resource), args.Get(1).([]construct.Edge), args.Error(2)
}
