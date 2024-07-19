package constructs

import (
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/k2/constructs/template"
	"github.com/klothoplatform/klotho/pkg/k2/constructs/template/property"
	"github.com/klothoplatform/klotho/pkg/k2/model"
)

type (
	ResourceOwner interface {
		GetResource(resourceId string) (resource *Resource, ok bool)
		SetResource(resourceId string, resource *Resource)
		GetResources() map[string]*Resource
		GetTemplateResourcesIterator() template.Iterator[string, template.ResourceTemplate]
		template.InterpolationSource
	}

	EdgeOwner interface {
		GetTemplateEdges() []template.EdgeTemplate
		GetEdges() []*Edge
		SetEdges(edges []*Edge)
		template.InterpolationSource
	}

	InfraOwner interface {
		GetURN() model.URN
		GetInputRules() []template.InputRuleTemplate
		ResourceOwner
		EdgeOwner
		GetTemplateOutputs() map[string]template.OutputTemplate
		DeclareOutput(key string, declaration OutputDeclaration)
		ForEachInput(f func(input property.Property) error) error
		GetInputValue(name string) (value any, err error)
		GetInitialGraph() construct.Graph
		GetConstruct() *Construct
	}
)

// marshalRef marshals a resource reference into a [construct.ResourceId] or [construct.PropertyRef]
func (ce *ConstructEvaluator) marshalRef(owner InfraOwner, ref template.ResourceRef) (any, error) {
	var resourceId construct.ResourceId
	r, ok := owner.GetResource(ref.ResourceKey)
	if ok {
		resourceId = r.Id
	} else {
		err := resourceId.Parse(ref.ResourceKey)
		if err != nil {
			return nil, err
		}
	}

	if ref.Property != "" {
		return construct.PropertyRef{
			Resource: resourceId,
			Property: ref.Property,
		}, nil
	}

	return resourceId, nil
}
