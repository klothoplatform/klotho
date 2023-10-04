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

type result struct {
	dataflow   []any
	deployment []any
}

func Test_ListResources(t *testing.T) {
	assert := assert.New(t)
	mockKB := &enginetesting.MockKB{}
	ctx := NewSolutionContext(mockKB)
	ctx.dataflowGraph = graphtest.MakeGraph(t, construct.NewGraph(), "mock:mock:test", "mock:mock:test2")
	resources, err := ctx.ListResources()
	assert.NoError(err)
	assert.Equal(resources, []*construct.Resource{
		{ID: construct.ResourceId{Provider: "mock", Type: "mock", Name: "test"}},
		{ID: construct.ResourceId{Provider: "mock", Type: "mock", Name: "test2"}},
	})

}

func Test_addResource(t *testing.T) {
	tests := []struct {
		name            string
		resource        *construct.Resource
		makeOperational bool
		mocks           []mock.Call
		want            []any
	}{
		{
			name: "AddResource",
			resource: &construct.Resource{
				ID: construct.ResourceId{
					Provider: "mock",
					Type:     "resource1",
					Name:     "test",
				},
			},
			want: []any{"mock:resource1:test"},
		},
		{
			name: "AddResource calls make operational",
			resource: &construct.Resource{
				ID: construct.ResourceId{
					Provider: "mock",
					Type:     "resource2",
					Name:     "test",
				},
			},
			makeOperational: true,
			mocks: []mock.Call{
				{
					Method:          "GetResourceTemplate",
					Arguments:       mock.Arguments{mock.Anything},
					ReturnArguments: mock.Arguments{&knowledgebase.ResourceTemplate{}, nil},
				},
			},
			want: []any{"mock:resource2:test"},
		},
	}
	for _, tt := range tests {
		assert := assert.New(t)
		mockKB := &enginetesting.MockKB{}
		for _, m := range tt.mocks {
			mockKB.On(m.Method, m.Arguments...).Return(m.ReturnArguments...)
		}
		ctx := NewSolutionContext(mockKB)
		err := ctx.addResource(tt.resource, tt.makeOperational)
		assert.NoError(err)
		resources, err := ctx.ListResources()
		assert.NoError(err)
		assert.Equal([]*construct.Resource{tt.resource}, resources)
		mockKB.AssertExpectations(t)
	}
}

