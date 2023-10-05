package operational_rule

import (
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/enginetesting"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/stretchr/testify/mock"
)

func Test_SpreadPlacer(t *testing.T) {
	tests := []struct {
		name               string
		resource           *construct.Resource
		availableResources []*construct.Resource
		step               knowledgebase.OperationalStep
		numNeeded          int
		graphMocks         []mock.Call
	}{
		{
			name: "does nothing if no available resources",
			resource: construct.CreateResource(construct.ResourceId{
				Type: "test",
				Name: "test1",
			}),
			numNeeded: 1,
		},
		{
			name: "does nothing if one available resource",
			resource: construct.CreateResource(construct.ResourceId{
				Type: "test",
				Name: "test1",
			}),
			availableResources: []*construct.Resource{
				construct.CreateResource(construct.ResourceId{
					Type: "test",
					Name: "test2",
				}),
			},
			numNeeded: 1,
		},
		{ 
			name: "no resources placed yet, places in first resource",
			resource: construct.CreateResource(construct.ResourceId{
				Type: "test",
				Name: "test1",
			}),
			availableResources: []*construct.Resource{
				construct.CreateResource(construct.ResourceId{
					Type: "parent",
					Name: "test2",
				}),
				construct.CreateResource(construct.ResourceId{
					Type: "parent",
					Name: "test3",
				}),
			},
			step: knowledgebase.OperationalStep{
				Direction: knowledgebase.Downstream,
			},
			graphMocks: []mock.Call{
				{
					Method: "AddDependency",
					Arguments: mock.Arguments{
					},
				},
			},

		
		}
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &SpreadPlacer{}
			mockKB := enginetesting.MockKB{}
			mockGraph := enginetesting.MockGraph{}
			p.SetCtx(OperationalRuleContext{
				Graph: &mockGraph,
				KB:    &mockKB,
			})
			err := p.PlaceResources(tt.resource, tt.step, tt.availableResources, &tt.numNeeded)
			if err != nil {
				t.Errorf("PlaceResources() error = %v", err)
				return
			}
			mockGraph.AssertExpectations(t)
			mockKB.AssertExpectations(t)
		})
	}

}
