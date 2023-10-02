package solution_context

import (
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/construct2/graphtest"
	"github.com/klothoplatform/klotho/pkg/engine2/constraints"
	"github.com/klothoplatform/klotho/pkg/engine2/enginetesting"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_ApplyApplicationConstraint(t *testing.T) {
	tests := []struct {
		name         string
		constraint   *constraints.ApplicationConstraint
		initialState []any
		mocks        []mock.Call
		want         []any
	}{
		{
			name: "AddConstraintOperator",
			constraint: &constraints.ApplicationConstraint{
				Operator: constraints.AddConstraintOperator,
				Node: construct.ResourceId{
					Provider: "mock",
					Type:     "resource1",
					Name:     "test",
				},
			},
			want: []any{"mock:resource1:test"},
		},
		{
			name: "RemoveConstraintOperator",
			constraint: &constraints.ApplicationConstraint{
				Operator: constraints.RemoveConstraintOperator,
				Node: construct.ResourceId{
					Provider: "mock",
					Type:     "resource1",
					Name:     "test",
				},
			},
			initialState: []any{"mock:resource1:test"},
			mocks: []mock.Call{
				{
					Method:          "GetResourceTemplate",
					Arguments:       mock.Arguments{mock.Anything},
					ReturnArguments: mock.Arguments{&knowledgebase.ResourceTemplate{}, nil},
				},
				{
					Method:          "GetFunctionality",
					Arguments:       mock.Arguments{mock.Anything},
					ReturnArguments: mock.Arguments{knowledgebase.Compute},
				},
			},
			want: []any{},
		},
		// {
		// 	name: "ReplaceConstraintOperator same qualified type",
		// 	constraint: &constraints.ApplicationConstraint{
		// 		Operator: constraints.ReplaceConstraintOperator,
		// 		Node: construct.ResourceId{
		// 			Provider: "mock",
		// 			Type:     "resource1",
		// 			Name:     "test",
		// 		},
		// 		Replacement: construct.ResourceId{
		// 			Provider: "mock",
		// 			Type:     "resource1",
		// 			Name:     "test2",
		// 		},
		// 	},
		// 	initialState: []any{"mock:resource1:test"},
		// 	mocks: []mock.Call{
		// 		{
		// 			Method:          "GetResourceTemplate",
		// 			Arguments:       mock.Arguments{mock.Anything},
		// 			ReturnArguments: mock.Arguments{&knowledgebase.ResourceTemplate{}, nil},
		// 		},
		// 		{

	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			mockKb := &enginetesting.MockKB{}
			for _, m := range tt.mocks {
				mockKb.On(m.Method, m.Arguments...).Return(m.ReturnArguments...)
			}
			ctx := NewSolutionContext(mockKb)
			ctx.dataflowGraph = graphtest.MakeGraph(t, construct.NewGraph(), tt.initialState...)
			ctx.deploymentGraph = graphtest.MakeGraph(t, construct.NewGraph(), tt.initialState...)
			err := ctx.ApplyApplicationConstraint(tt.constraint)
			if !assert.NoError(err) {
				return
			}
			want := graphtest.MakeGraph(t, construct.NewGraph(), tt.want...)
			graphtest.AssertGraphEqual(t, want, ctx.dataflowGraph)
		})
	}
}
