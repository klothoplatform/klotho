package reconciler

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
		resource     string
		explicit     bool
		initialstate []any
		mocks        []mock.Call
		want         enginetesting.ExpectedGraphs
	}{
		{
			name:         "reconnectFunctionalResources reconnects functional resources ",
			resource:     "mock:resource2:test",
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
			want: enginetesting.ExpectedGraphs{
				Dataflow: []any{"mock:resource1:test", "mock:resource2:test", "mock:resource3:test",
					"mock:resource1:test -> mock:resource2:test", "mock:resource2:test -> mock:resource3:test",
					"mock:resource1:test -> mock:resource3:test"},
				Deployment: []any{"mock:resource1:test", "mock:resource2:test", "mock:resource3:test",
					"mock:resource1:test -> mock:resource2:test", "mock:resource2:test -> mock:resource3:test",
					"mock:resource1:test -> mock:resource3:test"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			ctx := enginetesting.NewTestSolution(t, tt.initialstate...)
			for _, m := range tt.mocks {
				ctx.KB.On(m.Method, m.Arguments...).Return(m.ReturnArguments...)
			}

			resource := graphtest.ParseId(t, tt.resource)
			err := reconnectFunctionalResources(ctx, resource)
			if !assert.NoError(err) {
				return
			}

			tt.want.AssertEqual(t, ctx)
		})
	}

}
