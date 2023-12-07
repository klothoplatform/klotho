package reconciler

import (
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/enginetesting"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/klothoplatform/klotho/pkg/knowledge_base2/properties"
	"github.com/klothoplatform/klotho/pkg/set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_canDeleteResource(t *testing.T) {
	type args struct {
		resource    construct.ResourceId
		explicit    bool
		template    *knowledgebase.ResourceTemplate
		upstreams   []construct.ResourceId
		downstreams []construct.ResourceId
		mockKBCalls []mock.Call
	}
	tests := []struct {
		name        string
		args        args
		mockKBCalls []mock.Call
		want        bool
		wantErr     bool
	}{
		{
			name: "can delete resource when no upstreams or downstreams",
			args: args{
				resource: construct.ResourceId{},
				explicit: false,
				template: &knowledgebase.ResourceTemplate{},
			},
			want: true,
		},
		{
			name: "can delete if explicit",
			args: args{
				resource: construct.ResourceId{},
				explicit: true,
				template: &knowledgebase.ResourceTemplate{},
				upstreams: []construct.ResourceId{
					{Type: "one"},
				},
				downstreams: []construct.ResourceId{
					{Type: "two"},
				},
			},
			mockKBCalls: []mock.Call{
				{
					Method: "GetEdgeTemplate",
					Arguments: mock.Arguments{
						mock.Anything,
						mock.Anything,
					},
					ReturnArguments: mock.Arguments{
						&knowledgebase.EdgeTemplate{},
					},
				},
			},
			want: true,
		},
		{
			name: "cannot delete if functional and not explicit",
			args: args{
				resource: construct.ResourceId{},
				explicit: false,
				template: &knowledgebase.ResourceTemplate{
					Classification: knowledgebase.Classification{Is: []string{string(knowledgebase.Compute)}},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			upstreamSet := set.Set[construct.ResourceId]{}
			upstreamSet.Add(tt.args.upstreams...)
			downstreamSet := set.Set[construct.ResourceId]{}
			downstreamSet.Add(tt.args.downstreams...)
			sol := &enginetesting.TestSolution{
				KB: enginetesting.MockKB{},
			}
			for _, call := range tt.mockKBCalls {
				sol.KB.On(call.Method, call.Arguments...).Return(call.ReturnArguments...)
			}
			got, err := canDeleteResource(sol, tt.args.resource, tt.args.explicit, tt.args.template, upstreamSet, downstreamSet)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.want, got)
		})
	}
}

