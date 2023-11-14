package operational_rule

import (
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/construct2/graphtest"
	"github.com/klothoplatform/klotho/pkg/engine2/enginetesting"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_handleOperationalResourceAction(t *testing.T) {
	tests := []struct {
		name         string
		action       operationalResourceAction
		resource     *construct.Resource
		initialState []any
		want         operationalResourceAction
		wantGraphs   enginetesting.ExpectedGraphs
		wantErr      bool
	}{
		{
			name: "creates explicit and non unique resource",
			action: operationalResourceAction{
				Step: knowledgebase.OperationalStep{
					Resources: []knowledgebase.ResourceSelector{
						{
							Selector: "test:test:myname",
						},
						{
							Selector: "test:test",
						},
					},
					Direction: knowledgebase.DirectionDownstream,
				},
				numNeeded: 2,
				result:    Result{},
			},
			resource:     construct.CreateResource(graphtest.ParseId(t, "test:step:test")),
			initialState: []any{"test:step:test"},
			want: operationalResourceAction{
				numNeeded: 0,
				result: Result{
					CreatedResources: []*construct.Resource{
						construct.CreateResource(graphtest.ParseId(t, "test:test:myname")),
						construct.CreateResource(graphtest.ParseId(t, "test:test:test-1")),
					},
					AddedDependencies: []construct.Edge{
						{
							Source: graphtest.ParseId(t, "test:step:test"),
							Target: graphtest.ParseId(t, "test:test:myname"),
						},
						{
							Source: graphtest.ParseId(t, "test:step:test"),
							Target: graphtest.ParseId(t, "test:test:test-1"),
						},
					},
				},
			},
			wantGraphs: enginetesting.ExpectedGraphs{
				Dataflow: []any{"test:step:test", "test:test:myname", "test:test:test-1",
					"test:step:test -> test:test:myname", "test:step:test -> test:test:test-1"},
				Deployment: []any{"test:step:test", "test:test:myname", "test:test:test-1",
					"test:step:test -> test:test:myname", "test:step:test -> test:test:test-1"},
			},
		},
		{
			name: "creates unique resource",
			action: operationalResourceAction{
				Step: knowledgebase.OperationalStep{
					Resources: []knowledgebase.ResourceSelector{
						{
							Selector: "test:test",
						},
					},
					Unique:    true,
					Direction: knowledgebase.DirectionDownstream,
				},
				numNeeded: 1,
				result:    Result{},
			},
			resource:     construct.CreateResource(graphtest.ParseId(t, "test:step:test")),
			initialState: []any{"test:step:test"},
			want: operationalResourceAction{
				numNeeded: 0,
				result: Result{
					CreatedResources: []*construct.Resource{
						construct.CreateResource(graphtest.ParseId(t, "test:test:test-test-0")),
					},
					AddedDependencies: []construct.Edge{
						{
							Source: graphtest.ParseId(t, "test:step:test"),
							Target: graphtest.ParseId(t, "test:test:test-test-0"),
						},
					},
				},
			},
			wantGraphs: enginetesting.ExpectedGraphs{
				Dataflow:   []any{"test:step:test", "test:test:test-test-0", "test:step:test -> test:test:test-test-0"},
				Deployment: []any{"test:step:test", "test:test:test-test-0", "test:step:test -> test:test:test-test-0"},
			},
		},
		{
			name: "uses available resource",
			action: operationalResourceAction{
				Step: knowledgebase.OperationalStep{
					Resources: []knowledgebase.ResourceSelector{
						{
							Selector: "test:test",
						},
					},
					Direction: knowledgebase.DirectionDownstream,
				},
				numNeeded: 1,
				result:    Result{},
			},
			resource:     construct.CreateResource(graphtest.ParseId(t, "test:step:test")),
			initialState: []any{"test:step:test", "test:test:test"},
			want: operationalResourceAction{
				numNeeded: 0,
				result: Result{
					AddedDependencies: []construct.Edge{
						{
							Source: graphtest.ParseId(t, "test:step:test"),
							Target: graphtest.ParseId(t, "test:test:test"),
						},
					},
				},
			},
			wantGraphs: enginetesting.ExpectedGraphs{
				Dataflow:   []any{"test:step:test", "test:test:test", "test:step:test -> test:test:test"},
				Deployment: []any{"test:step:test", "test:test:test", "test:step:test -> test:test:test"},
			},
		},
		{
			name: "fails if it cant create non unique resource",
			action: operationalResourceAction{
				Step: knowledgebase.OperationalStep{
					Resources: []knowledgebase.ResourceSelector{
						{
							Selector: "test:test:myname",
						},
					},
					Direction: knowledgebase.DirectionDownstream,
				},
				numNeeded: 2,
				result:    Result{},
			},
			resource:     construct.CreateResource(graphtest.ParseId(t, "test:step:test")),
			initialState: []any{"test:step:test"},
			want: operationalResourceAction{
				numNeeded: 1,
				result: Result{
					CreatedResources: []*construct.Resource{
						construct.CreateResource(graphtest.ParseId(t, "test:test:myname")),
					},
					AddedDependencies: []construct.Edge{
						{
							Source: graphtest.ParseId(t, "test:step:test"),
							Target: graphtest.ParseId(t, "test:test:myname"),
						},
					},
				},
			},
			wantGraphs: enginetesting.ExpectedGraphs{
				Dataflow:   []any{"test:step:test", "test:test:myname", "test:step:test -> test:test:myname"},
				Deployment: []any{"test:step:test", "test:test:myname", "test:step:test -> test:test:myname"},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			ctx := enginetesting.NewTestSolution()
			ctx.KB.On("HasFunctionalPath", mock.Anything, mock.Anything).Return(true, nil)
			ctx.KB.On("GetAllowedNamespacedResourceIds", mock.Anything, mock.Anything).Return([]construct.ResourceId{}, nil)
			ctx.KB.On("GetResourceTemplate", mock.Anything).Return(&knowledgebase.ResourceTemplate{}, nil)
			ctx.KB.On("GetEdgeTemplate", mock.Anything, mock.Anything).Return(&knowledgebase.EdgeTemplate{}, nil)
			ctx.LoadState(t, tt.initialState...)
			tt.action.ruleCtx = OperationalRuleContext{Solution: ctx}
			err := tt.action.handleOperationalResourceAction(tt.resource)
			if tt.wantErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
			assert.Equal(tt.want.numNeeded, tt.action.numNeeded)
			assert.Equal(tt.action.result, tt.want.result)
			tt.wantGraphs.AssertEqual(t, ctx)
		})
	}
}

func Test_handleExplicitResources(t *testing.T) {
	tests := []struct {
		name         string
		action       operationalResourceAction
		resource     *construct.Resource
		initialState []any
		want         operationalResourceAction
	}{
		{
			name: "single explicit resource",
			action: operationalResourceAction{
				Step: knowledgebase.OperationalStep{
					Resources: []knowledgebase.ResourceSelector{
						{
							Selector: "test:test:myname",
						},
					},
					Direction: knowledgebase.DirectionDownstream,
				},
				numNeeded: 2,
				result:    Result{},
			},
			resource:     construct.CreateResource(graphtest.ParseId(t, "test:step:test")),
			initialState: []any{"test:step:test"},
			want: operationalResourceAction{
				numNeeded: 1,
				result: Result{
					CreatedResources: []*construct.Resource{
						construct.CreateResource(graphtest.ParseId(t, "test:test:myname")),
					},
					AddedDependencies: []construct.Edge{
						{
							Source: graphtest.ParseId(t, "test:step:test"),
							Target: graphtest.ParseId(t, "test:test:myname"),
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			ctx := enginetesting.NewTestSolution()
			ctx.KB.On("GetResourceTemplate", mock.Anything).Return(&knowledgebase.ResourceTemplate{}, nil)
			ctx.KB.On("GetEdgeTemplate", mock.Anything, mock.Anything).Return(&knowledgebase.EdgeTemplate{}, nil)
			ctx.LoadState(t, tt.initialState...)
			tt.action.ruleCtx = OperationalRuleContext{Solution: ctx}
			err := tt.action.handleExplicitResources(tt.resource)
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.want.numNeeded, tt.action.numNeeded)
			assert.Equal(tt.action.result, tt.want.result)
		})
	}
}

func Test_createUniqueResources(t *testing.T) {
	tests := []struct {
		name         string
		action       operationalResourceAction
		resource     *construct.Resource
		initialState []any
		want         operationalResourceAction
	}{
		{
			name: "multiple unique",
			action: operationalResourceAction{
				Step: knowledgebase.OperationalStep{
					Resources: []knowledgebase.ResourceSelector{
						{
							Selector: "test:test",
						},
					},
					Direction: knowledgebase.DirectionDownstream,
				},
				numNeeded: 2,
				result:    Result{},
			},
			resource:     construct.CreateResource(graphtest.ParseId(t, "test:step:test")),
			initialState: []any{"test:step:test"},
			want: operationalResourceAction{
				numNeeded: 0,
				result: Result{
					CreatedResources: []*construct.Resource{
						construct.CreateResource(graphtest.ParseId(t, "test:test:test-0")),
						construct.CreateResource(graphtest.ParseId(t, "test:test:test-1")),
					},
					AddedDependencies: []construct.Edge{
						{
							Source: graphtest.ParseId(t, "test:step:test"),
							Target: graphtest.ParseId(t, "test:test:test-0"),
						},
						{
							Source: graphtest.ParseId(t, "test:step:test"),
							Target: graphtest.ParseId(t, "test:test:test-1"),
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			ctx := enginetesting.NewTestSolution()
			ctx.KB.On("GetResourceTemplate", mock.Anything).Return(&knowledgebase.ResourceTemplate{}, nil)
			ctx.KB.On("GetEdgeTemplate", mock.Anything, mock.Anything).Return(&knowledgebase.EdgeTemplate{}, nil)
			ctx.LoadState(t, tt.initialState...)
			tt.action.ruleCtx = OperationalRuleContext{Solution: ctx}
			err := tt.action.createUniqueResources(tt.resource)
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.want.numNeeded, tt.action.numNeeded)
			assert.Equal(tt.action.result, tt.want.result)
		})
	}
}

func Test_useAvailableResources(t *testing.T) {
	tests := []struct {
		name         string
		action       operationalResourceAction
		resource     *construct.Resource
		initialState []any
		want         operationalResourceAction
	}{
		{
			name: "single available",
			action: operationalResourceAction{
				Step: knowledgebase.OperationalStep{
					Resources: []knowledgebase.ResourceSelector{
						{
							Selector: "test:test",
						},
					},
					Direction: knowledgebase.DirectionDownstream,
				},
				numNeeded: 1,
				result:    Result{},
			},
			resource:     construct.CreateResource(graphtest.ParseId(t, "test:step:test")),
			initialState: []any{"test:step:test", "test:test:test-0"},
			want: operationalResourceAction{
				numNeeded: 0,
				result: Result{
					AddedDependencies: []construct.Edge{
						{
							Source: graphtest.ParseId(t, "test:step:test"),
							Target: graphtest.ParseId(t, "test:test:test-0"),
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			ctx := enginetesting.NewTestSolution()
			ctx.KB.On("HasFunctionalPath", mock.Anything, mock.Anything).Return(true, nil)
			ctx.KB.On("GetAllowedNamespacedResourceIds", mock.Anything, mock.Anything).Return([]construct.ResourceId{}, nil)
			ctx.KB.On("GetResourceTemplate", mock.Anything).Return(&knowledgebase.ResourceTemplate{}, nil)
			ctx.KB.On("GetEdgeTemplate", mock.Anything, mock.Anything).Return(&knowledgebase.EdgeTemplate{}, nil)
			ctx.LoadState(t, tt.initialState...)
			tt.action.ruleCtx = OperationalRuleContext{Solution: ctx}
			err := tt.action.useAvailableResources(tt.resource)
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.want.numNeeded, tt.action.numNeeded)
			assert.Equal(tt.action.result, tt.want.result)
		})
	}
}
