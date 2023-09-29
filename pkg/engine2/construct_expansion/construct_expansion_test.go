package constructexpansion

import (
	"testing"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/enginetesting"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_findPossibleExpansions(t *testing.T) {
	tests := []struct {
		name             string
		construct        *construct.Resource
		constructType    string
		attributes       []string
		mockCalls        []mock.Call
		listResourceCall mock.Call
		want             []ExpansionSolution
	}{
		{
			name: "can expand a single construct - gets all options",
			construct: &construct.Resource{
				ID: construct.ResourceId{
					Name: "test",
				},
				Properties: map[string]interface{}(nil),
			},
			mockCalls: []mock.Call{
				{
					Method: "GetFunctionality",
					Arguments: mock.Arguments{
						construct.ResourceId{
							Name: "test",
						},
					},
					ReturnArguments: mock.Arguments{
						knowledgebase.Compute,
						nil,
					},
				},
			},
			listResourceCall: mocklistResourcesCall,
			want: []ExpansionSolution{
				{
					DirectlyMappedResource: construct.ResourceId{
						Provider: "mock",
						Type:     "resource1",
						Name:     "test",
					},
				},
				{
					DirectlyMappedResource: construct.ResourceId{
						Provider: "mock",
						Type:     "resource2",
						Name:     "test",
					},
				},
			},
		},
		{
			name: "can expand a single construct uses constraint type to get single option",
			construct: &construct.Resource{
				ID: construct.ResourceId{
					Name: "test",
				},
				Properties: map[string]interface{}(nil),
			},
			constructType: "mock:resource1",
			mockCalls: []mock.Call{
				{
					Method: "GetFunctionality",
					Arguments: mock.Arguments{
						construct.ResourceId{
							Name: "test",
						},
					},
					ReturnArguments: mock.Arguments{
						knowledgebase.Compute,
						nil,
					},
				},
			},
			listResourceCall: mocklistResourcesCall,
			want: []ExpansionSolution{
				{
					DirectlyMappedResource: construct.ResourceId{
						Provider: "mock",
						Type:     "resource1",
						Name:     "test",
					},
				},
			},
		},
		{
			name: "can expand a single construct using attributes to get single option",
			construct: &construct.Resource{
				ID: construct.ResourceId{
					Name: "test",
				},
				Properties: map[string]interface{}(nil),
			},
			attributes: []string{"serverless"},
			mockCalls: []mock.Call{
				{
					Method: "GetFunctionality",
					Arguments: mock.Arguments{
						construct.ResourceId{
							Name: "test",
						},
					},
					ReturnArguments: mock.Arguments{
						knowledgebase.Compute,
						nil,
					},
				},
				{
					Method: "HasFunctionalPath",
					Arguments: mock.Arguments{
						construct.ResourceId{Provider: "mock", Type: "resource1", Name: "test"},
						construct.ResourceId{Provider: "mock", Type: "resource2"},
					},
					ReturnArguments: mock.Arguments{false},
				},
			},
			listResourceCall: mocklistResourcesCall,
			want: []ExpansionSolution{
				{
					DirectlyMappedResource: construct.ResourceId{
						Provider: "mock",
						Type:     "resource2",
						Name:     "test",
					},
				},
			},
		},
		{
			name: "can expand using path(gives) attributes",
			construct: &construct.Resource{
				ID: construct.ResourceId{
					Name: "test",
				},
				Properties: map[string]interface{}(nil),
			},
			attributes: []string{"reliability"},
			mockCalls: []mock.Call{
				{
					Method: "GetFunctionality",
					Arguments: mock.Arguments{
						construct.ResourceId{
							Name: "test",
						},
					},
					ReturnArguments: mock.Arguments{
						knowledgebase.Storage,
						nil,
					},
				},
				{
					Method: "HasFunctionalPath",
					Arguments: mock.Arguments{
						construct.ResourceId{Provider: "mock", Type: "resource3", Name: "test"},
						construct.ResourceId{Provider: "mock", Type: "resource1"},
					},
					ReturnArguments: mock.Arguments{true},
				},
				{
					Method: "HasFunctionalPath",
					Arguments: mock.Arguments{
						construct.ResourceId{Provider: "mock", Type: "resource3", Name: "test"},
						construct.ResourceId{Provider: "mock", Type: "resource2"},
					},
					ReturnArguments: mock.Arguments{true},
				},
			},
			listResourceCall: mocklistResourcesCall2,
			want: []ExpansionSolution{
				{
					Edges: []graph.Edge[construct.Resource]{
						{
							Source: construct.Resource{
								ID: construct.ResourceId{
									Provider: "mock",
									Type:     "resource3",
									Name:     "test",
								},
								Properties: make(construct.Properties),
							},

							Target: construct.Resource{
								ID: construct.ResourceId{
									Provider: "mock",
									Type:     "resource2",
									Name:     "test",
								},
								Properties: make(construct.Properties),
							},
						},
					},
					DirectlyMappedResource: construct.ResourceId{
						Provider: "mock",
						Type:     "resource3",
						Name:     "test",
					},
				},
			},
		},
		{
			name: "fails when no matches",
			construct: &construct.Resource{
				ID: construct.ResourceId{
					Name: "test",
				},
				Properties: map[string]interface{}(nil),
			},
			attributes: []string{"reliability"},
			mockCalls: []mock.Call{
				{
					Method: "GetFunctionality",
					Arguments: mock.Arguments{
						construct.ResourceId{
							Name: "test",
						},
					},
					ReturnArguments: mock.Arguments{
						knowledgebase.Api,
						nil,
					},
				},
			},
			listResourceCall: mocklistResourcesCall2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			mockKB := &enginetesting.MockKB{}
			mockKB.On(tt.listResourceCall.Method, tt.listResourceCall.Arguments...).Return(tt.listResourceCall.ReturnArguments...)
			for _, call := range tt.mockCalls {
				mockKB.On(call.Method, call.Arguments...).Return(call.ReturnArguments...).Once()
			}
			ctx := ConstructExpansionContext{
				Construct: tt.construct,
				Kb:        mockKB,
			}
			got, err := ctx.findPossibleExpansions(ExpansionSet{
				Construct:  tt.construct,
				Attributes: tt.attributes,
			}, tt.constructType)
			if tt.want == nil {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.want, got)
			mockKB.AssertExpectations(t)
		})
	}
}

