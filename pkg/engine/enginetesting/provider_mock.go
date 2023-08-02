package enginetesting

import (
	"github.com/klothoplatform/klotho/pkg/core"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
)

type MockProvider struct {
}

func (p *MockProvider) CreateResourceFromId(id core.ResourceId, dag *core.ConstructGraph) (core.Resource, error) {
	switch id.Type {
	case "mock1":
		return &MockResource1{Name: id.Name}, nil
	case "mock2":
		return &MockResource2{Name: id.Name}, nil
	case "mock3":
		return &MockResource3{Name: id.Name}, nil
	case "mock4":
		return &MockResource4{Name: id.Name}, nil
	}
	return nil, nil
}
func (p *MockProvider) ExpandConstruct(construct core.Construct, cg *core.ConstructGraph, dag *core.ResourceGraph, constructType string, attributes map[string]any) (directlyMappedResources []core.Resource, err error) {
	switch c := construct.(type) {
	case *core.ExecutionUnit:
		switch constructType {
		case "mock1":
			mock1 := &MockResource1{Name: c.Name, ConstructRefs: core.BaseConstructSetOf(c)}
			dag.AddResource(mock1)
			return []core.Resource{mock1}, nil
		}
		return nil, nil
	case *core.Orm:
		res := &MockResource3{Name: c.Name, ConstructRefs: core.BaseConstructSetOf(c)}
		dag.AddResource(res)
		return []core.Resource{res}, nil
	}
	return nil, nil
}

func (p *MockProvider) ListResources() []core.Resource {
	return []core.Resource{
		&MockResource1{},
		&MockResource2{},
		&MockResource3{},
		&MockResource4{},
	}
}

func (p *MockProvider) GetOperationalTempaltes() map[core.ResourceId]*core.ResourceTemplate {
	return map[core.ResourceId]*core.ResourceTemplate{}
}

func (p *MockProvider) GetEdgeTempaltes() map[string]*knowledgebase.EdgeTemplate {
	return map[string]*knowledgebase.EdgeTemplate{}
}

func (p *MockProvider) Name() string {
	return "mock"
}
