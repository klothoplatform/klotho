package constructs

import (
	"fmt"
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/k2/model"
)

type (
	ConstructContext struct {
		Urn                model.URN
		ConstructTemplate  ConstructTemplate
		Meta               map[string]any
		Inputs             map[string]any
		Resources          map[string]*Resource
		Edges              []*Edge
		OutputDeclarations map[string]OutputDeclaration
		AppEnvState        model.State
	}

	ResourceRef struct {
		ResourceKey string
		Property    string
		Type        ResourceRefType
		//functionsType ResourceRefType
	}

	OutputDeclaration struct {
		Name  string
		Ref   construct.PropertyRef
		Value any
	}

	ResourceRefType      string
	InterpolationSource  string
	InterpolationContext []InterpolationSource
)

func (r *ResourceRef) MarshalValue() any {
	return r.String()
}

func (r *ResourceRef) String() string {
	if r.Type == ResourceRefTypeIaC {
		return fmt.Sprintf("%s#%s", r.ResourceKey, r.Property)
	}
	return r.ResourceKey
}

const (
	// ResourceRefTypeTemplate is a reference to a resource template and will be fully resolved prior to constraint generation
	ResourceRefTypeTemplate ResourceRefType = "template"
	// ResourceRefTypeIaC is a reference to an infrastructure as code resource that will be resolved by the engine
	ResourceRefTypeIaC ResourceRefType = "iac"
	// ResourceRefTypeInterpolated is a reference to an interpolated value. An interpolated value will be evaluated during initial processing and will be converted to one of the other types.
	ResourceRefTypeInterpolated ResourceRefType = "interpolated"
)

const (
	InputsInterpolation    InterpolationSource = "inputs"
	ResourcesInterpolation InterpolationSource = "resources"
	EdgesInterpolation     InterpolationSource = "edges"
	MetaInterpolation      InterpolationSource = "meta"
	StackInterpolation     InterpolationSource = "stack"
)

var (
	ResourceInterpolationContext  = InterpolationContext{StackInterpolation, InputsInterpolation, ResourcesInterpolation, ResourcesInterpolation}
	EdgeInterpolationContext      = InterpolationContext{StackInterpolation, InputsInterpolation, ResourcesInterpolation, EdgesInterpolation}
	OutputInterpolationContext    = InterpolationContext{StackInterpolation, InputsInterpolation, ResourcesInterpolation, EdgesInterpolation, MetaInterpolation}
	InputRuleInterpolationContext = InterpolationContext{StackInterpolation, InputsInterpolation, ResourcesInterpolation, EdgesInterpolation, MetaInterpolation}
)

// NewConstructContext creates a new ConstructContext instance
func NewConstructContext(constructUrn model.URN, inputs map[string]any, appEnvState model.State) (*ConstructContext, error) {
	var templateId ConstructTemplateId
	err := templateId.FromURN(constructUrn)
	if err != nil {
		return nil, err
	}
	ct, err := loadConstructTemplate(templateId)
	if err != nil {
		return nil, err
	}
	return &ConstructContext{
		Urn:                constructUrn,
		ConstructTemplate:  ct,
		Meta:               map[string]any{},
		Inputs:             inputs,
		Resources:          map[string]*Resource{},
		Edges:              []*Edge{},
		OutputDeclarations: map[string]OutputDeclaration{},
		AppEnvState:        appEnvState,
	}, nil
}

func (c *ConstructContext) SerializeRef(ref ResourceRef) (any, error) {
	resource, ok := c.Resources[ref.ResourceKey]
	if !ok {
		return nil, fmt.Errorf("invalid ref: resource with key %s not found", ref.ResourceKey)
	}

	if ref.Property == "" {
		return resource.Id.String(), nil
	}

	if ref.Property != "" && ref.Type == ResourceRefTypeIaC {
		return fmt.Sprintf("%s#%s", resource.Id.String(), ref.Property), nil
	}

	return resource.Id.String(), nil
}