func Test_ignoreCriteria(t *testing.T) {
	type args struct {
		resource  construct.ResourceId
		nodes     []construct.ResourceId
		direction knowledgebase.Direction
	}
	tests := []struct {
		name        string
		args        args
		mockKBCalls []mock.Call
		want        bool
	}{
		{
			name: "ignore criteria when no nodes",
			args: args{
				resource:  construct.ResourceId{Type: "one"},
				direction: knowledgebase.DirectionUpstream,
			},
			want: true,
		},
		{
			name: "ignore criteria when single upstream node is deletion dependent",
			args: args{
				resource: construct.ResourceId{Type: "one"},
				nodes: []construct.ResourceId{
					{Type: "two"},
				},
				direction: knowledgebase.DirectionUpstream,
			},
			mockKBCalls: []mock.Call{
				{
					Method: "GetEdgeTemplate",
					Arguments: mock.Arguments{
						construct.ResourceId{Type: "two"},
						construct.ResourceId{Type: "one"},
					},
					ReturnArguments: mock.Arguments{
						&knowledgebase.EdgeTemplate{
							DeletionDependent: true,
						},
					},
				},
			},
			want: true,
		},
		{
			name: "ignore criteria is false when only one upstream node is deletion dependent",
			args: args{
				resource: construct.ResourceId{Type: "one"},
				nodes: []construct.ResourceId{
					{Type: "two"},
					{Type: "three"},
				},
				direction: knowledgebase.DirectionUpstream,
			},
			mockKBCalls: []mock.Call{
				{
					Method: "GetEdgeTemplate",
					Arguments: mock.Arguments{
						construct.ResourceId{Type: "two"},
						construct.ResourceId{Type: "one"},
					},
					ReturnArguments: mock.Arguments{
						&knowledgebase.EdgeTemplate{
							DeletionDependent: true,
						},
					},
				},
				{
					Method: "GetEdgeTemplate",
					Arguments: mock.Arguments{
						construct.ResourceId{Type: "three"},
						construct.ResourceId{Type: "one"},
					},
					ReturnArguments: mock.Arguments{
						&knowledgebase.EdgeTemplate{},
					},
				},
			},
			want: false,
		},
		{
			name: "ignore criteria when single downstream node is deletion dependent",
			args: args{
				resource: construct.ResourceId{Type: "one"},
				nodes: []construct.ResourceId{
					{Type: "two"},
				},
				direction: knowledgebase.DirectionDownstream,
			},
			mockKBCalls: []mock.Call{
				{
					Method: "GetEdgeTemplate",
					Arguments: mock.Arguments{
						construct.ResourceId{Type: "one"},
						construct.ResourceId{Type: "two"},
					},
					ReturnArguments: mock.Arguments{
						&knowledgebase.EdgeTemplate{
							DeletionDependent: true,
						},
					},
				},
			},
			want: true,
		},
		{
			name: "ignore criteria is false when only one downstream node is deletion dependent",
			args: args{
				resource: construct.ResourceId{Type: "one"},
				nodes: []construct.ResourceId{
					{Type: "two"},
					{Type: "three"},
				},
				direction: knowledgebase.DirectionDownstream,
			},
			mockKBCalls: []mock.Call{
				{
					Method: "GetEdgeTemplate",
					Arguments: mock.Arguments{
						construct.ResourceId{Type: "one"},
						construct.ResourceId{Type: "two"},
					},
					ReturnArguments: mock.Arguments{
						&knowledgebase.EdgeTemplate{
							DeletionDependent: true,
						},
					},
				},
				{
					Method: "GetEdgeTemplate",
					Arguments: mock.Arguments{
						construct.ResourceId{Type: "one"},
						construct.ResourceId{Type: "three"},
					},
					ReturnArguments: mock.Arguments{
						&knowledgebase.EdgeTemplate{},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			nodesSet := set.Set[construct.ResourceId]{}
			nodesSet.Add(tt.args.nodes...)
			sol := &enginetesting.TestSolution{
				KB: enginetesting.MockKB{},
			}
			for _, call := range tt.mockKBCalls {
				sol.KB.On(call.Method, call.Arguments...).Return(call.ReturnArguments...)
			}
			got := ignoreCriteria(sol, tt.args.resource, nodesSet, tt.args.direction)
			assert.Equal(tt.want, got)
		})
	}
}

func Test_findAllResourcesInNamespace(t *testing.T) {

	mockCall := mock.Call{
		Method: "GetResourceTemplate",
		Arguments: mock.Arguments{
			mock.Anything,
		},
		ReturnArguments: mock.Arguments{
			&knowledgebase.ResourceTemplate{
				Properties: knowledgebase.Properties{
					"namespaceProp": &properties.ResourceProperty{
						PropertyDetails: knowledgebase.PropertyDetails{
							Path:      "namespaceProp",
							Namespace: true,
						},
					},
				},
			},
			nil,
		},
	}
	type args struct {
		namespace construct.ResourceId
	}
	tests := []struct {
		name        string
		args        args
		resources   []*construct.Resource
		mockKBCalls []mock.Call
		want        []construct.ResourceId
	}{
		{
			name: "nothing if not namespaced",
			args: args{
				namespace: construct.ResourceId{Type: "one", Name: "one"},
			},
			mockKBCalls: []mock.Call{mockCall},
			resources: []*construct.Resource{
				{
					ID: construct.ResourceId{Type: "two"},
					Properties: construct.Properties{
						"namespaceProp": construct.ResourceId{Type: "one"},
					},
				},
			},
			want: []construct.ResourceId{},
		},
		{
			name: "nothing if not namespaced to a different resource",
			args: args{
				namespace: construct.ResourceId{Type: "one", Name: "one"},
			},
			mockKBCalls: []mock.Call{mockCall},
			resources: []*construct.Resource{
				{
					ID: construct.ResourceId{Type: "two", Namespace: "not one"},
					Properties: construct.Properties{
						"namespaceProp": construct.ResourceId{Type: "one"},
					},
				},
			},
			want: []construct.ResourceId{},
		},
		{
			name: "finds matching namespaces by property ref",
			args: args{
				namespace: construct.ResourceId{Type: "one", Name: "one"},
			},
			mockKBCalls: []mock.Call{{
				Method: "GetResourceTemplate",
				Arguments: mock.Arguments{
					mock.Anything,
				},
				ReturnArguments: mock.Arguments{
					&knowledgebase.ResourceTemplate{
						Properties: knowledgebase.Properties{
							"namespaceProp": &properties.StringProperty{
								PropertyDetails: knowledgebase.PropertyDetails{
									Path:      "namespaceProp",
									Namespace: true,
								},
							},
						},
					},
					nil,
				},
			}},
			resources: []*construct.Resource{
				{
					ID: construct.ResourceId{Type: "two", Namespace: "one"},
					Properties: construct.Properties{
						"namespaceProp": "one",
					},
				},
				{
					ID: construct.ResourceId{Type: "three", Namespace: "one"},
					Properties: construct.Properties{
						"namespaceProp": construct.PropertyRef{
							Resource: construct.ResourceId{Type: "one"},
							Property: "name",
						},
					},
				},
			},
			want: []construct.ResourceId{
				{Type: "three", Namespace: "one"},
			},
		},
		{
			name: "finds multiple matching namespace",
			args: args{
				namespace: construct.ResourceId{Type: "one", Name: "one"},
			},
			mockKBCalls: []mock.Call{mockCall},
			resources: []*construct.Resource{
				{
					ID: construct.ResourceId{Type: "two", Namespace: "one"},
					Properties: construct.Properties{
						"namespaceProp": construct.ResourceId{Type: "one"},
					},
				},
				{
					ID: construct.ResourceId{Type: "three", Namespace: "one"},
					Properties: construct.Properties{
						"namespaceProp": construct.ResourceId{Type: "one"},
					},
				},
			},
			want: []construct.ResourceId{
				{Type: "two", Namespace: "one"},
				{Type: "three", Namespace: "one"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			sol := enginetesting.NewTestSolution()
			sol.KB = enginetesting.MockKB{}
			for _, call := range tt.mockKBCalls {
				sol.KB.On(call.Method, call.Arguments...).Return(call.ReturnArguments...)
			}
			for _, resource := range tt.resources {
				err := sol.RawView().AddVertex(resource)
				if !assert.NoError(err) {
					return
				}
			}
			got, err := findAllResourcesInNamespace(sol, tt.args.namespace)
			if !assert.NoError(err) {
				return
			}
			assert.ElementsMatch(tt.want, got.ToSlice())
		})
	}
}
