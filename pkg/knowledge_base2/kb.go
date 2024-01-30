package knowledgebase2

import (
	"errors"
	"fmt"
	"sort"
	"text/template"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"go.uber.org/zap"
)

//go:generate mockgen --source=./kb.go -destination=./template_kb_mock_test.go -package=knowledgebase2
//go:generate mockgen --source=./kb.go -destination=../engine2/operational_eval/template_kb_mock_test.go -package=operational_eval

type (
	TemplateKB interface {
		ListResources() []*ResourceTemplate
		GetModel(model string) *Model
		Edges() ([]graph.Edge[*ResourceTemplate], error)
		AddResourceTemplate(template *ResourceTemplate) error
		AddEdgeTemplate(template *EdgeTemplate) error
		GetResourceTemplate(id construct.ResourceId) (*ResourceTemplate, error)
		GetEdgeTemplate(from, to construct.ResourceId) *EdgeTemplate
		HasDirectPath(from, to construct.ResourceId) bool
		HasFunctionalPath(from, to construct.ResourceId) bool
		AllPaths(from, to construct.ResourceId) ([][]*ResourceTemplate, error)
		GetAllowedNamespacedResourceIds(ctx DynamicValueContext, resourceId construct.ResourceId) ([]construct.ResourceId, error)
		GetClassification(id construct.ResourceId) Classification
		GetResourcesNamespaceResource(resource *construct.Resource) (construct.ResourceId, error)
		GetResourcePropertyType(resource construct.ResourceId, propertyName string) string
		GetPathSatisfactionsFromEdge(source, target construct.ResourceId) ([]EdgePathSatisfaction, error)
	}

	// KnowledgeBase is a struct that represents the object which contains the knowledge of how to make resources operational
	KnowledgeBase struct {
		underlying graph.Graph[string, *ResourceTemplate]
		Models     map[string]*Model
	}

	EdgePathSatisfaction struct {
		// Signals if the classification is derived from the target or not
		// we need this to know how to construct the edge we are going to run expansion on if we have resource values in the classification
		Classification string
		Source         PathSatisfactionRoute
		Target         PathSatisfactionRoute
	}

	ValueOrTemplate struct {
		Value    any
		Template *template.Template
	}
)

const (
	glueEdgeWeight               = 0
	defaultEdgeWeight            = 1
	functionalBoundaryEdgeWeight = 10000
)

func NewKB() *KnowledgeBase {
	return &KnowledgeBase{
		underlying: graph.New[string, *ResourceTemplate](func(t *ResourceTemplate) string {
			return t.Id().QualifiedTypeName()
		}, graph.Directed()),
	}
}

