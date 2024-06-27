package constructs

import (
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"go.uber.org/zap"
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
		BindingTemplate    BindingTemplate
		Meta               map[string]any
		Inputs             construct.Properties
		Resources          map[string]*Resource
		Edges              []*Edge
		OutputDeclarations map[string]OutputDeclaration
		Outputs            map[string]any
		ImportedResources  map[construct.ResourceId]map[string]any
	}
)

func (b *Binding) GetInput(name string) (val any, ok bool) {
	val, ok = b.Inputs[name]
	return val, ok
}

func (b *Binding) GetTemplateResourcesIterator() Iterator[string, ResourceTemplate] {
	return b.BindingTemplate.ResourcesIterator()
}

func (b *Binding) GetTemplateEdges() []EdgeTemplate {
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

func (b *Binding) GetInputRules() []InputRuleTemplate {
	return b.BindingTemplate.InputRules
}

func (b *Binding) GetTemplateOutputs() map[string]OutputTemplate {
	return b.BindingTemplate.Outputs
}

func (b *Binding) GetImportedResources() map[construct.ResourceId]map[string]any {
	return b.ImportedResources
}

func (b *Binding) DeclareOutput(key string, declaration OutputDeclaration) {
	b.OutputDeclarations[key] = declaration
}

func (b *Binding) GetTemplateInputs() map[string]InputTemplate {
	return b.BindingTemplate.Inputs
}

func (b *Binding) GetURN() model.URN {
	if b.Owner == nil {
		return model.URN{}
	}
	return b.Owner.GetURN()
}

func (b *Binding) GetPropertySource() *PropertySource {

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
	return NewPropertySource(ps)
}

// newBinding creates a new Binding instance
// owner: the construct that owns the binding
// from: the construct that is the source of the binding
// to: the construct that is the target of the binding
// inputs: the inputs to the binding (default values will be populated for any missing inputs)
// returns: the new binding instance or an error if one occurred
func (ce *ConstructEvaluator) newBinding(owner, from, to model.URN) (*Binding, error) {
	ownerTemplateId, err := ParseConstructTemplateId(owner.Subtype)
	if err != nil {
		return nil, err
	}
	fromTemplateId, err := ParseConstructTemplateId(from.Subtype)
	if err != nil {
		return nil, err
	}
	toTemplateId, err := ParseConstructTemplateId(to.Subtype)
	if err != nil {
		return nil, err
	}

	oc := ce.constructs[owner]
	fc := ce.constructs[from]
	tc := ce.constructs[to]

	bt, err := loadBindingTemplate(ownerTemplateId, fromTemplateId, toTemplateId)
	if err != nil {
		zap.S().Debugf("template not found for binding %s -> %s -> %s", ownerTemplateId, fromTemplateId, toTemplateId)
		bt = BindingTemplate{
			From:      fromTemplateId,
			To:        toTemplateId,
			Priority:  0,
			Inputs:    make(map[string]InputTemplate),
			Outputs:   make(map[string]OutputTemplate),
			Resources: make(map[string]ResourceTemplate),
		}
	}

	return &Binding{
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
		ImportedResources:  make(map[construct.ResourceId]map[string]any),
	}, nil
}
