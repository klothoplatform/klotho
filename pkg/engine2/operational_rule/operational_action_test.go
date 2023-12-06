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
		want         enginetesting.ExpectedGraphs
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
			},
			resource:     &construct.Resource{ID: graphtest.ParseId(t, "test:step:test")},
			initialState: []any{"test:step:test"},
			want: enginetesting.ExpectedGraphs{
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
			},
			resource:     &construct.Resource{ID: graphtest.ParseId(t, "test:step:resource")},
			initialState: []any{"test:step:resource"},
			want: enginetesting.ExpectedGraphs{
				Dataflow: []any{"test:step:resource -> test:test:test-resource"},
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
			},
			resource:     &construct.Resource{ID: graphtest.ParseId(t, "test:step:test")},
			initialState: []any{"test:step:test", "test:test:test"},
			want: enginetesting.ExpectedGraphs{
				Dataflow: []any{"test:step:test", "test:test:test", "test:step:test -> test:test:test"},
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
			},
			resource:     &construct.Resource{ID: graphtest.ParseId(t, "test:step:test")},
			initialState: []any{"test:step:test"},
			want: enginetesting.ExpectedGraphs{
				Dataflow: []any{"test:step:test", "test:test:myname", "test:step:test -> test:test:myname"},
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
			tt.want.AssertEqual(t, ctx)
		})
	}
}

func Test_createUniqueResources(t *testing.T) {
	tests := []struct {
		name         string
		action       operationalResourceAction
		resource     *construct.Resource
		initialState []any
		want         enginetesting.ExpectedGraphs
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
			},
			resource:     &construct.Resource{ID: graphtest.ParseId(t, "test:step:test")},
			initialState: []any{"test:step:test"},
			want: enginetesting.ExpectedGraphs{
				Dataflow: []any{
					"test:test:test-0", "test:test:test-1",
					"test:step:test -> test:test:test-0", "test:step:test -> test:test:test-1",
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
			tt.want.AssertEqual(t, ctx)
		})
	}
}

func Test_useAvailableResources(t *testing.T) {
	tests := []struct {
		name         string
		action       operationalResourceAction
		resource     *construct.Resource
		initialState []any
		want         enginetesting.ExpectedGraphs
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
			},
			resource:     &construct.Resource{ID: graphtest.ParseId(t, "test:step:test")},
			initialState: []any{"test:step:test", "test:test:test-0"},
			want: enginetesting.ExpectedGraphs{
				Dataflow: []any{"test:step:test", "test:test:test-0", "test:step:test -> test:test:test-0"},
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
			tt.want.AssertEqual(t, ctx)
		})
	}
}
