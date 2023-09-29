package knowledgebase2

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"go.uber.org/zap"
)

type (
	TemplateKB interface {
		ListResources() []*ResourceTemplate
		AddResourceTemplate(template *ResourceTemplate) error
		AddEdgeTemplate(template *EdgeTemplate) error
		GetResourceTemplate(id construct.ResourceId) (*ResourceTemplate, error)
		GetEdgeTemplate(from, to construct.ResourceId) *EdgeTemplate
		HasDirectPath(from, to construct.ResourceId) bool
		HasFunctionalPath(from, to construct.ResourceId) bool
		AllPaths(from, to construct.ResourceId) ([][]*ResourceTemplate, error)
		GetAllowedNamespacedResourceIds(ctx ConfigTemplateContext, resourceId construct.ResourceId) ([]construct.ResourceId, error)
		GetFunctionality(id construct.ResourceId) Functionality
		GetClassification(id construct.ResourceId) Classification
		GetResourcesNamespaceResource(resource *construct.Resource) *construct.Resource
		GetResourcePropertyType(resource *construct.Resource, propertyName string) string
		TransformToPropertyValue(resource *construct.Resource, propertyName string, value interface{}, ctx ConfigTemplateContext, data ConfigTemplateData) (interface{}, error)
	}

	// KnowledgeBase is a struct that represents the object which contains the knowledge of how to make resources operational
	KnowledgeBase struct {
		underlying graph.Graph[string, *ResourceTemplate]
	}
)

func NewKB() *KnowledgeBase {
	return &KnowledgeBase{
		underlying: graph.New[string, *ResourceTemplate](func(t *ResourceTemplate) string {
			return t.Id().QualifiedTypeName()
		}, graph.Directed()),
	}
}

// ListResources returns a list of all resources in the knowledge base
// The returned list of resource templates will be sorted by the templates fully qualified type name
func (kb *KnowledgeBase) ListResources() []*ResourceTemplate {
	predecessors, err := kb.underlying.PredecessorMap()
	if err != nil {
		panic(err)
	}
	var result []*ResourceTemplate
	var ids []string
	for vId := range predecessors {
		ids = append(ids, vId)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if v, err := kb.underlying.Vertex(id); err == nil {
			result = append(result, v)
		} else {
			panic(err)
		}
	}
	return result
}

func (kb *KnowledgeBase) AddResourceTemplate(template *ResourceTemplate) error {
	return kb.underlying.AddVertex(template)
}

func (kb *KnowledgeBase) AddEdgeTemplate(template *EdgeTemplate) error {
	return kb.underlying.AddEdge(template.Source.QualifiedTypeName(), template.Target.QualifiedTypeName(), graph.EdgeData(template))
}

func (kb *KnowledgeBase) GetResourceTemplate(id construct.ResourceId) (*ResourceTemplate, error) {
	return kb.underlying.Vertex(id.QualifiedTypeName())
}

func (kb *KnowledgeBase) GetEdgeTemplate(from, to construct.ResourceId) *EdgeTemplate {
	edge, err := kb.underlying.Edge(from.QualifiedTypeName(), to.QualifiedTypeName())
	// Even if the edge does not exist, we still return nil so that we know there is no edge template since there is no edge
	if err != nil {
		return nil
	}
	data := edge.Properties.Data
	if data == nil {
		return nil
	}
	if template, ok := data.(*EdgeTemplate); ok {
		return template
	}
	return nil
}

func (kb *KnowledgeBase) HasDirectPath(from, to construct.ResourceId) bool {
	_, err := kb.underlying.Edge(from.QualifiedTypeName(), to.QualifiedTypeName())
	return err == nil
}

func (kb *KnowledgeBase) HasFunctionalPath(from, to construct.ResourceId) bool {
	paths, err := graph.AllPathsBetween(kb.underlying, from.QualifiedTypeName(), to.QualifiedTypeName())
	if err != nil {
		zap.S().Errorf("error in finding paths from %s to %s: %v", from.QualifiedTypeName(), to.QualifiedTypeName(), err)
	}
PATHS:
	for _, path := range paths {
		for i, id := range path {
			if i == len(path)-1 || i == 0 {
				continue
			}
			template, err := kb.underlying.Vertex(id)
			if err != nil {
				panic(err)
			}
			if template.GetFunctionality() != Unknown {
				continue PATHS
			}
		}
		return true
	}
	return false
}