func (kb *KnowledgeBase) GetModel(model string) *Model {
	return kb.Models[model]
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

func (kb *KnowledgeBase) Edges() ([]graph.Edge[*ResourceTemplate], error) {
	edges, err := kb.underlying.Edges()
	if err != nil {
		return nil, err
	}
	var result []graph.Edge[*ResourceTemplate]
	for _, edge := range edges {
		src, err := kb.underlying.Vertex(edge.Source)
		if err != nil {
			return nil, err
		}
		dst, err := kb.underlying.Vertex(edge.Target)
		if err != nil {
			return nil, err
		}
		result = append(result, graph.Edge[*ResourceTemplate]{
			Source: src,
			Target: dst,
		})
	}
	return result, nil

}

func (kb *KnowledgeBase) AddResourceTemplate(template *ResourceTemplate) error {
	return kb.underlying.AddVertex(template)
}

func (kb *KnowledgeBase) AddEdgeTemplate(template *EdgeTemplate) error {
	sourceTmpl, err := kb.underlying.Vertex(template.Source.QualifiedTypeName())
	if err != nil {
		return fmt.Errorf("could not find source template: %w", err)
	}
	targetTmpl, err := kb.underlying.Vertex(template.Target.QualifiedTypeName())
	if err != nil {
		return fmt.Errorf("could not find target template: %w", err)
	}
	weight := defaultEdgeWeight
	if sourceTmpl.GetFunctionality() == Unknown {
		if targetTmpl.GetFunctionality() == Unknown {
			weight = glueEdgeWeight
		} else {
			weight = functionalBoundaryEdgeWeight
		}
	}
	return kb.underlying.AddEdge(
		template.Source.QualifiedTypeName(),
		template.Target.QualifiedTypeName(),
		graph.EdgeData(template),
		graph.EdgeWeight(weight),
	)
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
	fromType := from.QualifiedTypeName()
	toType := to.QualifiedTypeName()
	if fromType == toType {
		// For resources that can reference themselves, such as aws:api_resource
		return true
	}
	path, err := graph.ShortestPathStable(
		kb.underlying,
		from.QualifiedTypeName(),
		to.QualifiedTypeName(),
		func(a, b string) bool { return a < b },
	)
	if errors.Is(err, graph.ErrTargetNotReachable) {
		return false
	}
	if err != nil {
		zap.S().Errorf(
			"error in finding shortes path from %s to %s: %v",
			from.QualifiedTypeName(), to.QualifiedTypeName(), err,
		)
		return false
	}
	for _, id := range path[1 : len(path)-1] {
		template, err := kb.underlying.Vertex(id)
		if err != nil {
			panic(err)
		}
		if template.GetFunctionality() != Unknown {
			return false
		}
	}
	return true
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

func (kb *KnowledgeBase) GetAllowedNamespacedResourceIds(ctx DynamicValueContext, resourceId construct.ResourceId) ([]construct.ResourceId, error) {

	template, err := kb.GetResourceTemplate(resourceId)
	if err != nil {
		return nil, fmt.Errorf("could not find resource template for %s: %w", resourceId, err)
	}
	var result []construct.ResourceId
	property := template.GetNamespacedProperty()
	if property == nil {
		return result, nil
	}
	rule := property.Details().OperationalRule
	if rule == nil {
		return result, nil
	}
	if rule.Step.Resources != nil {
		for _, resource := range rule.Step.Resources {
			if resource.Selector != "" {
				id, err := ExecuteDecodeAsResourceId(ctx, resource.Selector, DynamicValueData{Resource: resourceId})
				if err != nil {
					return nil, err
				}
				template, err := kb.GetResourceTemplate(id)
				if err != nil {
					return nil, err
				}
				if template.ResourceContainsClassifications(resource.Classifications) {
					result = append(result, id)
				}
			}
			if resource.Classifications != nil && resource.Selector == "" {
				for _, resTempalte := range kb.ListResources() {
					if resTempalte.ResourceContainsClassifications(resource.Classifications) {
						result = append(result, resTempalte.Id())
					}
				}

			}
		}
	}
	return result, nil
}

func GetFunctionality(kb TemplateKB, id construct.ResourceId) Functionality {
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

func (kb *KnowledgeBase) GetResourcesNamespaceResource(resource *construct.Resource) (construct.ResourceId, error) {
	template, err := kb.GetResourceTemplate(resource.ID)
	if err != nil {
		return construct.ResourceId{}, err
	}
	namespaceProperty := template.GetNamespacedProperty()
	if namespaceProperty != nil {
		ns, err := resource.GetProperty(namespaceProperty.Details().Name)
		if err != nil {
			return construct.ResourceId{}, err
		}
		if ns == nil {
			return construct.ResourceId{}, nil
		}
		if _, ok := ns.(construct.ResourceId); !ok {
			return construct.ResourceId{}, fmt.Errorf("namespace property does not contain a ResourceId, got %s", ns)
		}
		return ns.(construct.ResourceId), nil
	}
	return construct.ResourceId{}, nil
}

func (kb *KnowledgeBase) GetResourcePropertyType(resource construct.ResourceId, propertyName string) string {
	template, err := kb.GetResourceTemplate(resource)
	if err != nil {
		return ""
	}
	for _, property := range template.Properties {
		if property.Details().Name == propertyName {
			return property.Type()
		}
	}
	return ""
}

// TransformToPropertyValue transforms a value to the correct type for a given property
// This is used for transforming values from the config template (and any interface value we want to set on a resource) to the correct type for the resource
func TransformToPropertyValue(
	resource construct.ResourceId,
	propertyName string,
	value interface{},
	ctx DynamicContext,
	data DynamicValueData,
) (interface{}, error) {
	template, err := ctx.KB().GetResourceTemplate(resource)
	if err != nil {
		return nil, err
	}
	property := template.GetProperty(propertyName)
	if property == nil {
		return nil, fmt.Errorf(
			"could not find property %s on resource %s",
			propertyName, resource,
		)
	}
	if value == nil {
		return property.ZeroValue(), nil
	}
	val, err := property.Parse(value, ctx, data)
	if err != nil {
		return nil, fmt.Errorf(
			"could not parse value %v for property %s on resource %s: %w",
			value, property.Details().Name, resource, err,
		)
	}
	return val, nil
}

func TransformAllPropertyValues(ctx DynamicValueContext) error {
	ids, err := construct.TopologicalSort(ctx.DAG())
	if err != nil {
		return err
	}
	resources, err := construct.ResolveIds(ctx.DAG(), ids)
	if err != nil {
		return err
	}

	var errs error

resourceLoop:
	for _, resource := range resources {
		tmpl, err := ctx.KB().GetResourceTemplate(resource.ID)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		data := DynamicValueData{Resource: resource.ID}

		for name := range tmpl.Properties {
			path, err := resource.PropertyPath(name)
			if err != nil {
				errs = errors.Join(errs, err)
				continue
			}
			preXform := path.Get()
			if preXform == nil {
				continue
			}
			val, err := TransformToPropertyValue(resource.ID, name, preXform, ctx, data)
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf("error transforming %s#%s: %w", resource.ID, name, err))
				continue resourceLoop
			}
			err = path.Set(val)
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf("errors setting %s#%s: %w", resource.ID, name, err))
				continue resourceLoop
			}
		}
	}
	return errs
}
