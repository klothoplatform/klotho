package properties

import (
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/enginetesting"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledgebase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_ValidatePropertyRef(t *testing.T) {
	tests := []struct {
		name          string
		propertyType  string
		ref           construct.PropertyRef
		testResources []*construct.Resource
		mockKBCalls   []mock.Call
		expect        any
		wantErr       bool
	}{
		{
			name:         "string property ref, existing resource with value",
			propertyType: "string",
			ref: construct.PropertyRef{
				Resource: construct.ResourceId{Name: "test"},
				Property: "test",
			},
			testResources: []*construct.Resource{
				{
					ID: construct.ResourceId{Name: "test"},
					Properties: map[string]any{
						"test": "testval",
					},
				},
			},
			mockKBCalls: []mock.Call{
				{
					Method: "GetResourceTemplate",
					Arguments: mock.Arguments{
						construct.ResourceId{Name: "test"},
					},
					ReturnArguments: mock.Arguments{
						&knowledgebase.ResourceTemplate{
							Properties: knowledgebase.Properties{
								"test": &StringProperty{PropertyDetails: knowledgebase.PropertyDetails{Path: "test", Name: "test"}},
							},
						}, nil,
					},
				},
			},
			expect: "testval",
		},
		{
			name:         "string property is deploy time returns nil",
			propertyType: "string",
			ref: construct.PropertyRef{
				Resource: construct.ResourceId{Name: "test"},
				Property: "test",
			},
			testResources: []*construct.Resource{
				{
					ID: construct.ResourceId{Name: "test"},
					Properties: map[string]any{
						"test": "testval",
					},
				},
			},
			mockKBCalls: []mock.Call{
				{
					Method: "GetResourceTemplate",
					Arguments: mock.Arguments{
						construct.ResourceId{Name: "test"},
					},
					ReturnArguments: mock.Arguments{
						&knowledgebase.ResourceTemplate{
							Properties: knowledgebase.Properties{
								"test": &StringProperty{PropertyDetails: knowledgebase.PropertyDetails{
									Path:       "test",
									Name:       "test",
									DeployTime: true,
								}},
							},
						}, nil,
					},
				},
			},
		},
		{
			name: "nested property ref",
			ref: construct.PropertyRef{
				Resource: construct.ResourceId{Name: "test"},
				Property: "test",
			},
			propertyType: "string",
			testResources: []*construct.Resource{
				{
					ID: construct.ResourceId{Name: "test"},
					Properties: map[string]any{
						"test": construct.PropertyRef{
							Resource: construct.ResourceId{Name: "test2"},
							Property: "test2",
						},
					},
				},
				{
					ID: construct.ResourceId{Name: "test2"},
					Properties: map[string]any{
						"test2": "testval",
					},
				},
			},
			mockKBCalls: []mock.Call{
				{
					Method: "GetResourceTemplate",
					Arguments: mock.Arguments{
						construct.ResourceId{Name: "test"},
					},
					ReturnArguments: mock.Arguments{
						&knowledgebase.ResourceTemplate{
							Properties: knowledgebase.Properties{
								"test": &StringProperty{PropertyDetails: knowledgebase.PropertyDetails{Path: "test", Name: "test"}},
							},
						}, nil,
					},
				},
				{
					Method: "GetResourceTemplate",
					Arguments: mock.Arguments{
						construct.ResourceId{Name: "test2"},
					},
					ReturnArguments: mock.Arguments{
						&knowledgebase.ResourceTemplate{
							Properties: knowledgebase.Properties{
								"test2": &StringProperty{PropertyDetails: knowledgebase.PropertyDetails{Path: "test2", Name: "test2"}},
							},
						}, nil,
					},
				},
			},
			expect: "testval",
		},
		{
			name:         "no resource, throws err",
			propertyType: "string",
			ref: construct.PropertyRef{
				Resource: construct.ResourceId{Name: "test"},
				Property: "test",
			},
			mockKBCalls: []mock.Call{
				{
					Method: "GetResourceTemplate",
					Arguments: mock.Arguments{
						construct.ResourceId{Name: "test"},
					},
					ReturnArguments: mock.Arguments{
						&knowledgebase.ResourceTemplate{
							Properties: knowledgebase.Properties{
								"test": &StringProperty{PropertyDetails: knowledgebase.PropertyDetails{
									Path:       "test",
									Name:       "test",
									DeployTime: true,
								}},
							},
						}, nil,
					},
				},
			},
			wantErr: true,
		},
		{
			name:         "no property throws error",
			propertyType: "string",
			ref: construct.PropertyRef{
				Resource: construct.ResourceId{Name: "test"},
				Property: "test",
			},
			testResources: []*construct.Resource{
				{
					ID: construct.ResourceId{Name: "test"},
					Properties: map[string]any{
						"test": "testval",
					},
				},
			},
			mockKBCalls: []mock.Call{
				{
					Method: "GetResourceTemplate",
					Arguments: mock.Arguments{
						construct.ResourceId{Name: "test"},
					},
					ReturnArguments: mock.Arguments{
						&knowledgebase.ResourceTemplate{
							Properties: knowledgebase.Properties{},
						}, nil,
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			graph := construct.NewGraph()
			for _, r := range tt.testResources {
				graph.AddVertex(r)
			}
			mockKB := &enginetesting.MockKB{}
			for _, call := range tt.mockKBCalls {
				mockKB.On(call.Method, call.Arguments...).Return(call.ReturnArguments...)
			}
			ctx := knowledgebase.DynamicValueContext{
				Graph:         graph,
				KnowledgeBase: mockKB,
			}
			val, err := ValidatePropertyRef(tt.ref, tt.propertyType, ctx)
			if tt.wantErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
				assert.Equal(tt.expect, val)
			}
		})
	}
}
