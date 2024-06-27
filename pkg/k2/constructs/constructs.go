package constructs

import (
	"fmt"
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/klothoplatform/klotho/pkg/k2/reflectutil"
	"reflect"
	"text/template"
)

type (
	ResourceOwner interface {
		GetImportedResources() map[construct.ResourceId]map[string]any
		GetResource(resourceId string) (resource *Resource, ok bool)
		SetResource(resourceId string, resource *Resource)
		GetResources() map[string]*Resource
		GetTemplateResourcesIterator() Iterator[string, ResourceTemplate]
		InterpolationSource
	}

	EdgeOwner interface {
		GetTemplateEdges() []EdgeTemplate
		GetEdges() []*Edge
		SetEdges(edges []*Edge)
		InterpolationSource
	}

	InfraOwner interface {
		GetURN() model.URN
		GetInputRules() []InputRuleTemplate
		ResourceOwner
		EdgeOwner
		GetTemplateOutputs() map[string]OutputTemplate
		DeclareOutput(key string, declaration OutputDeclaration)
		GetTemplateInputs() map[string]InputTemplate
		GetInput(name string) (value any, ok bool)
	}

	InterpolationSource interface {
		GetPropertySource() *PropertySource
	}

	PropertySource struct {
		source reflect.Value
	}

	TemplateFuncSupplier interface {
		GetTemplateFuncs() template.FuncMap
	}
)

func NewPropertySource(source any) *PropertySource {
	return &PropertySource{
		source: reflect.ValueOf(source),
	}
}

func (p *PropertySource) GetProperty(key string) (value any, ok bool) {
	v, err := reflectutil.GetField(p.source, key)
	if err != nil || !v.IsValid() {
		return nil, false
	}
	return v.Interface(), true
}

func (ce *ConstructEvaluator) serializeRef(ref ResourceRef) (any, error) {
	owner := ce.constructs[ref.ConstructURN]
	if owner == nil {
		return nil, fmt.Errorf("construct with key %s not found", ref.ConstructURN.String())
	}

	var resourceId construct.ResourceId
	r, ok := owner.GetResource(ref.ResourceKey)
	if ok {
		resourceId = r.Id
	} else {
		err := resourceId.Parse(ref.ResourceKey)
		if err != nil {
			return nil, err
		}
		importedResources := owner.GetImportedResources()
		if _, ok := importedResources[resourceId]; !ok {
			return nil, fmt.Errorf("resource with key %s not found", ref.ResourceKey)
		}
	}

	if ref.Property != "" {
		return fmt.Sprintf("%s#%s", resourceId.String(), ref.Property), nil
	}

	return resourceId.String(), nil
}

func GetTypedProperty[T any](source *PropertySource, key string) (T, bool) {
	var typedField T
	v, ok := source.GetProperty(key)

	if !ok {
		return typedField, false
	}

	return reflectutil.GetTypedValue[T](v)
}
