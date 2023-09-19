package engine

import (
	"fmt"
	"testing"

	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/enginetesting"
	"github.com/klothoplatform/klotho/pkg/graph"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider"
	"github.com/stretchr/testify/assert"
)

func Test_handleOperationalRule(t *testing.T) {
	tests := []struct {
		name                 string
		rule                 knowledgebase.OperationalRule
		resource             *enginetesting.MockResource5
		parent               construct.Resource
		existingDependencies []graph.Edge[construct.Resource]
		want                 []Decision
		wantErr              []error
	}{
		{
			name: "only one none exists",
			rule: knowledgebase.OperationalRule{
				Enforcement:   knowledgebase.ExactlyOne,
				Direction:     knowledgebase.Downstream,
				ResourceTypes: []string{"mock1"},
				SetField:      "Mock1",
				UnsatisfiedAction: knowledgebase.UnsatisfiedAction{
					Operation: knowledgebase.CreateUnsatisfiedResource,
				},
			},
			resource: &enginetesting.MockResource5{Name: "this"},
			wantErr: []error{&OperationalResourceError{
				Count:    1,
				Resource: &enginetesting.MockResource5{Name: "this"},
				Cause:    fmt.Errorf("rule with enforcement exactly one has less than the required number of resources of type [mock1]  or classifications [], 0 for resource mock:mock5:this"),
			}},
		},
		{
			name: "only one one exists",
			rule: knowledgebase.OperationalRule{
				Enforcement:   knowledgebase.ExactlyOne,
				Direction:     knowledgebase.Downstream,
				ResourceTypes: []string{"mock1"},
				SetField:      "Mock1",
			},
			resource: &enginetesting.MockResource5{Name: "this"},
			existingDependencies: []graph.Edge[construct.Resource]{
				{Source: &enginetesting.MockResource5{Name: "this"}, Destination: &enginetesting.MockResource1{Name: "that"}},
			},
			want: []Decision{
				{
					Action: ActionConnect,
					Result: &DecisionResult{
						Edge: &graph.Edge[construct.Resource]{Source: &enginetesting.MockResource5{Name: "this"}, Destination: &enginetesting.MockResource1{Name: "that"}},
					},
				},
			},
		},
		{
			name: "only one multiple exist error",
			rule: knowledgebase.OperationalRule{
				Enforcement:   knowledgebase.ExactlyOne,
				Direction:     knowledgebase.Downstream,
				ResourceTypes: []string{"mock1"},
				SetField:      "Mock1",
			},
			resource: &enginetesting.MockResource5{Name: "this"},
			existingDependencies: []graph.Edge[construct.Resource]{
				{Source: &enginetesting.MockResource5{Name: "this"}, Destination: &enginetesting.MockResource1{Name: "that"}},
				{Source: &enginetesting.MockResource5{Name: "this"}, Destination: &enginetesting.MockResource1{Name: "that2"}},
			},
			wantErr: []error{&ResourceNotOperationalError{
				Resource: &enginetesting.MockResource5{Name: "this"},
				Cause:    fmt.Errorf("rule with enforcement exactly one has more than one resource for rule exactly_one [mock1] for resource mock:mock5:this ([mock:mock1:that mock:mock1:that2])"),
			},
			},
		},
		{
			name: "if one none exists",
			rule: knowledgebase.OperationalRule{
				Enforcement:   knowledgebase.Conditional,
				Direction:     knowledgebase.Downstream,
				ResourceTypes: []string{"mock1"},
				SetField:      "Mock1",
			},
			resource: &enginetesting.MockResource5{Name: "this"},
		},
		{
			name: "if one one exists",
			rule: knowledgebase.OperationalRule{
				Enforcement:   knowledgebase.Conditional,
				Direction:     knowledgebase.Downstream,
				ResourceTypes: []string{"mock1"},
				SetField:      "Mock1",
			},
			resource: &enginetesting.MockResource5{Name: "this"},
			existingDependencies: []graph.Edge[construct.Resource]{
				{Source: &enginetesting.MockResource5{Name: "this"}, Destination: &enginetesting.MockResource1{Name: "that"}},
			},
		},
		{
			name: "if one one exists with sub rules",
			rule: knowledgebase.OperationalRule{
				Enforcement:            knowledgebase.Conditional,
				Direction:              knowledgebase.Downstream,
				ResourceTypes:          []string{"mock3"},
				RemoveDirectDependency: true,
				Rules: []knowledgebase.OperationalRule{
					{
						Enforcement:   knowledgebase.AnyAvailable,
						Direction:     knowledgebase.Downstream,
						ResourceTypes: []string{"mock2"},
						SetField:      "Mock2s",
						NumNeeded:     2,
						UnsatisfiedAction: knowledgebase.UnsatisfiedAction{
							Operation: knowledgebase.CreateUnsatisfiedResource,
						},
					},
				},
			},
			resource: &enginetesting.MockResource5{Name: "this"},
			existingDependencies: []graph.Edge[construct.Resource]{
				{Source: &enginetesting.MockResource5{Name: "this"}, Destination: &enginetesting.MockResource3{Name: "that"}},
			},
			wantErr: []error{&OperationalResourceError{
				Rule: knowledgebase.OperationalRule{ // the subrule
					Enforcement:   knowledgebase.AnyAvailable,
					Direction:     knowledgebase.Downstream,
					ResourceTypes: []string{"mock2"},
					SetField:      "Mock2s",
					NumNeeded:     2,
					UnsatisfiedAction: knowledgebase.UnsatisfiedAction{
						Operation: knowledgebase.CreateUnsatisfiedResource,
					},
				},
				Resource: &enginetesting.MockResource5{Name: "this"},
				Count:    2,
				Parent:   &enginetesting.MockResource3{Name: "that"},
				Cause:    fmt.Errorf("rule with enforcement any has less than the required number of resources of type [mock2]  or classifications [], 0 for resource mock:mock5:this"),
			}},
		},
		{
			name: "if one multiple exist error",
			rule: knowledgebase.OperationalRule{
				Enforcement:   knowledgebase.Conditional,
				Direction:     knowledgebase.Downstream,
				ResourceTypes: []string{"mock1"},
				SetField:      "Mock1",
			},
			resource: &enginetesting.MockResource5{Name: "this"},
			existingDependencies: []graph.Edge[construct.Resource]{
				{Source: &enginetesting.MockResource5{Name: "this"}, Destination: &enginetesting.MockResource1{Name: "that"}},
				{Source: &enginetesting.MockResource5{Name: "this"}, Destination: &enginetesting.MockResource1{Name: "that2"}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			mp := &enginetesting.MockProvider{}
			engine := NewEngine(map[string]provider.Provider{
				mp.Name(): mp,
			}, enginetesting.MockKB, types.ListAllConstructs())
			engine.ClassificationDocument = enginetesting.BaseClassificationDocument

			dag := construct.NewResourceGraph()
			for _, dep := range tt.existingDependencies {
				dag.AddDependency(dep.Source, dep.Destination)
			}

			decisions, errs := engine.handleOperationalRule(tt.resource, tt.rule, dag, tt.parent)
			if tt.wantErr != nil {
				for _, e := range tt.wantErr {
					if opErr, ok := e.(*OperationalResourceError); ok {
						if opErr.Rule.Direction == "" {
							opErr.Rule = tt.rule
						}
						if opErr.Resource == nil {
							opErr.Resource = tt.resource
						}
					}
				}
				assert.ElementsMatch(errs, tt.wantErr)
				return
			}
			if !assert.Len(errs, 0) {
				return
			}
			CompareDecisions(t, tt.want, decisions)
		})
	}
}

func Test_handleOperationalResourceError(t *testing.T) {
	tests := []struct {
		name                 string
		ore                  *OperationalResourceError
		existingDependencies []graph.Edge[construct.Resource]
		want                 []Decision
		wantErr              bool
	}{
		{
			name: "needs one downstream",
			ore: &OperationalResourceError{
				Resource: &enginetesting.MockResource5{Name: "this"},
				Rule:     knowledgebase.OperationalRule{ResourceTypes: []string{"mock1"}, Direction: knowledgebase.Downstream},
				Count:    1,
				Cause:    fmt.Errorf("0"),
			},
			want: []Decision{
				{
					Action: ActionConnect,
					Result: &DecisionResult{
						Edge: &graph.Edge[construct.Resource]{Source: &enginetesting.MockResource5{Name: "this"}, Destination: &enginetesting.MockResource1{Name: "mock1-0"}},
					},
				},
			},
		},
		{
			name: "needs multiple downstream",
			ore: &OperationalResourceError{
				Resource: &enginetesting.MockResource5{Name: "this"},
				Rule:     knowledgebase.OperationalRule{ResourceTypes: []string{"mock2"}, Direction: knowledgebase.Downstream},
				Count:    2,
				Cause:    fmt.Errorf("0"),
			},
			want: []Decision{
				{
					Action: ActionConnect,
					Result: &DecisionResult{
						Edge: &graph.Edge[construct.Resource]{Source: &enginetesting.MockResource5{Name: "this"}, Destination: &enginetesting.MockResource2{Name: "mock2-0"}},
					},
				},
				{
					Action: ActionConnect,
					Result: &DecisionResult{
						Edge: &graph.Edge[construct.Resource]{Source: &enginetesting.MockResource5{Name: "this"}, Destination: &enginetesting.MockResource2{Name: "mock2-1"}},
					},
				},
			},
		},
		{
			name: "needs parents resource",
			ore: &OperationalResourceError{
				Resource: &enginetesting.MockResource5{Name: "this"},
				Rule:     knowledgebase.OperationalRule{ResourceTypes: []string{"mock1"}, Direction: knowledgebase.Downstream},
				Count:    1,
				Parent:   &enginetesting.MockResource3{Name: "parent"},
				Cause:    fmt.Errorf("0"),
			},
			existingDependencies: []graph.Edge[construct.Resource]{
				{Source: &enginetesting.MockResource1{Name: "child"}, Destination: &enginetesting.MockResource3{Name: "parent"}},
			},
			want: []Decision{
				{
					Action: ActionConnect,
					Result: &DecisionResult{
						Edge: &graph.Edge[construct.Resource]{Source: &enginetesting.MockResource5{Name: "this"}, Destination: &enginetesting.MockResource1{Name: "child"}},
					},
				},
			},
		},
		{
			name: "needs 2 but parent only has 1 resource",
			ore: &OperationalResourceError{
				Resource: &enginetesting.MockResource5{Name: "this"},
				Rule:     knowledgebase.OperationalRule{ResourceTypes: []string{"mock1"}, Direction: knowledgebase.Downstream},
				Count:    2,
				Parent:   &enginetesting.MockResource3{Name: "parent"},
				Cause:    fmt.Errorf("0"),
			},
			existingDependencies: []graph.Edge[construct.Resource]{
				{Source: &enginetesting.MockResource1{Name: "child"}, Destination: &enginetesting.MockResource3{Name: "parent"}},
				{Source: &enginetesting.MockResource1{Name: "child2"}, Destination: &enginetesting.MockResource3{Name: "parent2"}},
			},
			want: []Decision{
				{
					Action: ActionConnect,
					Result: &DecisionResult{
						Edge: &graph.Edge[construct.Resource]{Source: &enginetesting.MockResource5{Name: "this"}, Destination: &enginetesting.MockResource1{Name: "child"}},
					},
				},
				{
					Action: ActionConnect,
					Result: &DecisionResult{
						Edge: &graph.Edge[construct.Resource]{Source: &enginetesting.MockResource5{Name: "this"}, Destination: &enginetesting.MockResource1{Name: "mock1-2"}},
					},
				},
				{
					Action: ActionConnect,
					Result: &DecisionResult{
						Edge: &graph.Edge[construct.Resource]{Source: &enginetesting.MockResource1{Name: "mock1-2"}, Destination: &enginetesting.MockResource3{Name: "parent"}},
					},
				},
			},
		},
		{
			name: "chooses existing resource to satisfy needs",
			ore: &OperationalResourceError{
				Resource: &enginetesting.MockResource5{Name: "this"},
				Rule:     knowledgebase.OperationalRule{ResourceTypes: []string{"mock1"}, Direction: knowledgebase.Downstream},
				Count:    2,
				Cause:    fmt.Errorf("0"),
			},
			existingDependencies: []graph.Edge[construct.Resource]{
				{Source: &enginetesting.MockResource1{Name: "child"}, Destination: &enginetesting.MockResource3{Name: "parent"}},
				{Source: &enginetesting.MockResource1{Name: "child2"}, Destination: &enginetesting.MockResource3{Name: "parent2"}},
			},
			want: []Decision{
				{
					Action: ActionConnect,
					Result: &DecisionResult{
						Edge: &graph.Edge[construct.Resource]{Source: &enginetesting.MockResource5{Name: "this"}, Destination: &enginetesting.MockResource1{Name: "child"}},
					},
				},
				{
					Action: ActionConnect,
					Result: &DecisionResult{
						Edge: &graph.Edge[construct.Resource]{Source: &enginetesting.MockResource5{Name: "this"}, Destination: &enginetesting.MockResource1{Name: "child2"}},
					},
				},
			},
		},
		{
			name: "must create new resource to satisfy needs",
			ore: &OperationalResourceError{
				Resource: &enginetesting.MockResource5{Name: "this"},
				Rule: knowledgebase.OperationalRule{
					ResourceTypes:     []string{"mock1"},
					UnsatisfiedAction: knowledgebase.UnsatisfiedAction{Unique: true},
					Direction:         knowledgebase.Downstream,
				},
				Count: 2,
				Cause: fmt.Errorf("0"),
			},
			existingDependencies: []graph.Edge[construct.Resource]{
				{Source: &enginetesting.MockResource1{Name: "child"}, Destination: &enginetesting.MockResource3{Name: "parent"}},
				{Source: &enginetesting.MockResource1{Name: "child2"}, Destination: &enginetesting.MockResource3{Name: "parent2"}},
			},
			want: []Decision{
				{
					Action: ActionConnect,
					Result: &DecisionResult{
						Edge: &graph.Edge[construct.Resource]{Source: &enginetesting.MockResource5{Name: "this"}, Destination: &enginetesting.MockResource1{Name: "mock1-this-2"}},
					},
				},
				{
					Action: ActionConnect,
					Result: &DecisionResult{
						Edge: &graph.Edge[construct.Resource]{Source: &enginetesting.MockResource5{Name: "this"}, Destination: &enginetesting.MockResource1{Name: "mock1-this-3"}},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			mp := &enginetesting.MockProvider{}
			engine := NewEngine(map[string]provider.Provider{
				mp.Name(): mp,
			}, enginetesting.MockKB, types.ListAllConstructs())
			engine.ClassificationDocument = enginetesting.BaseClassificationDocument

			dag := construct.NewResourceGraph()
			for _, dep := range tt.existingDependencies {
				dag.AddDependency(dep.Source, dep.Destination)
			}

			decisions, err := engine.handleOperationalResourceError(tt.ore, dag)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			CompareDecisions(t, tt.want, decisions)
		})
	}
}

func CompareDecisions(t *testing.T, expected, actual []Decision) {
	if !assert.Equal(t, len(expected), len(actual), "expected %d decisions, got %d: %v", len(expected), len(actual), actual) {
		return
	}
	for i, expected := range expected {
		switch expected.Action {
		case ActionCreate:
			assert.Equal(t, expected.Action, actual[i].Action, "expected decision %d to be equal", i)
			assert.Equal(t, expected.Result.Resource.Id(), actual[i].Result.Resource.Id(), "expected decision %d to be equal for result resource id", i)
		case ActionConnect:
			assert.Equal(t, expected.Action, actual[i].Action, "expected decision %d to be equal", i)
			assert.Equal(t, expected.Result.Edge.Source.Id(), actual[i].Result.Edge.Source.Id(), "expected decision %d to be equal for result edge source id", i)
			assert.Equal(t, expected.Result.Edge.Destination.Id(), actual[i].Result.Edge.Destination.Id(), "expected decision %d to be equal for result edge destination id", i)
			assert.Equal(t, expected.Result.Edge.Properties, actual[i].Result.Edge.Properties, "expected decision %d to be equal for result edge label", i)
		}
	}
}
