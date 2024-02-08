package operational_rule

import (
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/construct2/graphtest"
	"github.com/klothoplatform/klotho/pkg/engine/enginetesting"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_SpreadPlacer(t *testing.T) {
	tests := []struct {
		name               string
		resource           *construct.Resource
		availableResources []*construct.Resource
		initialState       []any
		step               knowledgebase.OperationalStep
		numNeeded          int
		want               graphtest.GraphChanges
	}{
		{
			name: "does nothing if no available resources",
			resource: &construct.Resource{ID: construct.ResourceId{
				Type: "test",
				Name: "test1",
			}},
			numNeeded: 1,
		},
		{
			name: "does nothing if one available resource",
			resource: &construct.Resource{ID: construct.ResourceId{
				Type: "test",
				Name: "test1",
			}},
			availableResources: []*construct.Resource{
				{ID: construct.ResourceId{
					Type: "test",
					Name: "test2",
				}},
			},
			numNeeded: 1,
		},
		{
			name:     "no resources placed yet, places in first resource",
			resource: &construct.Resource{ID: graphtest.ParseId(t, "test:test:test1")},
			availableResources: []*construct.Resource{
				{ID: graphtest.ParseId(t, "test:parent:test2")},
				{ID: graphtest.ParseId(t, "test:parent:test3")},
			},
			initialState: []any{
				"test:test:test1",
				"test:parent:test2",
				"test:parent:test3",
			},
			step: knowledgebase.OperationalStep{
				Direction: knowledgebase.DirectionDownstream,
			},
			numNeeded: 1,
			want: graphtest.GraphChanges{
				AddedEdges: []construct.Edge{
					{
						Source: graphtest.ParseId(t, "test:test:test1"),
						Target: graphtest.ParseId(t, "test:parent:test2"),
					},
				},
			},
		},
		{
			name:     "chooses placement in spot with least dependencies",
			resource: &construct.Resource{ID: graphtest.ParseId(t, "test:test:test1")},
			availableResources: []*construct.Resource{
				{ID: graphtest.ParseId(t, "test:parent:test2")},
				{ID: graphtest.ParseId(t, "test:parent:test3")},
			},
			initialState: []any{
				"test:test:test1",
				"test:test:test2",
				"test:parent:test2",
				"test:parent:test3",
				"test:test:test2 -> test:parent:test2",
			},
			step: knowledgebase.OperationalStep{
				Direction: knowledgebase.DirectionDownstream,
			},
			numNeeded: 1,
			want: graphtest.GraphChanges{
				AddedEdges: []construct.Edge{
					{
						Source: graphtest.ParseId(t, "test:test:test1"),
						Target: graphtest.ParseId(t, "test:parent:test3"),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			p := &SpreadPlacer{}
			testSol := enginetesting.NewTestSolution()
			testSol.KB.On("GetResourceTemplate", mock.Anything).Return(&knowledgebase.ResourceTemplate{}, nil)
			testSol.KB.On("GetEdgeTemplate", mock.Anything, mock.Anything).Return(&knowledgebase.EdgeTemplate{})
			testSol.LoadState(t, tt.initialState...)
			p.SetCtx(OperationalRuleContext{
				Solution: testSol,
			})
			err := p.PlaceResources(tt.resource, tt.step, tt.availableResources, &tt.numNeeded)
			if !assert.NoError(err) {
				return
			}
			tt.want.AssertEqual(t, testSol.DataflowChanges())
		})
	}
}

func Test_ClusterPlacer(t *testing.T) {
	tests := []struct {
		name               string
		resource           *construct.Resource
		availableResources []*construct.Resource
		initialState       []any
		step               knowledgebase.OperationalStep
		numNeeded          int
		want               graphtest.GraphChanges
	}{
		{
			name: "does nothing if no available resources",
			resource: &construct.Resource{ID: construct.ResourceId{
				Type: "test",
				Name: "test1",
			}},
			numNeeded: 1,
		},
		{
			name:     "places if one available resource",
			resource: &construct.Resource{ID: graphtest.ParseId(t, "test:test:test1")},
			availableResources: []*construct.Resource{
				{ID: graphtest.ParseId(t, "test:test:test2")},
			},
			initialState: []any{
				"test:test:test1",
				"test:test:test2",
			},
			numNeeded: 1,
			want: graphtest.GraphChanges{
				AddedEdges: []construct.Edge{
					{
						Source: graphtest.ParseId(t, "test:test:test1"),
						Target: graphtest.ParseId(t, "test:test:test2"),
					},
				},
			},
		},
		{
			name:     "no resources placed yet, places in first resource",
			resource: &construct.Resource{ID: graphtest.ParseId(t, "test:test:test1")},
			availableResources: []*construct.Resource{
				{ID: graphtest.ParseId(t, "test:parent:test2")},
				{ID: graphtest.ParseId(t, "test:parent:test3")},
			},
			initialState: []any{
				"test:test:test1",
				"test:parent:test2",
				"test:parent:test3",
			},
			step: knowledgebase.OperationalStep{
				Direction: knowledgebase.DirectionDownstream,
			},
			numNeeded: 1,
			want: graphtest.GraphChanges{
				AddedEdges: []construct.Edge{
					{
						Source: graphtest.ParseId(t, "test:test:test1"),
						Target: graphtest.ParseId(t, "test:parent:test2"),
					},
				},
			},
		},
		{
			name:     "chooses placement in spot with most dependencies",
			resource: &construct.Resource{ID: graphtest.ParseId(t, "test:test:test1")},
			availableResources: []*construct.Resource{
				{ID: graphtest.ParseId(t, "test:parent:test2")},
				{ID: graphtest.ParseId(t, "test:parent:test3")},
			},
			initialState: []any{
				"test:test:test1",
				"test:test:test2",
				"test:parent:test2",
				"test:parent:test3",
				"test:test:test2 -> test:parent:test2",
			},
			step: knowledgebase.OperationalStep{
				Direction: knowledgebase.DirectionDownstream,
			},
			numNeeded: 1,
			want: graphtest.GraphChanges{
				AddedEdges: []construct.Edge{
					{
						Source: graphtest.ParseId(t, "test:test:test1"),
						Target: graphtest.ParseId(t, "test:parent:test2"),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			p := &ClusterPlacer{}
			testSol := enginetesting.NewTestSolution()
			testSol.KB.On("GetResourceTemplate", mock.Anything).Return(&knowledgebase.ResourceTemplate{}, nil)
			testSol.KB.On("GetEdgeTemplate", mock.Anything, mock.Anything).Return(&knowledgebase.EdgeTemplate{})
			testSol.LoadState(t, tt.initialState...)
			p.SetCtx(OperationalRuleContext{
				Solution: testSol,
			})
			err := p.PlaceResources(tt.resource, tt.step, tt.availableResources, &tt.numNeeded)
			if !assert.NoError(err) {
				return
			}
			tt.want.AssertEqual(t, testSol.DataflowChanges())
		})
	}
}

func Test_ClosestPlacer(t *testing.T) {
	tests := []struct {
		name               string
		resource           *construct.Resource
		availableResources []*construct.Resource
		initialState       []any
		step               knowledgebase.OperationalStep
		numNeeded          int
		mockKB             []mock.Call
		want               graphtest.GraphChanges
	}{
		{
			name:      "does nothing if no available resources",
			resource:  &construct.Resource{ID: graphtest.ParseId(t, "test:test:test1")},
			numNeeded: 1,
		},
		{
			name:     "places if one available resource",
			resource: &construct.Resource{ID: graphtest.ParseId(t, "test:test:test1")},
			availableResources: []*construct.Resource{
				{ID: graphtest.ParseId(t, "test:test:test2")},
			},
			initialState: []any{
				"test:test:test1",
				"test:test:test2",
			},
			numNeeded: 1,
			want: graphtest.GraphChanges{
				AddedEdges: []construct.Edge{
					{
						Source: graphtest.ParseId(t, "test:test:test1"),
						Target: graphtest.ParseId(t, "test:test:test2"),
					},
				},
			},
		},
		{
			name:     "no resources placed yet, places in first resource",
			resource: &construct.Resource{ID: graphtest.ParseId(t, "test:test:test1")},
			availableResources: []*construct.Resource{
				{ID: graphtest.ParseId(t, "test:parent:test2")},
				{ID: graphtest.ParseId(t, "test:parent:test3")},
			},
			initialState: []any{
				"test:test:test1",
				"test:parent:test2",
				"test:parent:test3",
			},
			step: knowledgebase.OperationalStep{
				Direction: knowledgebase.DirectionDownstream,
			},
			numNeeded: 1,
			want: graphtest.GraphChanges{
				AddedEdges: []construct.Edge{
					{
						Source: graphtest.ParseId(t, "test:test:test1"),
						Target: graphtest.ParseId(t, "test:parent:test2"),
					},
				},
			},
		},
		{
			name:     "chooses placement by first resource if tie in closest",
			resource: &construct.Resource{ID: graphtest.ParseId(t, "test:test:test1")},
			availableResources: []*construct.Resource{
				{ID: graphtest.ParseId(t, "test:parent:test2")},
				{ID: graphtest.ParseId(t, "test:parent:test3")},
			},
			initialState: []any{
				"test:test:test1",
				"test:path:path1",
				"test:parent:test2",
				"test:parent:test3",
				"test:test:test1 -> test:path:path1",
				"test:path:path1 -> test:parent:test2",
				"test:path:path1 -> test:parent:test3",
			},
			step: knowledgebase.OperationalStep{
				Direction: knowledgebase.DirectionDownstream,
			},
			numNeeded: 1,
			want: graphtest.GraphChanges{
				AddedEdges: []construct.Edge{
					{
						Source: graphtest.ParseId(t, "test:test:test1"),
						Target: graphtest.ParseId(t, "test:parent:test2"),
					},
				},
			},
		},
		{
			name:     "chooses placement by closest",
			resource: &construct.Resource{ID: graphtest.ParseId(t, "test:test:test1")},
			availableResources: []*construct.Resource{
				{ID: graphtest.ParseId(t, "test:parent:test2")},
				{ID: graphtest.ParseId(t, "test:parent:test3")},
			},
			initialState: []any{
				"test:test:test1",
				"test:path:path1",
				"test:path:path2",
				"test:parent:test2",
				"test:parent:test3",
				"test:test:test1 -> test:path:path1",
				"test:path:path1 -> test:parent:test3",
				"test:path:path1 -> test:path:path2",
				"test:path:path2 -> test:parent:test2",
			},
			step: knowledgebase.OperationalStep{
				Direction: knowledgebase.DirectionDownstream,
			},
			numNeeded: 1,
			want: graphtest.GraphChanges{
				AddedEdges: []construct.Edge{
					{
						Source: graphtest.ParseId(t, "test:test:test1"),
						Target: graphtest.ParseId(t, "test:parent:test3"),
					},
				},
			},
		},
		{
			name:     "chooses placement by closest with functionality being taken into account",
			resource: &construct.Resource{ID: graphtest.ParseId(t, "test:test:test1")},
			availableResources: []*construct.Resource{
				{ID: graphtest.ParseId(t, "test:parent:test2")},
				{ID: graphtest.ParseId(t, "test:parent:test3")},
			},
			initialState: []any{
				"test:test:test1",
				"test:path:path1",
				"test:path2:path2",
				"test:path3:path3",
				"test:parent:test2",
				"test:parent:test3",
				"test:test:test1 -> test:path:path1",
				"test:test:test1 -> test:path3:path3",
				"test:path:path1 -> test:parent:test3",
				"test:path3:path3 -> test:path2:path2",
				"test:path2:path2 -> test:parent:test2",
			},
			step: knowledgebase.OperationalStep{
				Direction: knowledgebase.DirectionDownstream,
			},
			numNeeded: 1,
			mockKB: []mock.Call{
				{
					Method: "GetResourceTemplate",
					Arguments: mock.Arguments{
						graphtest.ParseId(t, "test:test:test1"),
					},
					ReturnArguments: mock.Arguments{
						&knowledgebase.ResourceTemplate{},
						nil,
					},
				},
				{
					Method: "GetResourceTemplate",
					Arguments: mock.Arguments{
						mock.MatchedBy(graphtest.ParseId(t, "test:parent").Matches),
					},
					ReturnArguments: mock.Arguments{
						&knowledgebase.ResourceTemplate{},
						nil,
					},
				},
				{
					Method: "GetResourceTemplate",
					Arguments: mock.Arguments{
						graphtest.ParseId(t, "test:path:path1"),
					},
					ReturnArguments: mock.Arguments{
						&knowledgebase.ResourceTemplate{
							Classification: knowledgebase.Classification{Is: []string{"compute"}},
						},
						nil,
					},
				},
				{
					Method: "GetResourceTemplate",
					Arguments: mock.Arguments{
						graphtest.ParseId(t, "test:path2:path2"),
					},
					ReturnArguments: mock.Arguments{
						&knowledgebase.ResourceTemplate{},
						nil,
					},
				},
				{
					Method: "GetResourceTemplate",
					Arguments: mock.Arguments{
						graphtest.ParseId(t, "test:path3:path3"),
					},
					ReturnArguments: mock.Arguments{
						&knowledgebase.ResourceTemplate{},
						nil,
					},
				},
			},
			want: graphtest.GraphChanges{
				AddedEdges: []construct.Edge{
					{
						Source: graphtest.ParseId(t, "test:test:test1"),
						Target: graphtest.ParseId(t, "test:parent:test2"),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			p := &ClosestPlacer{}
			testSol := enginetesting.NewTestSolution()
			testSol.KB.On("GetEdgeTemplate", mock.Anything, mock.Anything).Return(&knowledgebase.EdgeTemplate{})
			if len(tt.mockKB) == 0 {
				testSol.KB.On("GetResourceTemplate", mock.Anything).Return(&knowledgebase.ResourceTemplate{}, nil)
			}
			for _, call := range tt.mockKB {
				testSol.KB.On(call.Method, call.Arguments...).Return(call.ReturnArguments...)
			}
			testSol.LoadState(t, tt.initialState...)
			p.SetCtx(OperationalRuleContext{
				Solution: testSol,
			})
			err := p.PlaceResources(tt.resource, tt.step, tt.availableResources, &tt.numNeeded)
			if !assert.NoError(err) {
				return
			}
			tt.want.AssertEqual(t, testSol.DataflowChanges())
		})
	}
}
