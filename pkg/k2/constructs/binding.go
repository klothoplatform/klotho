package constructs

import (
	"errors"
	"fmt"
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/k2/constructs/template"
	"github.com/klothoplatform/klotho/pkg/k2/constructs/template/property"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"go.uber.org/zap"
	"io/fs"
)

type (
	BindingDeclaration struct {
		From   model.URN
		To     model.URN
		Inputs map[string]model.Input
	}

	Binding struct {
		Owner              *Construct
		From               *Construct
		To                 *Construct
		Priority           int
		BindingTemplate    template.BindingTemplate
		Meta               map[string]any
		Inputs             construct.Properties
		Resources          map[string]*Resource
		Edges              []*Edge
		OutputDeclarations map[string]OutputDeclaration
		Outputs            map[string]any
		InitialGraph       construct.Graph
	}
)

func (b *Binding) GetInputs() construct.Properties {
	return b.Inputs
}

func (b *Binding) GetInputValue(name string) (value any, err error) {
	return b.Inputs.GetProperty(name)
}

func (b *Binding) GetTemplateResourcesIterator() template.Iterator[string, template.ResourceTemplate] {
	return b.BindingTemplate.ResourcesIterator()
}

func (b *Binding) GetTemplateEdges() []template.EdgeTemplate {
	return b.BindingTemplate.Edges
}

func (b *Binding) GetEdges() []*Edge {
	return b.Edges
}

func (b *Binding) SetEdges(edges []*Edge) {
	b.Edges = edges
}

func (b *Binding) GetResource(resourceId string) (resource *Resource, ok bool) {
	resource, ok = b.Resources[resourceId]
	return
}

func (b *Binding) SetResource(resourceId string, resource *Resource) {
	b.Resources[resourceId] = resource
}

func (b *Binding) GetResources() map[string]*Resource {
	return b.Resources
}

func (b *Binding) GetInputRules() []template.InputRuleTemplate {
	return b.BindingTemplate.InputRules
}

func (b *Binding) GetTemplateOutputs() map[string]template.OutputTemplate {
	return b.BindingTemplate.Outputs
}

func (b *Binding) GetInitialGraph() construct.Graph {
	return b.InitialGraph
}

func (b *Binding) DeclareOutput(key string, declaration OutputDeclaration) {
	b.OutputDeclarations[key] = declaration
}

func (b *Binding) GetURN() model.URN {
	if b.Owner == nil {
		return model.URN{}
	}
	return b.Owner.GetURN()
}

func (b *Binding) String() string {
	e := Edge{
		From: template.ResourceRef{ConstructURN: b.From.URN},
		To:   template.ResourceRef{ConstructURN: b.To.URN},
	}
	return e.String()
}

func (b *Binding) GetPropertySource() *template.PropertySource {
	ps := map[string]any{
		"inputs":    b.Inputs,
		"resources": b.Resources,
		"edges":     b.Edges,
		"meta":      b.Meta,
	}
	if b.From != nil {
		ps["from"] = map[string]any{
			"urn":       b.From.URN,
			"inputs":    b.From.Inputs,
			"resources": b.From.Resources,
			"edges":     b.From.Edges,
			"meta":      b.From.Meta,
		}
	}
	if b.To != nil {
		ps["to"] = map[string]any{
			"urn":       b.To.URN,
			"inputs":    b.To.Inputs,
			"resources": b.To.Resources,
			"edges":     b.To.Edges,
			"meta":      b.To.Meta,
			"outputs":   b.To.Outputs,
		}
	}
	return template.NewPropertySource(ps)
}

func (b *Binding) GetConstruct() *Construct {
	return b.Owner
}

// newBinding initializes a new binding instance using the template associated with the owner construct
// returns: the new binding instance or an error if one occurred
func (ce *ConstructEvaluator) newBinding(owner model.URN, d BindingDeclaration) (*Binding, error) {
	ownerTemplateId, err := property.ParseConstructType(owner.Subtype)
	if err != nil {
		return nil, err
	}
	fromTemplateId, err := property.ParseConstructType(d.From.Subtype)
	if err != nil {
		return nil, err
	}
	toTemplateId, err := property.ParseConstructType(d.To.Subtype)
	if err != nil {
		return nil, err
	}

	oc, _ := ce.Constructs.Get(owner)
	fc, _ := ce.Constructs.Get(d.From)
	tc, _ := ce.Constructs.Get(d.To)

	bt, err := template.LoadBindingTemplate(ownerTemplateId, fromTemplateId, toTemplateId)
	var pathError *fs.PathError
	if errors.As(err, &pathError) {
		zap.S().Debugf("template not found for binding %s -> %s -> %s", ownerTemplateId, fromTemplateId, toTemplateId)
		bt = template.BindingTemplate{
			From:      fromTemplateId,
			To:        toTemplateId,
			Priority:  0,
			Inputs:    template.NewProperties(nil),
			Outputs:   make(map[string]template.OutputTemplate),
			Resources: make(map[string]template.ResourceTemplate),
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to load binding template %s -> %s -> %s: %w", ownerTemplateId.String(), fromTemplateId.String(), toTemplateId.String(), err)
	}

	b := &Binding{
		Owner:              oc,
		From:               fc,
		To:                 tc,
		BindingTemplate:    bt,
		Priority:           bt.Priority,
		Meta:               make(map[string]any),
		Inputs:             make(map[string]any),
		Resources:          make(map[string]*Resource),
		Edges:              []*Edge{},
		OutputDeclarations: make(map[string]OutputDeclaration),
		Outputs:            make(map[string]any),
		InitialGraph:       construct.NewGraph(),
	}

	inputs, err := ce.convertInputs(d.Inputs)
	if err != nil {
		return nil, fmt.Errorf("invalid inputs for binding %s -> %s: %w", d.From, d.To, err)
	}
	err = ce.initializeInputs(b, inputs)
	if err != nil {
		return nil, fmt.Errorf("input initialization failed for binding %s -> %s: %w", d.From, d.To, err)
	}
	return b, nil
}

func (b *Binding) ForEachInput(f func(property.Property) error) error {
	return b.BindingTemplate.ForEachInput(b.Inputs, f)
}
