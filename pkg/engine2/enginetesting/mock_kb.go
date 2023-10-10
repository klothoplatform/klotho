package enginetesting

import (
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/stretchr/testify/mock"
)

type MockKB struct {
	mock.Mock
}

func (m *MockKB) ListResources() []*knowledgebase.ResourceTemplate {
	args := m.Called()
	return args.Get(0).([]*knowledgebase.ResourceTemplate)
}
func (m *MockKB) AddResourceTemplate(template *knowledgebase.ResourceTemplate) error {
	args := m.Called(template)
	return args.Error(0)
}
func (m *MockKB) AddEdgeTemplate(template *knowledgebase.EdgeTemplate) error {
	args := m.Called(template)
	return args.Error(0)
}
func (m *MockKB) GetResourceTemplate(id construct.ResourceId) (*knowledgebase.ResourceTemplate, error) {
	args := m.Called(id)
	return args.Get(0).(*knowledgebase.ResourceTemplate), args.Error(1)
}
func (m *MockKB) GetEdgeTemplate(from, to construct.ResourceId) *knowledgebase.EdgeTemplate {
	args := m.Called(from, to)
	return args.Get(0).(*knowledgebase.EdgeTemplate)
}
func (m *MockKB) HasDirectPath(from, to construct.ResourceId) bool {
	args := m.Called(from, to)
	return args.Bool(0)
}
func (m *MockKB) HasFunctionalPath(from, to construct.ResourceId) bool {
	args := m.Called(from, to)
	return args.Bool(0)
}
func (m *MockKB) AllPaths(from, to construct.ResourceId) ([][]*knowledgebase.ResourceTemplate, error) {
	args := m.Called(from, to)
	return args.Get(0).([][]*knowledgebase.ResourceTemplate), args.Error(1)
}
func (m *MockKB) GetAllowedNamespacedResourceIds(
	ctx knowledgebase.DynamicValueContext,
	resourceId construct.ResourceId,
) ([]construct.ResourceId, error) {
	args := m.Called(ctx, resourceId)
	return args.Get(0).([]construct.ResourceId), args.Error(1)
}
func (m *MockKB) GetFunctionality(id construct.ResourceId) knowledgebase.Functionality {
	args := m.Called(id)
	return args.Get(0).(knowledgebase.Functionality)
}
func (m *MockKB) GetClassification(id construct.ResourceId) knowledgebase.Classification {
	args := m.Called(id)
	return args.Get(0).(knowledgebase.Classification)
}
func (m *MockKB) GetResourcesNamespaceResource(resource *construct.Resource) construct.ResourceId {
	args := m.Called(resource)
	return args.Get(0).(construct.ResourceId)
}
func (m *MockKB) GetResourcePropertyType(resource construct.ResourceId, propertyName string) string {
	args := m.Called(resource, propertyName)
	return args.String(0)
}
