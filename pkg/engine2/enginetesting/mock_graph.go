package enginetesting

import (
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/stretchr/testify/mock"
)

type MockGraph struct {
	mock.Mock
}

func (g *MockGraph) ListResources() ([]*construct.Resource, error) {
	args := g.Called()
	return args.Get(0).([]*construct.Resource), args.Error(1)
}
func (g *MockGraph) AddResource(resource *construct.Resource) {
}
func (g *MockGraph) RemoveResource(resource *construct.Resource, explicit bool) error {
	args := g.Called(resource, explicit)
	return args.Error(0)
}
func (g *MockGraph) AddDependency(from *construct.Resource, to *construct.Resource) error {
	args := g.Called(from, to)
	return args.Error(0)
}
func (g *MockGraph) RemoveDependency(from construct.ResourceId, to construct.ResourceId) error {
	args := g.Called(from, to)
	return args.Error(0)
}
func (g *MockGraph) GetResource(id construct.ResourceId) (*construct.Resource, error) {
	args := g.Called(id)
	return args.Get(0).(*construct.Resource), args.Error(1)
}

func (g *MockGraph) DownstreamOfType(resource *construct.Resource, layer int, qualifiedType string) ([]*construct.Resource, error) {
	args := g.Called(resource, layer, qualifiedType)
	return args.Get(0).([]*construct.Resource), args.Error(1)
}
func (g *MockGraph) Downstream(resource *construct.Resource, layer int) ([]*construct.Resource, error) {
	args := g.Called(resource, layer)
	return args.Get(0).([]*construct.Resource), args.Error(1)
}

func (g *MockGraph) UpstreamOfType(resource *construct.Resource, layer int, qualifiedType string) ([]*construct.Resource, error) {
	args := g.Called(resource, layer, qualifiedType)
	return args.Get(0).([]*construct.Resource), args.Error(1)
}

func (g *MockGraph) Upstream(resource *construct.Resource, layer int) ([]*construct.Resource, error) {
	args := g.Called(resource, layer)
	return args.Get(0).([]*construct.Resource), args.Error(1)
}

func (g *MockGraph) ReplaceResourceId(oldId construct.ResourceId, resource construct.ResourceId) error {
	args := g.Called(oldId, resource)
	return args.Error(0)
}
func (g *MockGraph) ConfigureResource(resource *construct.Resource, configuration knowledgebase.Configuration, data knowledgebase.DynamicValueData, action string) error {
	args := g.Called(resource, configuration, data, action)
	return args.Error(0)
}

func (g *MockGraph) ShortestPath(from, to construct.ResourceId) ([]*construct.Resource, error) {
	args := g.Called(from, to)
	return args.Get(0).([]*construct.Resource), args.Error(1)
}

func (g *MockGraph) KnowledgeBase() knowledgebase.TemplateKB {
	args := g.Called()
	return args.Get(0).(knowledgebase.TemplateKB)
}

func (g *MockGraph) GetDataflowGraph() construct.Graph {
	args := g.Called()
	return args.Get(0).(construct.Graph)
}