func (kb *KnowledgeBase) AllPaths(from, to construct.ResourceId) ([][]*ResourceTemplate, error) {
	paths, err := graph.AllPathsBetween(kb.underlying, from.QualifiedTypeName(), to.QualifiedTypeName())
	if err != nil {
		return nil, err
	}
	resources := make([][]*ResourceTemplate, len(paths))
	for i, path := range paths {
		resources[i] = make([]*ResourceTemplate, len(path))
		for j, id := range path {
			resources[i][j], _ = kb.underlying.Vertex(id)
		}
	}
	return resources, nil
}

func (kb *KnowledgeBase) GetAllowedNamespacedResourceIds(ctx ConfigTemplateContext, resourceId construct.ResourceId) ([]construct.ResourceId, error) {

	template, _ := kb.GetResourceTemplate(resourceId)
	var result []construct.ResourceId
	property := template.GetNamespacedProperty()
	if property == nil {
		return result, nil
	}
	rule := property.OperationalRule
	if rule == nil {
		return result, nil
	}
	for _, step := range rule.Steps {
		if step.Resources != nil {
			for _, resource := range step.Resources {
				id, err := ctx.ExecuteDecodeAsResourceId(resource, ConfigTemplateData{Resource: resourceId})
				if err != nil {
					return nil, err
				}
				result = append(result, id)
			}
		}
		if step.Classifications != nil {
			for _, resTempalte := range kb.ListResources() {
				if resTempalte.ResourceContainsClassifications(step.Classifications) {
					result = append(result, resTempalte.Id())
				}
			}

		}
	}
	return result, nil
}

func (kb *KnowledgeBase) GetFunctionality(id construct.ResourceId) Functionality {
	template, _ := kb.GetResourceTemplate(id)
	if template == nil {
		return Unknown
	}
	return template.GetFunctionality()
}

func (kb *KnowledgeBase) GetClassification(id construct.ResourceId) Classification {
	template, _ := kb.GetResourceTemplate(id)
	if template == nil {
		return Classification{}
	}
	return template.Classification
}

func (kb *KnowledgeBase) GetResourcesNamespaceResource(resource *construct.Resource) *construct.Resource {
	template, err := kb.GetResourceTemplate(resource.ID)
	if err != nil {
		return nil
	}
	namespaceProperty := template.GetNamespacedProperty()
	if namespaceProperty != nil {
		return reflect.ValueOf(resource).Elem().FieldByName(namespaceProperty.Name).Interface().(*construct.Resource)
	}
	return nil
}

func (kb *KnowledgeBase) GetResourcePropertyType(resource *construct.Resource, propertyName string) string {
	template, err := kb.GetResourceTemplate(resource.ID)
	if err != nil {
		return ""
	}
	for _, property := range template.Properties {
		if property.Name == propertyName {
			return property.Type
		}
	}
	return ""
}

// TransformToPropertyValue transforms a value to the correct type for a given property
// This is used for transforming values from the config template (and any interface value we want to set on a resource) to the correct type for the resource
func (kb *KnowledgeBase) TransformToPropertyValue(resource *construct.Resource, propertyName string, value interface{}, ctx ConfigTemplateContext, data ConfigTemplateData) (interface{}, error) {
	template, err := kb.GetResourceTemplate(resource.ID)
	if err != nil {
		return nil, err
	}
	property := template.GetProperty(propertyName)
	if property == nil {
		return nil, fmt.Errorf("could not find property %s on resource %s", propertyName, resource.ID)
	}
	propertyType, err := property.getPropertyType()
	if err != nil {
		return nil, fmt.Errorf("could not find property type %s on resource %s for property %s", property.Type, resource.ID, property.Name)
	}
	val, err := propertyType.Parse(value, ctx, data)
	if err != nil {
		return nil, fmt.Errorf("could not parse value %v for property %s on resource %s: %w", value, property.Name, resource.ID, err)
	}
	return val, nil
}
