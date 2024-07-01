package constructs

import (
	"errors"
	"fmt"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"go.uber.org/zap"
	"sort"
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
		Bindings           []*Binding
	}

	Resource struct {
		Id         construct.ResourceId
		Properties construct.Properties
	}

	Edge struct {
		From ResourceRef
		To   ResourceRef
		Data construct.EdgeData
	}

	ResourceRef struct {
		ConstructURN model.URN
		ResourceKey  string
		Property     string
		Type         ResourceRefType
	}

	OutputDeclaration struct {
		Name  string
		Ref   construct.PropertyRef
		Value any
	}

	ResourceRefType        string
	InterpolationSourceKey string
	InterpolationContext   struct {
		AllowedKeys []InterpolationSourceKey
		Construct   *Construct
	}
)

func NewInterpolationContext(c *Construct, keys []InterpolationSourceKey) InterpolationContext {
	if c == nil {
		c = &Construct{}
	}
	return InterpolationContext{
		AllowedKeys: keys,
		Construct:   c,
	}
}

func (c *Construct) GetInput(name string) (value any, ok bool) {
	value, ok = c.Inputs[name]
	return value, ok
}

func (c *Construct) GetTemplateResourcesIterator() Iterator[string, ResourceTemplate] {
	return c.ConstructTemplate.ResourcesIterator()

}

func (c *Construct) GetTemplateEdges() []EdgeTemplate {
	return c.ConstructTemplate.Edges
}

func (c *Construct) GetEdges() []*Edge {
	return c.Edges
}

func (c *Construct) SetEdges(edges []*Edge) {
	c.Edges = edges
}

func (c *Construct) GetInputRules() []InputRuleTemplate {
	return c.ConstructTemplate.InputRules
}

func (c *Construct) GetTemplateOutputs() map[string]OutputTemplate {
	return c.ConstructTemplate.Outputs
}

func (c *Construct) GetPropertySource() *PropertySource {
	return NewPropertySource(map[string]any{
		"inputs":    c.Inputs,
		"resources": c.Resources,
		"edges":     c.Edges,
		"meta":      c.Meta,
	})
}

func (c *Construct) GetResource(resourceId string) (resource *Resource, ok bool) {
	resource, ok = c.Resources[resourceId]
	return
}

func (c *Construct) SetResource(resourceId string, resource *Resource) {
	c.Resources[resourceId] = resource
}

func (c *Construct) GetResources() map[string]*Resource {
	return c.Resources
}

func (c *Construct) GetImportedResources() map[construct.ResourceId]map[string]any {
	return c.ImportedResources
}

func (c *Construct) DeclareOutput(key string, declaration OutputDeclaration) {
	c.OutputDeclarations[key] = declaration
}

func (c *Construct) GetTemplateInputs() map[string]InputTemplate {
	return c.ConstructTemplate.Inputs
}

func (c *Construct) GetURN() model.URN {
	return c.URN
}

func (e *Edge) PrettyPrint() string {
	return e.From.String() + " -> " + e.To.String()
}

func (e *Edge) String() string {
	return e.PrettyPrint() + " :: " + fmt.Sprintf("%v", e.Data)
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
	// ResourceRefTypeInterpolated is a reference to an interpolated value.
	// An interpolated value will be evaluated during initial processing and will be converted to one of the other types.
	ResourceRefTypeInterpolated ResourceRefType = "interpolated"
)

const (
	// InputsInterpolation is an interpolation source used to interpolate values from the construct's inputs
	InputsInterpolation InterpolationSourceKey = "inputs"
	// ResourcesInterpolation is an interpolation source used to interpolate values from the construct's resources
	ResourcesInterpolation InterpolationSourceKey = "resources"
	// EdgesInterpolation is an interpolation source used to interpolate values from the construct's edges
	EdgesInterpolation InterpolationSourceKey = "edges"
	// MetaInterpolation is an interpolation source used to interpolate values from the construct's metadata
	// (i.e., non-properties fields)
	MetaInterpolation InterpolationSourceKey = "meta"
	// BindingInterpolation is an interpolation source used to interpolate values
	// from a binding's from/to constructs using the "from" and "to" interpolation prefixes respectively.
	FromInterpolation InterpolationSourceKey = "from"
	ToInterpolation   InterpolationSourceKey = "to"
)

var (
	ResourceInterpolationContext  = []InterpolationSourceKey{InputsInterpolation, ResourcesInterpolation, ResourcesInterpolation}
	EdgeInterpolationContext      = []InterpolationSourceKey{InputsInterpolation, ResourcesInterpolation, EdgesInterpolation}
	OutputInterpolationContext    = []InterpolationSourceKey{InputsInterpolation, ResourcesInterpolation, EdgesInterpolation, MetaInterpolation}
	InputRuleInterpolationContext = []InterpolationSourceKey{InputsInterpolation, ResourcesInterpolation, EdgesInterpolation, MetaInterpolation}
	BindingInterpolationContext   = []InterpolationSourceKey{InputsInterpolation, ResourcesInterpolation, EdgesInterpolation, MetaInterpolation, FromInterpolation, ToInterpolation}
)

// NewConstruct creates a new Construct instance from the given URN and inputs.
// The URN must be a construct URN.
// Any inputs that are not provided will be populated with default values from the construct template.
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

	populateDefaultInputValues(inputs, ct.Inputs)

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

func populateDefaultInputValues(inputs map[string]any, templates map[string]InputTemplate) {
	for key, t := range templates {
		if _, hasVal := inputs[key]; !hasVal && t.Default != nil {
			defaultValue := t.Default
			if t.Type == "path" {
				pStr, ok := defaultValue.(string)
				if !ok {
					continue
				}
				var err error
				defaultValue, err = handlePathInput(pStr)
				if err != nil {
					zap.S().Warnf("failed to handle path input %s=%v: %v", key, pStr, err)
					continue
				}
			}
			inputs[key] = defaultValue
		}
		zap.S().Debugf("populated default value for input %s=%v", key, t)
	}
}

// OrderedBindings returns the bindings sorted by priority (lowest to highest).
// If two bindings have the same priority, their declaration order is preserved.
func (c *Construct) OrderedBindings() []*Binding {
	if len(c.Bindings) == 0 {
		return nil
	}

	sorted := append([]*Binding{}, c.Bindings...)

	sort.Slice(sorted, func(i, j int) bool {
		if c.Bindings[i].Priority == c.Bindings[j].Priority {
			return i < j
		}
		return c.Bindings[i].Priority < c.Bindings[j].Priority
	})
	return sorted
}
