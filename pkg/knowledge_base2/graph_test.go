package knowledgebase2

import (
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/construct2/graphtest"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestIsOperationalResourceSideEffect(t *testing.T) {
	type args struct {
		resource   *construct.Resource
		sideEffect construct.ResourceId
	}
	tests := []struct {
		name         string
		initialState []any
		args         args
		want         bool
		mocks        func(mockProp *MockProperty, mockNestedProp *MockProperty, mockKB *MockTemplateKB)
		wantErr      bool
	}{
		{
			name:         "operational resource side effect, shallow resource id",
			initialState: []any{"p:t:side", "p:t:test -> p:t:side"},
			args: args{
				resource: &construct.Resource{
					ID: construct.ResourceId{
						Provider: "p",
						Type:     "t",
						Name:     "test",
					},
					Properties: construct.Properties{
						"test": construct.ResourceId{
							Provider: "p",
							Type:     "t",
							Name:     "side",
						},
					},
				},
				sideEffect: construct.ResourceId{
					Provider: "p",
					Type:     "t",
					Name:     "side",
				},
			},
			mocks: func(mockProp *MockProperty, mockNestedProp *MockProperty, mockKB *MockTemplateKB) {
				template := &ResourceTemplate{
					Properties: Properties{
						"test": mockProp,
					},
				}
				mockKB.EXPECT().GetResourceTemplate(gomock.Any()).Return(template, nil).AnyTimes()
				mockProp.EXPECT().Details().Return(&PropertyDetails{
					OperationalRule: &PropertyRule{
						Step: OperationalStep{
							Direction: DirectionDownstream,
							Resources: []ResourceSelector{
								{
									Selector: "p:t:side",
								},
							},
						},
					},
					Name: "test",
					Path: "test",
				}).AnyTimes()
				mockProp.EXPECT().Type().Return("resource").AnyTimes()
				mockProp.EXPECT().SubProperties().Return(nil).AnyTimes()
			},
			want: true,
		},
		{
			name:         "operational resource side effect, shallow property ref",
			initialState: []any{"p:t:side", "p:t:test -> p:t:side"},
			args: args{
				resource: &construct.Resource{
					ID: construct.ResourceId{
						Provider: "p",
						Type:     "t",
						Name:     "test",
					},
					Properties: construct.Properties{
						"test": construct.PropertyRef{
							Resource: construct.ResourceId{
								Provider: "p",
								Type:     "t",
								Name:     "side",
							},
							Property: "test",
						},
					},
				},
				sideEffect: construct.ResourceId{
					Provider: "p",
					Type:     "t",
					Name:     "side",
				},
			},
			mocks: func(mockProp *MockProperty, mockNestedProp *MockProperty, mockKB *MockTemplateKB) {
				template := &ResourceTemplate{
					Properties: Properties{
						"test": mockProp,
					},
				}
				mockKB.EXPECT().GetResourceTemplate(gomock.Any()).Return(template, nil).AnyTimes()
				mockProp.EXPECT().Details().Return(&PropertyDetails{
					OperationalRule: &PropertyRule{
						Step: OperationalStep{
							Direction: DirectionDownstream,
							Resources: []ResourceSelector{
								{
									Selector: "p:t:side",
								},
							},
						},
					},
					Name: "test",
					Path: "test",
				}).AnyTimes()
				mockProp.EXPECT().Type().Return("resource").AnyTimes()
				mockProp.EXPECT().SubProperties().Return(nil).AnyTimes()
			},
			want: true,
		},
		{
			name:         "operational resource side effect, nested resource id",
			initialState: []any{"p:t:side", "p:t:test -> p:t:side"},
			args: args{
				resource: &construct.Resource{
					ID: construct.ResourceId{
						Provider: "p",
						Type:     "t",
						Name:     "test",
					},
					Properties: construct.Properties{
						"test": map[string]any{
							"test": construct.ResourceId{
								Provider: "p",
								Type:     "t",
								Name:     "side",
							},
						},
					},
				},
				sideEffect: construct.ResourceId{
					Provider: "p",
					Type:     "t",
					Name:     "side",
				},
			},
			mocks: func(mockProp *MockProperty, mockNestedProp *MockProperty, mockKB *MockTemplateKB) {
				template := &ResourceTemplate{
					Properties: Properties{
						"test": mockProp,
					},
				}
				mockKB.EXPECT().GetResourceTemplate(gomock.Any()).Return(template, nil).AnyTimes()
				mockProp.EXPECT().Type().Return("map").AnyTimes()
				mockProp.EXPECT().SubProperties().Return(map[string]Property{
					"test": mockNestedProp,
				}).AnyTimes()
				mockProp.EXPECT().Details().Return(&PropertyDetails{
					Name: "test",
					Path: "test",
				}).AnyTimes()
				mockNestedProp.EXPECT().Details().Return(&PropertyDetails{
					OperationalRule: &PropertyRule{
						Step: OperationalStep{
							Direction: DirectionDownstream,
							Resources: []ResourceSelector{
								{
									Selector: "p:t:side",
								},
							},
						},
					},
					Name: "test",
					Path: "test.test",
				}).AnyTimes()
				mockNestedProp.EXPECT().Type().Return("resource").AnyTimes()
				mockNestedProp.EXPECT().SubProperties().Return(nil).AnyTimes()
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			ctrl := gomock.NewController(t)
			mockProp := NewMockProperty(ctrl)
			mockNestedProp := NewMockProperty(ctrl)
			mockKB := NewMockTemplateKB(ctrl)
			g := construct.NewGraph()
			err := g.AddVertex(tt.args.resource)
			assert.NoError(err)
			dag := graphtest.MakeGraph(t, g, tt.initialState...)
			tt.mocks(mockProp, mockNestedProp, mockKB)
			got, err := IsOperationalResourceSideEffect(dag, mockKB, tt.args.resource.ID, tt.args.sideEffect)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)
			assert.Equal(tt.want, got)
			ctrl.Finish()
		})
	}
}
