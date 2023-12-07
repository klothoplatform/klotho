package enginetesting

import (
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/constraints"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/stretchr/testify/mock"
)

type MockSolution struct {
	mock.Mock
	KB MockKB
}

func (m *MockSolution) With(key string, value interface{}) solution_context.SolutionContext {
	return m
}

func (m *MockSolution) KnowledgeBase() knowledgebase.TemplateKB {
	args := m.Called()
	return args.Get(0).(knowledgebase.TemplateKB)
}

func (m *MockSolution) Constraints() *constraints.Constraints {
	args := m.Called()
	return args.Get(0).(*constraints.Constraints)
}

func (m *MockSolution) RecordDecision(d solution_context.SolveDecision) {
	m.Called(d)
}

func (m *MockSolution) GetDecisions() solution_context.DecisionRecords {
	args := m.Called()
	return args.Get(0).(solution_context.DecisionRecords)
}

func (m *MockSolution) DataflowGraph() construct.Graph {
	args := m.Called()
	return args.Get(0).(construct.Graph)
}

func (m *MockSolution) DeploymentGraph() construct.Graph {
	args := m.Called()
	return args.Get(0).(construct.Graph)
}

func (m *MockSolution) OperationalView() solution_context.OperationalView {
	args := m.Called()
	return args.Get(0).(solution_context.OperationalView)
}

func (m *MockSolution) RawView() construct.Graph {
	args := m.Called()
	return args.Get(0).(construct.Graph)
}
