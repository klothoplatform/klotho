package engine

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/stretchr/testify/assert"
)

func Test_Engine_Run(t *testing.T) {
	tests := []struct {
		name        string
		constructs  []core.Construct
		edges       []constraints.Edge
		constraints map[constraints.ConstraintScope][]constraints.Constraint
		want        coretesting.ResourcesExpectation
	}{
		{
			name: "sample exec unit -> orm case",
			constructs: []core.Construct{
				&core.ExecutionUnit{Name: "compute"},
				&core.Orm{Name: "orm"},
			},
			edges: []constraints.Edge{
				{
					Source: core.ResourceId{Provider: core.AbstractConstructProvider, Type: core.EXECUTION_UNIT_TYPE, Name: "compute"},
					Target: core.ResourceId{Provider: core.AbstractConstructProvider, Type: core.ORM_TYPE, Name: "orm"},
				},
			},
			constraints: map[constraints.ConstraintScope][]constraints.Constraint{
				constraints.ConstructConstraintScope: {
					&constraints.ConstructConstraint{
						Operator: constraints.EqualsConstraintOperator,
						Target:   core.ResourceId{Provider: core.AbstractConstructProvider, Type: core.ORM_TYPE, Name: "orm"},
						Type:     "mock3",
					},
					&constraints.ConstructConstraint{
						Operator: constraints.EqualsConstraintOperator,
						Target:   core.ResourceId{Provider: core.AbstractConstructProvider, Type: core.EXECUTION_UNIT_TYPE, Name: "compute"},
						Type:     "mock1",
					},
				},
				constraints.EdgeConstraintScope: {
					&constraints.EdgeConstraint{
						Operator: constraints.MustContainConstraintOperator,
						Target: constraints.Edge{
							Source: core.ResourceId{Provider: core.AbstractConstructProvider, Type: core.EXECUTION_UNIT_TYPE, Name: "compute"},
							Target: core.ResourceId{Provider: core.AbstractConstructProvider, Type: core.ORM_TYPE, Name: "orm"},
						},
						Node: core.ResourceId{Provider: "mock", Type: "mock2", Name: "Corm"},
					},
				},
			},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"mock:mock1:compute",
					"mock:mock2:Corm",
					"mock:mock3:orm",
				},
				Deps: []coretesting.StringDep{
					{Source: "mock:mock1:compute", Destination: "mock:mock2:Corm"},
					{Source: "mock:mock2:Corm", Destination: "mock:mock3:orm"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			engine := NewEngine(&MockProvider{}, MockKB, core.ListAllConstructs())

			cg := core.NewConstructGraph()
			for _, c := range tt.constructs {
				cg.AddConstruct(c)
			}
			for _, e := range tt.edges {
				cg.AddDependency(e.Source, e.Target)
			}

			engine.LoadContext(cg, tt.constraints, "test")
			dag, err := engine.Run()
			if !assert.NoError(err) {
				return
			}
			tt.want.Assert(t, dag)
		})
	}
}

type MockProvider struct {
}

func (p *MockProvider) CreateResourceFromId(id core.ResourceId, dag *core.ConstructGraph) (core.Resource, error) {
	switch id.Type {
	case "mock1":
		return &mockResource1{Name: id.Name}, nil
	case "mock2":
		return &mockResource2{Name: id.Name}, nil
	case "mock3":
		return &mockResource3{Name: id.Name}, nil
	}
	return nil, nil
}
func (p *MockProvider) ExpandConstruct(construct core.Construct, cg *core.ConstructGraph, dag *core.ResourceGraph, constructType string, attributes map[string]any) (directlyMappedResources []core.Resource, err error) {
	switch c := construct.(type) {
	case *core.ExecutionUnit:
		switch constructType {
		case "mock1":
			mock1 := &mockResource1{Name: c.Name, ConstructsRef: core.BaseConstructSetOf(c)}
			dag.AddResource(mock1)
			return []core.Resource{mock1}, nil
		}
		return nil, nil
	case *core.Orm:
		res := &mockResource3{Name: c.Name, ConstructsRef: core.BaseConstructSetOf(c)}
		dag.AddResource(res)
		return []core.Resource{res}, nil
	}
	return nil, nil
}

func (p *MockProvider) LoadResources(graph core.InputGraph, resources map[core.ResourceId]core.BaseConstruct) error {
	return nil
}
func (p *MockProvider) Name() string {
	return "mock"
}

type (
	mockResource1 struct {
		Name          string
		ConstructsRef core.BaseConstructSet `yaml:"-"`
	}
	mockResource2 struct {
		Name          string
		ConstructsRef core.BaseConstructSet `yaml:"-"`
	}
	mockResource3 struct {
		Name          string
		ConstructsRef core.BaseConstructSet `yaml:"-"`
	}
)

func (f *mockResource1) Id() core.ResourceId {
	return core.ResourceId{Provider: "mock", Type: "mock1", Name: f.Name}
}
func (f *mockResource1) BaseConstructsRef() core.BaseConstructSet { return f.ConstructsRef }
func (f *mockResource1) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream:   true,
		RequiresNoDownstream: true,
	}
}
func (f *mockResource2) Id() core.ResourceId {
	return core.ResourceId{Provider: "mock", Type: "mock2", Name: f.Name}
}
func (f *mockResource2) BaseConstructsRef() core.BaseConstructSet { return f.ConstructsRef }
func (f *mockResource2) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream:   true,
		RequiresNoDownstream: true,
	}
}
func (f *mockResource3) Id() core.ResourceId {
	return core.ResourceId{Provider: "mock", Type: "mock3", Name: f.Name}
}
func (f *mockResource3) BaseConstructsRef() core.BaseConstructSet { return f.ConstructsRef }
func (f *mockResource3) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream:   true,
		RequiresNoDownstream: true,
	}
}

var MockKB = knowledgebase.Build(
	knowledgebase.EdgeBuilder[*mockResource1, *mockResource2]{
		Expand: func(source *mockResource1, target *mockResource2, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			dag.AddDependency(source, target)
			return nil
		},
		ValidDestinations: []core.Resource{&mockResource3{}},
	},
	knowledgebase.EdgeBuilder[*mockResource1, *mockResource3]{},
	knowledgebase.EdgeBuilder[*mockResource2, *mockResource3]{
		Expand: func(source *mockResource2, target *mockResource3, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			dag.AddDependency(source, target)
			return nil
		},
	},
)
