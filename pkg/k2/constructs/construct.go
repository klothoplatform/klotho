package constructs

import (
	"errors"
	"fmt"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/k2/model"
)

type (
	Construct struct {
		URN                model.URN
		ConstructTemplate  ConstructTemplate
		Meta               map[string]any
		Inputs             map[string]any
		Resources          map[string]*Resource
		Edges              []*Edge
		OutputDeclarations map[string]OutputDeclaration
		Outputs            map[string]any
		ImportedResources  map[construct.ResourceId]map[string]any
	}

	Resource struct {
		Id         construct.ResourceId
		Properties map[string]any
	}

	Edge struct {
		From ResourceRef
		To   ResourceRef
		Data map[string]any
	}

	// StackInput is a type we may store in the Inputs map of a Construct. It represents a reference a stack input
	// such as pulumi.Config or pulumi.ConfigSecret.
	StackInput struct {
		Value     string
		PulumiKey string
	}

	ResourceRef struct {
		ResourceKey string
		Property    string
		Type        ResourceRefType
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

func (e *Edge) PrettyPrint() string {
	return e.From.String() + " -> " + e.To.String()
}

func (e *Edge) String() string {
	return e.PrettyPrint() + " :: " + fmt.Sprintf("%v", e.Data)
}

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
	// InputsInterpolation is an interpolation source used to interpolate values from the construct's inputs
	InputsInterpolation InterpolationSource = "inputs"
	// ResourcesInterpolation is an interpolation source used to interpolate values from the construct's resources
	ResourcesInterpolation InterpolationSource = "resources"
	// EdgesInterpolation is an interpolation source used to interpolate values from the construct's edges
	EdgesInterpolation InterpolationSource = "edges"
	// MetaInterpolation is an interpolation source used to interpolate values from the construct's metadata
	// (i.e., non-properties fields)
	MetaInterpolation InterpolationSource = "meta"
)

var (
	ResourceInterpolationContext  = InterpolationContext{InputsInterpolation, ResourcesInterpolation, ResourcesInterpolation}
	EdgeInterpolationContext      = InterpolationContext{InputsInterpolation, ResourcesInterpolation, EdgesInterpolation}
	OutputInterpolationContext    = InterpolationContext{InputsInterpolation, ResourcesInterpolation, EdgesInterpolation, MetaInterpolation}
	InputRuleInterpolationContext = InterpolationContext{InputsInterpolation, ResourcesInterpolation, EdgesInterpolation, MetaInterpolation}
)

// NewConstruct creates a new Construct instance
func NewConstruct(constructUrn model.URN, inputs map[string]any) (*Construct, error) {
	if _, ok := inputs["Name"]; ok {
		return nil, errors.New("'Name' is a reserved input key")
	}
	if !constructUrn.IsResource() || constructUrn.Type != "construct" {
		return nil, errors.New("invalid construct URN")
	}

	// Add the construct name to the inputs
	inputs["Name"] = constructUrn.ResourceID

	var templateId ConstructTemplateId
	err := templateId.FromURN(constructUrn)
	if err != nil {
		return nil, err
	}
	ct, err := loadConstructTemplate(templateId)
	if err != nil {
		return nil, err
	}
	return &Construct{
		URN:                constructUrn,
		ConstructTemplate:  ct,
		Meta:               make(map[string]any),
		Inputs:             inputs,
		Resources:          make(map[string]*Resource),
		Edges:              []*Edge{},
		OutputDeclarations: make(map[string]OutputDeclaration),
		Outputs:            make(map[string]any),
		ImportedResources:  make(map[construct.ResourceId]map[string]any),
	}, nil
}

func (c *Construct) SerializeRef(ref ResourceRef) (any, error) {
	var resourceId construct.ResourceId
	r, ok := c.Resources[ref.ResourceKey]
	if ok {
		resourceId = r.Id
	} else {
		err := resourceId.Parse(ref.ResourceKey)
		if err != nil {
			return nil, err
		}
		if _, ok := c.ImportedResources[resourceId]; !ok {
			return nil, fmt.Errorf("resource with key %s not found", ref.ResourceKey)
		}
	}

	if ref.Property != "" {
		return fmt.Sprintf("%s#%s", resourceId.String(), ref.Property), nil
	}

	return resourceId.String(), nil
}