func Test_addDependency(t *testing.T) {
	tests := []struct {
		name            string
		resource        *construct.Resource
		dependency      *construct.Resource
		initialstate    []any
		makeOperational bool
		mocks           []mock.Call
		want            result
	}{
		{
			name: "AddDependency",
			resource: &construct.Resource{
				ID: construct.ResourceId{
					Provider: "mock",
					Type:     "resource1",
					Name:     "test",
				},
			},
			dependency: &construct.Resource{
				ID: construct.ResourceId{
					Provider: "mock",
					Type:     "resource2",
					Name:     "test",
				},
			},
			initialstate: []any{"mock:resource1:test", "mock:resource2:test"},
			mocks: []mock.Call{
				{
					Method:          "GetEdgeTemplate",
					Arguments:       mock.Arguments{mock.Anything, mock.Anything},
					ReturnArguments: mock.Arguments{&knowledgebase.EdgeTemplate{}, nil},
				},
			},
			want: result{
				dataflow:   []any{"mock:resource1:test", "mock:resource2:test", "mock:resource1:test -> mock:resource2:test"},
				deployment: []any{"mock:resource1:test", "mock:resource2:test", "mock:resource1:test -> mock:resource2:test"},
			},
		},
		{
			name: "AddDependency calls add path",
			resource: &construct.Resource{
				ID: construct.ResourceId{

					Provider: "mock",
					Type:     "resource1",
					Name:     "test",
				},
			},
			dependency: &construct.Resource{
				ID: construct.ResourceId{
					Provider: "mock",
					Type:     "resource2",

					Name: "test",
				},
			},
			initialstate:    []any{"mock:resource1:test", "mock:resource2:test"},
			makeOperational: true,
			mocks: []mock.Call{
				{
					Method:          "GetEdgeTemplate",
					Arguments:       mock.Arguments{mock.Anything, mock.Anything},
					ReturnArguments: mock.Arguments{&knowledgebase.EdgeTemplate{DeploymentOrderReversed: true}, nil},
				},
				{
					Method:          "HasDirectPath",
					Arguments:       mock.Arguments{mock.Anything, mock.Anything},
					ReturnArguments: mock.Arguments{true},
				},
				{
					Method:          "GetResourceTemplate",
					Arguments:       mock.Arguments{mock.Anything},
					ReturnArguments: mock.Arguments{&knowledgebase.ResourceTemplate{}, nil},
				},
			},
			want: result{
				dataflow:   []any{"mock:resource1:test", "mock:resource2:test", "mock:resource1:test -> mock:resource2:test"},
				deployment: []any{"mock:resource1:test", "mock:resource2:test", "mock:resource2:test -> mock:resource1:test"},
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
		err := ctx.addDependency(tt.resource, tt.dependency, tt.makeOperational)
		if !assert.NoError(err) {
			return
		}
		graphtest.AssertGraphEqual(t, graphtest.MakeGraph(t, construct.NewGraph(), tt.want.dataflow...), ctx.dataflowGraph)
		graphtest.AssertGraphEqual(t, graphtest.MakeGraph(t, construct.NewGraph(), tt.want.deployment...), ctx.deploymentGraph)
		mockKB.AssertExpectations(t)
	}
}

func Test_RemoveResource(t *testing.T) {
	tests := []struct {
		name         string
		resource     *construct.Resource
		explicit     bool
		initialstate []any
		mocks        []mock.Call
		want         result
	}{
		{
			name: "RemoveResource works on explicit and known functionality",
			resource: &construct.Resource{
				ID: construct.ResourceId{
					Provider: "mock",
					Type:     "resource1",
					Name:     "test",
				},
			},
			initialstate: []any{"mock:resource1:test"},
			explicit:     true,
			mocks: []mock.Call{
				{
					Method:          "GetResourceTemplate",
					Arguments:       mock.Arguments{mock.Anything},
					ReturnArguments: mock.Arguments{&knowledgebase.ResourceTemplate{Classification: knowledgebase.Classification{Is: []string{"compute"}}}, nil},
				},
			},
			want: result{
				dataflow:   []any{},
				deployment: []any{},
			},
		},
		{
			name: "RemoveResource doesnt remove on non explicit and known functionality",
			resource: &construct.Resource{
				ID: construct.ResourceId{
					Provider: "mock",
					Type:     "resource1",
					Name:     "test",
				},
			},
			initialstate: []any{"mock:resource1:test"},
			explicit:     false,
			mocks: []mock.Call{
				{
					Method:          "GetResourceTemplate",
					Arguments:       mock.Arguments{mock.Anything},
					ReturnArguments: mock.Arguments{&knowledgebase.ResourceTemplate{Classification: knowledgebase.Classification{Is: []string{"compute"}}}, nil},
				},
			},
			want: result{
				dataflow:   []any{"mock:resource1:test"},
				deployment: []any{"mock:resource1:test"},
			},
		},
		{
			name: "RemoveResource reconnects functional resources if removal is for glue",
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
					Method:          "GetResourceTemplate",
					Arguments:       mock.Arguments{construct.ResourceId{Provider: "mock", Type: "resource1", Name: "test"}},
					ReturnArguments: mock.Arguments{&knowledgebase.ResourceTemplate{Classification: knowledgebase.Classification{Is: []string{"compute"}}}, nil},
				},
				{
					Method:          "GetResourceTemplate",
					Arguments:       mock.Arguments{construct.ResourceId{Provider: "mock", Type: "resource2", Name: "test"}},
					ReturnArguments: mock.Arguments{&knowledgebase.ResourceTemplate{}, nil},
				},
				{
					Method:          "GetResourceTemplate",
					Arguments:       mock.Arguments{construct.ResourceId{Provider: "mock", Type: "resource3", Name: "test"}},
					ReturnArguments: mock.Arguments{&knowledgebase.ResourceTemplate{Classification: knowledgebase.Classification{Is: []string{"compute"}}}, nil},
				},
				{
					Method:          "GetEdgeTemplate",
					Arguments:       mock.Arguments{mock.Anything, mock.Anything},
					ReturnArguments: mock.Arguments{&knowledgebase.EdgeTemplate{}, nil},
				},
				{
					Method:          "HasDirectPath",
					Arguments:       mock.Arguments{mock.Anything, mock.Anything},
					ReturnArguments: mock.Arguments{true},
				},
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
			},
			want: result{
				dataflow:   []any{"mock:resource1:test", "mock:resource3:test", "mock:resource1:test -> mock:resource3:test"},
				deployment: []any{"mock:resource1:test", "mock:resource3:test", "mock:resource1:test -> mock:resource3:test"},
			},
		},
		{
			name: "RemoveResource removes all orphaned resources",
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
					Method:          "GetResourceTemplate",
					Arguments:       mock.Arguments{mock.Anything},
					ReturnArguments: mock.Arguments{&knowledgebase.ResourceTemplate{}, nil},
				},
				{
					Method:          "GetEdgeTemplate",
					Arguments:       mock.Arguments{mock.Anything, mock.Anything},
					ReturnArguments: mock.Arguments{&knowledgebase.EdgeTemplate{}, nil},
				},
				{
					Method:          "GetFunctionality",
					Arguments:       mock.Arguments{mock.Anything},
					ReturnArguments: mock.Arguments{knowledgebase.Unknown},
				},
			},
			want: result{
				dataflow:   []any{},
				deployment: []any{},
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
		err := ctx.RemoveResource(tt.resource, tt.explicit)
		if !assert.NoError(err) {
			return
		}
		graphtest.AssertGraphEqual(t, graphtest.MakeGraph(t, construct.NewGraph(), tt.want.dataflow...), ctx.dataflowGraph)
		graphtest.AssertGraphEqual(t, graphtest.MakeGraph(t, construct.NewGraph(), tt.want.deployment...), ctx.deploymentGraph)
		mockKB.AssertExpectations(t)
	}

}
