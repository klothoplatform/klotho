package enginetesting

import (
	"github.com/klothoplatform/klotho/pkg/core"
)

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
	case "mock4":
		return &mockResource4{Name: id.Name}, nil
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

func (p *MockProvider) ListResources() []core.Resource {
	return []core.Resource{
		&mockResource1{},
		&mockResource2{},
		&mockResource3{},
		&mockResource4{},
	}
}
func (p *MockProvider) Name() string {
	return "mock"
}
