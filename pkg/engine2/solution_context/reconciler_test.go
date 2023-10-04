package solution_context

import (
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/construct2/graphtest"
	"github.com/klothoplatform/klotho/pkg/engine2/enginetesting"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_reconnectFunctionalResources(t *testing.T) {
	tests := []struct {
		name         string
		resource     *construct.Resource
		explicit     bool
		initialstate []any
		mocks        []mock.Call
		want         result
	}{
		{
			name: "reconnectFunctionalResources reconnects functional resources ",
			resource: &construct.Resource{
				ID: construct.ResourceId{
					Provider: "mock",
					Type:     "resource2",
					Name:     "test",
				},
			},
			initialstate: []any{"mock:resource1:test", "mock:resource2:test", "mock:resource3:test", "mock:resource1:test -> mock:resource2:test", "mock:resource2:test -> mock:resource3:test"},
			mocks: []mock.Call{
				{
					Method:          "GetFunctionality",
					Arguments:       mock.Arguments{construct.ResourceId{Provider: "mock", Type: "resource1", Name: "test"}},
					ReturnArguments: mock.Arguments{knowledgebase.Compute},
				},
				{
					Method:          "GetFunctionality",
					Arguments:       mock.Arguments{construct.ResourceId{Provider: "mock", Type: "resource3", Name: "test"}},
					ReturnArguments: mock.Arguments{knowledgebase.Compute},
				},
				{
					Method:          "GetEdgeTemplate",
					Arguments:       mock.Arguments{construct.ResourceId{Provider: "mock", Type: "resource1", Name: "test"}, construct.ResourceId{Provider: "mock", Type: "resource3", Name: "test"}},
					ReturnArguments: mock.Arguments{&knowledgebase.EdgeTemplate{}, nil},
				},
				{
					Method:          "HasDirectPath",
					Arguments:       mock.Arguments{construct.ResourceId{Provider: "mock", Type: "resource1", Name: "test"}, construct.ResourceId{Provider: "mock", Type: "resource3", Name: "test"}},
					ReturnArguments: mock.Arguments{true},
				},
				{
					Method:          "GetResourceTemplate",
					Arguments:       mock.Arguments{mock.Anything},
					ReturnArguments: mock.Arguments{&knowledgebase.ResourceTemplate{}, nil},
				},
			},
			want: result{
				dataflow: []any{"mock:resource1:test", "mock:resource2:test", "mock:resource3:test",
					"mock:resource1:test -> mock:resource2:test", "mock:resource2:test -> mock:resource3:test",
					"mock:resource1:test -> mock:resource3:test"},
				deployment: []any{"mock:resource1:test", "mock:resource2:test", "mock:resource3:test",
					"mock:resource1:test -> mock:resource2:test", "mock:resource2:test -> mock:resource3:test",
					"mock:resource1:test -> mock:resource3:test"},
			},
		},
	}
	for _, tt := range tests {
		assert := assert.New(t)
		mockKB := &enginetesting.MockKB{}
		for _, m := range tt.mocks {
			mockKB.On(m.Method, m.Arguments...).Return(m.ReturnArguments...)
		}
		ctx := NewSolutionContext(mockKB)
		ctx.dataflowGraph = graphtest.MakeGraph(t, construct.NewGraph(), tt.initialstate...)
		ctx.deploymentGraph = graphtest.MakeGraph(t, construct.NewGraph(), tt.initialstate...)
		err := ctx.reconnectFunctionalResources(tt.resource)
		if !assert.NoError(err) {
			return
		}
		graphtest.AssertGraphEqual(t, graphtest.MakeGraph(t, construct.NewGraph(), tt.want.dataflow...), ctx.dataflowGraph)
		graphtest.AssertGraphEqual(t, graphtest.MakeGraph(t, construct.NewGraph(), tt.want.deployment...), ctx.deploymentGraph)
		mockKB.AssertExpectations(t)
	}

}
