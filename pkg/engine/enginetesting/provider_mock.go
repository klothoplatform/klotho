package enginetesting

import (
	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/klothoplatform/klotho/pkg/construct"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
)

type MockProvider struct {
}

func (p *MockProvider) CreateResourceFromId(id construct.ResourceId, dag *construct.ConstructGraph) (construct.Resource, error) {
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
func (p *MockProvider) ExpandConstruct(c construct.Construct, cg *construct.ConstructGraph, dag *construct.ResourceGraph, constructType string, attributes map[string]any) (directlyMappedResources []construct.Resource, err error) {
	switch c := c.(type) {
	case *types.ExecutionUnit:
		switch constructType {
		case "mock1":
			mock1 := &MockResource1{Name: c.Name, ConstructRefs: construct.BaseConstructSetOf(c)}
			dag.AddResource(mock1)
			return []construct.Resource{mock1}, nil
		}
		return nil, nil
	case *types.Orm:
		res := &MockResource3{Name: c.Name, ConstructRefs: construct.BaseConstructSetOf(c)}
		dag.AddResource(res)
		return []construct.Resource{res}, nil
	}
	return nil, nil
}

func (p *MockProvider) ListResources() []construct.Resource {
	return []construct.Resource{
		&MockResource1{},
		&MockResource2{},
		&MockResource3{},
		&MockResource4{},
	}
}

func (p *MockProvider) GetOperationalTempaltes() map[construct.ResourceId]*construct.ResourceTemplate {
	return map[construct.ResourceId]*construct.ResourceTemplate{}
}

func (p *MockProvider) GetEdgeTempaltes() map[string]*knowledgebase.EdgeTemplate {
	return map[string]*knowledgebase.EdgeTemplate{}
}

func (p *MockProvider) Name() string {
	return "mock"
}
