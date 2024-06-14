package enginetesting

import (
	construct "github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	"github.com/klothoplatform/klotho/pkg/engine/solution"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledgebase"
	"github.com/stretchr/testify/mock"
)

type MockSolution struct {
	mock.Mock
	KB MockKB
}

func (m *MockSolution) KnowledgeBase() knowledgebase.TemplateKB {
	args := m.Called()
	return args.Get(0).(knowledgebase.TemplateKB)
}

func (m *MockSolution) Constraints() *constraints.Constraints {
	args := m.Called()
	return args.Get(0).(*constraints.Constraints)
}

func (m *MockSolution) RecordDecision(d solution.SolveDecision) {
	m.Called(d)
}

func (m *MockSolution) GetDecisions() []solution.SolveDecision {
	args := m.Called()
	return args.Get(0).([]solution.SolveDecision)
}

func (m *MockSolution) DataflowGraph() construct.Graph {
	args := m.Called()
	return args.Get(0).(construct.Graph)
}

func (m *MockSolution) DeploymentGraph() construct.Graph {
	args := m.Called()
	return args.Get(0).(construct.Graph)
}

func (m *MockSolution) OperationalView() solution.OperationalView {
	args := m.Called()
	return args.Get(0).(solution.OperationalView)
}

func (m *MockSolution) RawView() construct.Graph {
	args := m.Called()
	return args.Get(0).(construct.Graph)
}

func (m *MockSolution) GlobalTag() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockSolution) Outputs() map[string]construct.Output {
	args := m.Called()
	return args.Get(0).(map[string]construct.Output)
}
