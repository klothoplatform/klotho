package constructs

import (
	"errors"
	"fmt"
	"github.com/klothoplatform/klotho/pkg/k2/constructs/template"
	inputs2 "github.com/klothoplatform/klotho/pkg/k2/constructs/template/property"
	"sort"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/solution"
	"github.com/klothoplatform/klotho/pkg/k2/model"
)

type (
	Construct struct {
		URN                model.URN
		ConstructTemplate  template.ConstructTemplate
		Meta               map[string]any
		Inputs             construct.Properties
		Resources          map[string]*Resource
		Edges              []*Edge
		OutputDeclarations map[string]OutputDeclaration
		Outputs            map[string]any
		InitialGraph       construct.Graph
		Bindings           []*Binding
		Solution           solution.Solution
	}

	Resource struct {
		Id         construct.ResourceId
		Properties construct.Properties
	}

	Edge struct {
		From template.ResourceRef
		To   template.ResourceRef
		Data construct.EdgeData
	}

	OutputDeclaration struct {
		Name  string
		Ref   construct.PropertyRef
		Value any
	}
)

func (c *Construct) GetInputValue(name string) (value any, err error) {
	return c.Inputs.GetProperty(name)
}

func (c *Construct) GetTemplateResourcesIterator() template.Iterator[string, template.ResourceTemplate] {
	return c.ConstructTemplate.ResourcesIterator()
}

func (c *Construct) GetTemplateEdges() []template.EdgeTemplate {
	return c.ConstructTemplate.Edges
}

func (c *Construct) GetEdges() []*Edge {
	return c.Edges
}

func (c *Construct) SetEdges(edges []*Edge) {
	c.Edges = edges
}

func (c *Construct) GetInputRules() []template.InputRuleTemplate {
	return c.ConstructTemplate.InputRules
}

func (c *Construct) GetTemplateOutputs() map[string]template.OutputTemplate {
	return c.ConstructTemplate.Outputs
}

func (c *Construct) GetPropertySource() *template.PropertySource {
	return template.NewPropertySource(map[string]any{
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

func (c *Construct) GetInitialGraph() construct.Graph {
	return c.InitialGraph
}

func (c *Construct) DeclareOutput(key string, declaration OutputDeclaration) {
	c.OutputDeclarations[key] = declaration
}

func (c *Construct) GetURN() model.URN {
	return c.URN
}

func (c *Construct) GetInputs() construct.Properties {
	return c.Inputs
}

func (e *Edge) PrettyPrint() string {
	return e.From.String() + " -> " + e.To.String()
}

func (e *Edge) String() string {
	return e.PrettyPrint() + " :: " + fmt.Sprintf("%v", e.Data)
}

// OrderedBindings returns the bindings sorted by priority (lowest to highest).
// If two bindings have the same priority, their declaration order is preserved.
func (c *Construct) OrderedBindings() []*Binding {
	if len(c.Bindings) == 0 {
		return nil
	}

	sorted := append([]*Binding{}, c.Bindings...)

	sort.SliceStable(sorted, func(i, j int) bool {
		if c.Bindings[i].Priority == c.Bindings[j].Priority {
			return i < j
		}
		return c.Bindings[i].Priority < c.Bindings[j].Priority
	})
	return sorted
}

func (c *Construct) GetConstruct() *Construct {
	return c
}

func (c *Construct) ForEachInput(f func(input inputs2.Property) error) error {
	return c.ConstructTemplate.ForEachInput(c.Inputs, f)
}

// newConstruct creates a new Construct instance from the given URN and inputs.
// The URN must be a construct URN.
// Any inputs that are not provided will be populated with default values from the construct template.
func (ce *ConstructEvaluator) newConstruct(constructUrn model.URN, i construct.Properties) (*Construct, error) {
	if _, ok := i["Name"]; ok {
		return nil, errors.New("'Name' is a reserved input key")
	}
	if !constructUrn.IsResource() || constructUrn.Type != "construct" {
		return nil, errors.New("invalid construct URN")
	}

	/// Load the construct template
	var templateId inputs2.ConstructType
	err := templateId.FromURN(constructUrn)
	if err != nil {
		return nil, err
	}
	ct, err := template.LoadConstructTemplate(templateId)
	if err != nil {
		return nil, err
	}

	c := &Construct{
		URN:                constructUrn,
		ConstructTemplate:  ct,
		Meta:               make(map[string]any),
		Inputs:             make(construct.Properties),
		Resources:          make(map[string]*Resource),
		Edges:              []*Edge{},
		OutputDeclarations: make(map[string]OutputDeclaration),
		Outputs:            make(map[string]any),
		InitialGraph:       construct.NewGraph(),
	}

	// Add the construct name to the inputs
	err = c.Inputs.SetProperty("Name", constructUrn.ResourceID)
	if err != nil {
		return nil, err
	}

	err = ce.initializeInputs(c, i)
	if err != nil {
		return nil, err
	}
	return c, nil
}