var mocklistResourcesCall = mock.Call{
	Method:    "ListResources",
	Arguments: mock.Arguments{},
	ReturnArguments: mock.Arguments{
		[]*knowledgebase.ResourceTemplate{
			createResourceTemplate(construct.ResourceId{Provider: "mock", Type: "resource1"}, knowledgebase.Classification{
				Is: []string{string(knowledgebase.Compute)},
			}),
			createResourceTemplate(construct.ResourceId{Provider: "mock", Type: "resource2"}, knowledgebase.Classification{
				Is: []string{string(knowledgebase.Compute), "serverless"},
				Gives: []knowledgebase.Gives{
					{
						Attribute:     "reliability",
						Functionality: []string{"*"},
					},
				},
			}),
		},
		nil,
	},
}

var mocklistResourcesCall2 = mock.Call{
	Method:    "ListResources",
	Arguments: mock.Arguments{},
	ReturnArguments: mock.Arguments{
		[]*knowledgebase.ResourceTemplate{
			createResourceTemplate(construct.ResourceId{Provider: "mock", Type: "resource1"}, knowledgebase.Classification{
				Is: []string{string(knowledgebase.Compute)},
			}),
			createResourceTemplate(construct.ResourceId{Provider: "mock", Type: "resource2"}, knowledgebase.Classification{
				Is: []string{string(knowledgebase.Compute), "serverless"},
				Gives: []knowledgebase.Gives{
					{
						Attribute:     "reliability",
						Functionality: []string{"*"},
					},
				},
			}),
			createResourceTemplate(construct.ResourceId{Provider: "mock", Type: "resource3"}, knowledgebase.Classification{
				Is: []string{string(knowledgebase.Storage)},
			}),
		},
		nil,
	},
}

func createResourceTemplate(id construct.ResourceId, classifications knowledgebase.Classification) *knowledgebase.ResourceTemplate {
	return &knowledgebase.ResourceTemplate{
		QualifiedTypeName: id.QualifiedTypeName(),
		Classification:    classifications,
	}
}
