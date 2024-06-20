package enginetesting

import (
	"context"
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/construct/graphtest"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	"github.com/klothoplatform/klotho/pkg/engine/solution"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledgebase"
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

func (sol *TestSolution) Context() context.Context {
	return context.Background()
}

func (sol *TestSolution) LoadState(t *testing.T, initGraph ...any) {
	graphtest.MakeGraph(t, sol.RawView(), initGraph...)

	// Start recording changes after initial graph is loaded.
	sol.dataflow = graphtest.RecordChanges(sol.dataflow)
	sol.deployment = graphtest.RecordChanges(sol.deployment)
}

func (sol *TestSolution) DataflowChanges() *graphtest.GraphChanges {
	return sol.dataflow.(*graphtest.GraphChanges)
}

func (sol *TestSolution) DeploymentChanges() *graphtest.GraphChanges {
	return sol.deployment.(*graphtest.GraphChanges)
}

func (sol *TestSolution) KnowledgeBase() knowledgebase.TemplateKB {
	return &sol.KB
}

func (sol *TestSolution) Constraints() *constraints.Constraints {
	return &sol.Constr
}

func (sol *TestSolution) RecordDecision(d solution.SolveDecision) {}

func (sol *TestSolution) GetDecisions() []solution.SolveDecision {
	return nil
}

func (sol *TestSolution) DataflowGraph() construct.Graph {
	return sol.dataflow
}

func (sol *TestSolution) DeploymentGraph() construct.Graph {
	return sol.deployment
}

func (sol *TestSolution) OperationalView() solution.OperationalView {
	return testOperationalView{Graph: sol.RawView(), Mock: &sol.Mock}
}

func (sol *TestSolution) RawView() construct.Graph {
	return solution.NewRawView(sol)
}

func (sol *TestSolution) GlobalTag() string {
	return "test"
}

func (sol *TestSolution) Outputs() map[string]construct.Output {
	return nil
}

type testOperationalView struct {
	construct.Graph
	Mock *mock.Mock
}

func (view testOperationalView) MakeResourcesOperational(resources []*construct.Resource) error {
	args := view.Mock.Called(resources)
	return args.Error(0)
}

func (view testOperationalView) UpdateResourceID(oldId, newId construct.ResourceId) error {
	args := view.Mock.Called(oldId, newId)
	return args.Error(0)
}

func (view testOperationalView) MakeEdgesOperational(edges []construct.Edge) error {
	args := view.Mock.Called(edges)
	return args.Error(0)
}

type ExpectedGraphs struct {
	Dataflow, Deployment []any
}

func (expect ExpectedGraphs) AssertEqual(t *testing.T, sol solution.Solution) {
	if expect.Dataflow != nil {
		graphtest.AssertGraphEqual(t,
			graphtest.MakeGraph(t, construct.NewGraph(), expect.Dataflow...),
			sol.DataflowGraph(),
			"Dataflow",
		)
	}
	if expect.Deployment != nil {
		graphtest.AssertGraphEqual(t,
			graphtest.MakeGraph(t, construct.NewGraph(), expect.Deployment...),
			sol.DeploymentGraph(),
			"Deployment",
		)
	}
}
