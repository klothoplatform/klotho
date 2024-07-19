package constructs

import (
	"context"
	"errors"
	"fmt"

	"github.com/klothoplatform/klotho/pkg/async"
	"github.com/klothoplatform/klotho/pkg/k2/constructs/template"
	"github.com/klothoplatform/klotho/pkg/k2/constructs/template/property"

	"reflect"
	"slices"
	"strings"

	"github.com/dominikbraun/graph"
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine"
	stateconverter "github.com/klothoplatform/klotho/pkg/infra/state_reader/state_converter"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/klothoplatform/klotho/pkg/k2/stack"
	"github.com/klothoplatform/klotho/pkg/logging"
)

type ConstructEvaluator struct {
	DryRun model.DryRun

	stateManager      *model.StateManager
	stackStateManager *stack.StateManager
	stateConverter    stateconverter.StateConverter

	Constructs *async.ConcurrentMap[model.URN, *Construct]
}

func NewConstructEvaluator(sm *model.StateManager, ssm *stack.StateManager) (*ConstructEvaluator, error) {
	stateConverter, err := loadStateConverter()
	if err != nil {
		return nil, err
	}

	return &ConstructEvaluator{
		stateManager:      sm,
		stackStateManager: ssm,
		stateConverter:    stateConverter,
		Constructs:        &async.ConcurrentMap[model.URN, *Construct]{},
	}, nil
}

func (ce *ConstructEvaluator) Evaluate(constructUrn model.URN, state model.State, ctx context.Context) (engine.SolveRequest, error) {
	ci, err := ce.evaluateConstruct(constructUrn, state, ctx)
	if err != nil {
		return engine.SolveRequest{}, fmt.Errorf("error evaluating construct %s: %w", constructUrn, err)
	}
	err = ce.evaluateBindings(ctx, ci)
	if err != nil {
		return engine.SolveRequest{}, fmt.Errorf("error evaluating bindings: %w", err)
	}

	marshaller := ConstructMarshaller{ConstructEvaluator: ce}
	constraintList, err := marshaller.Marshal(constructUrn)
	if err != nil {
		return engine.SolveRequest{}, fmt.Errorf("error marshalling construct to constraints: %w", err)
	}

	cs, err := constraintList.ToConstraints()
	if err != nil {
		return engine.SolveRequest{}, fmt.Errorf("error converting constraint list to constraints: %w", err)
	}

	return engine.SolveRequest{
		Constraints:  cs,
		InitialState: ci.InitialGraph,
	}, nil
}

/*
evaluateInputRules evaluates the input rules of the construct

An input rule is a conditional expression that determines a set of resources, edges, and outputs based on the inputs of the construct
An input rule is evaluated by checking the if condition and then evaluating the then or else condition based on the result
the if condition is a go template that can access the inputs of the construct
input rules cannot use interpolation in the if condition

	Example:
	  - if: {{ eq inputs("foo") "bar" }}
		then:
		resources:
		  "my-resource":
		properties:
		  foo: "bar"

in the example input() is a function that returns the value of the input with the given key
*/
func (ce *ConstructEvaluator) evaluateInputRules(o InfraOwner) error {
	for _, rule := range o.GetInputRules() {
		dv := &DynamicValueData{
			currentOwner: o,
		}

		if err := ce.evaluateInputRule(dv, rule); err != nil {
			return fmt.Errorf("could not evaluate input rule: %w", err)
		}
	}
	return nil
}

func (ce *ConstructEvaluator) evaluateInputRule(dv *DynamicValueData, rule template.InputRuleTemplate) error {
	if rule.ForEach != "" {
		return ce.evaluateForEachRule(dv, rule)
	}
	return ce.evaluateIfRule(dv, rule)
}

/*
Evaluation Order:

	Construct Inputs
	Construct Input Rules
	Construct Resources
	Construct Edges
	Binding Priorities
	Binding Inputs
	Binding Input Rules
	Binding Resources
	Binding Edges
*/
func (ce *ConstructEvaluator) evaluateConstruct(constructUrn model.URN, state model.State, ctx context.Context) (*Construct, error) {

	cState, ok := state.Constructs[constructUrn.ResourceID]
	if !ok {
		return nil, fmt.Errorf("could not get state state for construct: %s", constructUrn)
	}

	inputs, err := ce.convertInputs(cState.Inputs)
	if err != nil {
		return nil, fmt.Errorf("invalid inputs for construct: %w", err)

	}
	c, err := ce.newConstruct(constructUrn, inputs)
	if err != nil {
		return nil, fmt.Errorf("could not create construct: %w", err)
	}
	ce.Constructs.Set(constructUrn, c)

	if err = ce.initBindings(c, state); err != nil {
		return nil, fmt.Errorf("could not initialize bindings: %w", err)
	}

	if err = ce.importResourcesFromInputs(c, ctx); err != nil {
		return nil, fmt.Errorf("could not import resources: %w", err)
	}

	if err = ce.evaluateResources(c); err != nil {
		return nil, fmt.Errorf("could not evaluate resources: %w", err)
	}

	if err = ce.evaluateEdges(c); err != nil {
		return nil, fmt.Errorf("could not evaluate edges: %w", err)
	}

	if err = ce.evaluateInputRules(c); err != nil {
		return nil, err
	}

	if err = ce.evaluateOutputs(c); err != nil {
		return nil, fmt.Errorf("could not evaluate outputs: %w", err)
	}

	return c, nil
}

func (ce *ConstructEvaluator) getBindingDeclarations(constructURN model.URN, state model.State) ([]BindingDeclaration, error) {
	var bindings []BindingDeclaration
	var err error
	for _, c := range state.Constructs {
		if c.URN.Equals(constructURN) {
			for _, b := range c.Bindings {
				bindings = append(bindings, newBindingDeclaration(constructURN, b))
			}
			continue
		}
		for _, b := range c.Bindings {
			if b.URN.Equals(constructURN) {
				bindings = append(bindings, newBindingDeclaration(*c.URN, b))
			}
		}
	}
	return bindings, err
}

func newBindingDeclaration(constructURN model.URN, b model.Binding) BindingDeclaration {
	return BindingDeclaration{
		From:   constructURN,
		To:     *b.URN,
		Inputs: b.Inputs,
	}
}

func (ce *ConstructEvaluator) initBindings(c *Construct, state model.State) error {
	declarations, err := ce.getBindingDeclarations(c.URN, state)
	if err != nil {
		return fmt.Errorf("could not get bindings: %w", err)
	}

	for _, d := range declarations {
		if !d.From.Equals(c.URN) && !d.To.Equals(c.URN) {
			return fmt.Errorf("binding %s -> %s is not valid on construct of type %s", d.From, d.To, c.ConstructTemplate.Id)
		}

		if _, ok := d.Inputs["from"]; ok {
			return errors.New("from is a reserved input name")
		}
		if _, ok := d.Inputs["to"]; ok {
			return errors.New("to is a reserved input name")
		}

		b, err := ce.newBinding(c.URN, d)
		if err != nil {
			return fmt.Errorf("could not create binding: %w", err)
		}

		c.Bindings = append(c.Bindings, b)
	}
	return nil
}

func (ce *ConstructEvaluator) evaluateBindings(ctx context.Context, c *Construct) error {
	for _, binding := range c.OrderedBindings() {
		if err := ce.evaluateBinding(ctx, binding); err != nil {
			return fmt.Errorf("could not evaluate binding: %w", err)
		}
	}

	return nil
}

func (ce *ConstructEvaluator) evaluateBinding(ctx context.Context, b *Binding) error {
	if b == nil {
		return fmt.Errorf("binding is nil")
	}
	owner := b.Owner
	if owner == nil {
		return fmt.Errorf("binding owner is nil")

	}
	if b.BindingTemplate.From.Name == "" || b.BindingTemplate.To.Name == "" {
		return nil // assume that this binding does not modify the current construct
	}

	if err := ce.importResourcesFromInputs(b, ctx); err != nil {
		return fmt.Errorf("could not import resources: %w", err)
	}

	if b.From != nil && owner.URN.Equals(b.From.GetURN()) {
		// only import "to" resources if the binding is from the current construct
		if err := ce.importBindingToResources(ctx, b); err != nil {
			return fmt.Errorf("could not import binding resources: %w", err)
		}
	}

	if err := ce.evaluateResources(b); err != nil {
		return fmt.Errorf("could not evaluate resources: %w", err)
	}

	if err := ce.evaluateEdges(b); err != nil {
		return fmt.Errorf("could not evaluate edges: %w", err)
	}

	if err := ce.evaluateInputRules(b); err != nil {
		return fmt.Errorf("could not evaluate input rules: %w", err)
	}

	if err := ce.evaluateOutputs(b); err != nil {
		return fmt.Errorf("could not evaluate outputs: %w", err)
	}

	if err := ce.applyBinding(b.Owner, b); err != nil {
		return fmt.Errorf("could not apply bindings: %w", err)
	}

	return nil
}

func (ce *ConstructEvaluator) evaluateEdges(o InfraOwner) error {
	dv := &DynamicValueData{
		currentOwner:   o,
		propertySource: o.GetPropertySource(),
	}

	for _, edge := range o.GetTemplateEdges() {
		e, err := ce.resolveEdge(dv, edge)
		if err != nil {
			return fmt.Errorf("could not resolve edge: %w", err)
		}
		o.SetEdges(append(o.GetEdges(), e))
	}
	return nil
}

// applyBinding applies the bindings to the construct by merging the resources, edges, and output declarations
// of the construct's bindings with the construct's resources, edges, and output declarations
func (ce *ConstructEvaluator) applyBinding(c *Construct, binding *Binding) error {
	log := logging.GetLogger(context.Background()).Sugar()

	// Merge resources
	for key, bRes := range binding.Resources {
		if res, exists := c.Resources[key]; exists {
			res.Properties = mergeProperties(res.Properties, bRes.Properties)
		} else {
			c.Resources[key] = bRes
		}
	}

	// Merge edges
	for _, edge := range binding.Edges {
		if !edgeExists(c.Edges, edge) {
			c.Edges = append(c.Edges, edge)
		}
	}

	// Merge output declarations
	for key, output := range binding.OutputDeclarations {
		if _, exists := c.OutputDeclarations[key]; !exists {
			c.OutputDeclarations[key] = output
		} else {
			// If output already exists, log a warning or handle the conflict as needed
			log.Warnf("Output %s already exists in construct, skipping binding output", key)
		}
	}

	// upsert the vertices
	ids, err := construct.TopologicalSort(binding.InitialGraph)
	if err != nil {
		return fmt.Errorf("could not topologically sort binding %s graph: %w", binding, err)
	}

	resources, err := construct.ResolveIds(binding.InitialGraph, ids)
	if err != nil {
		return fmt.Errorf("could not resolve ids from binding %s graph: %w", binding, err)
	}

	for _, vertex := range resources {
		if err := c.InitialGraph.AddVertex(vertex); err != nil {
			if errors.Is(err, graph.ErrVertexAlreadyExists) {
				log.Debugf("Vertex already exists, skipping: %v", vertex)
				continue
			}
			return fmt.Errorf("could not add vertex %v from binding %s graph: %w", vertex, binding, err)
		}
	}

	// upsert the edges
	edges, err := binding.InitialGraph.Edges()
	if err != nil {
		return fmt.Errorf("could not get edges from binding %s graph: %w", binding, err)
	}

	for _, edge := range edges {
		// Attempt to add the edge to the initial graph
		err = c.InitialGraph.AddEdge(edge.Source, edge.Target)
		if err != nil {
			if errors.Is(err, graph.ErrEdgeAlreadyExists) {
				// Skip this edge if it already exists
				log.Debugf("Edge already exists, skipping: %v -> %v\n", edge.Source, edge.Target)
				continue
			}
			return fmt.Errorf("could not add edge %v -> %v from binding %s graph: %w", edge.Source, edge.Target, binding, err)
		}
	}

	return nil
}

func mergeProperties(existing, new construct.Properties) construct.Properties {
	merged := make(construct.Properties)

	for k, v := range existing {
		merged[k] = v
	}
	for k, v := range new {
		// If property exists in both, prefer the new value
		merged[k] = v
	}

	return merged
}

func edgeExists(edges []*Edge, newEdge *Edge) bool {
	for _, edge := range edges {
		if edge.From == newEdge.From && edge.To == newEdge.To {
			return true
		}
	}
	return false
}

func (ce *ConstructEvaluator) evaluateResources(o InfraOwner) error {
	var err error
	dv := &DynamicValueData{
		currentOwner:   o,
		propertySource: o.GetPropertySource(),
	}

	ri := o.GetTemplateResourcesIterator()
	ri.ForEach(func(key string, resource template.ResourceTemplate) error {
		var r *Resource
		r, err = ce.resolveResource(dv, key, resource)
		if err != nil {
			return template.StopIteration
		}
		o.SetResource(key, r)
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func GetPropertyFunc(ps *template.PropertySource, path string) func(string) any {
	return func(key string) any {
		i, ok := ps.GetProperty(fmt.Sprintf("%s.%s", path, key))
		if !ok {
			return nil
		}
		return i
	}
}

func (ce *ConstructEvaluator) evaluateForEachRule(dv *DynamicValueData, rule template.InputRuleTemplate) error {
	parentPrefix := dv.resourceKeyPrefix

	ctx := DynamicValueContext{
		constructs: ce.Constructs,
	}
	var selected bool
	if err := ctx.ExecuteUnmarshal(rule.ForEach, dv, &selected); err != nil {
		return fmt.Errorf("result parsing failed: %w", err)
	}

	if !selected {
		return nil
	}

	for _, hasNext := dv.currentSelection.Next(); hasNext; _, hasNext = dv.currentSelection.Next() {
		prefix, err := ce.interpolateValue(dv, rule.Prefix)
		if err != nil {
			return fmt.Errorf("could not interpolate resource prefix: %w", err)
		}

		dv := &DynamicValueData{
			currentOwner:     dv.currentOwner,
			currentSelection: dv.currentSelection,
			propertySource:   dv.propertySource,
		}

		if prefix != "" && prefix != nil {
			if parentPrefix != "" {
				dv.resourceKeyPrefix = strings.Join([]string{parentPrefix, fmt.Sprintf("%s", prefix)}, ".")
			} else {
				dv.resourceKeyPrefix = fmt.Sprintf("%s", prefix)
			}
		} else {
			dv.resourceKeyPrefix = parentPrefix
		}

		ri := rule.Do.ResourcesIterator()
		ri.ForEach(func(key string, resource template.ResourceTemplate) error {
			if dv.resourceKeyPrefix != "" {
				key = fmt.Sprintf("%s.%s", dv.resourceKeyPrefix, key)
			}

			r, err := ce.resolveResource(dv, key, resource)
			if err != nil {
				return fmt.Errorf("could not resolve resource %s : %w", key, err)
			}
			dv.currentOwner.SetResource(key, r)
			return nil
		})

		for _, edge := range rule.Do.Edges {
			e, err := ce.resolveEdge(dv, edge)
			if err != nil {
				return fmt.Errorf("could not resolve edge: %w", err)
			}
			dv.currentOwner.SetEdges(append(dv.currentOwner.GetEdges(), e))
		}

		for _, rule := range rule.Do.Rules {
			if err := ce.evaluateInputRule(dv, rule); err != nil {
				return fmt.Errorf("could not evaluate input rule: %w", err)
			}

		}
	}

	return nil
}

func (ce *ConstructEvaluator) evaluateIfRule(dv *DynamicValueData, rule template.InputRuleTemplate) error {
	parentPrefix := dv.resourceKeyPrefix

	prefix, err := ce.interpolateValue(dv, rule.Prefix)
	if err != nil {
		return fmt.Errorf("could not interpolate resource prefix: %w", err)
	}

	dv = &DynamicValueData{
		currentOwner:     dv.currentOwner,
		currentSelection: dv.currentSelection,
		propertySource:   dv.propertySource,
	}

	if prefix != "" && prefix != nil {
		if parentPrefix != "" {
			dv.resourceKeyPrefix = strings.Join([]string{parentPrefix, fmt.Sprintf("%s", prefix)}, ".")
		} else {
			dv.resourceKeyPrefix = fmt.Sprintf("%s", prefix)
		}
	} else {
		dv.resourceKeyPrefix = parentPrefix
	}

	ctx := DynamicValueContext{
		constructs: ce.Constructs,
	}

	var boolResult bool
	err = ctx.ExecuteUnmarshal(rule.If, dv, &boolResult)

	if err != nil {
		return fmt.Errorf("result parsing failed: %w", err)
	}
	executeThen := boolResult

	var body template.ConditionalExpressionTemplate
	if executeThen && rule.Then != nil {
		body = *rule.Then
	} else if rule.Else != nil {
		body = *rule.Else
	}

	ri := body.ResourcesIterator()
	ri.ForEach(func(key string, resource template.ResourceTemplate) error {
		if dv.resourceKeyPrefix != "" {
			key = fmt.Sprintf("%s.%s", dv.resourceKeyPrefix, key)
		}

		r, err := ce.resolveResource(dv, key, resource)
		if err != nil {
			return fmt.Errorf("could not resolve resource %s: %w", key, err)
		}
		dv.currentOwner.SetResource(key, r)
		return nil
	})

	for _, edge := range body.Edges {
		e, err := ce.resolveEdge(dv, edge)
		if err != nil {
			return fmt.Errorf("could not resolve edge: %w", err)
		}
		dv.currentOwner.SetEdges(append(dv.currentOwner.GetEdges(), e))
	}

	for _, rule := range body.Rules {
		if err := ce.evaluateInputRule(dv, rule); err != nil {
			return fmt.Errorf("could not evaluate input rule: %w", err)
		}
	}

	return nil
}

func (ce *ConstructEvaluator) resolveResource(dv *DynamicValueData, key string, rt template.ResourceTemplate) (*Resource, error) {
	// update the resource if it already exists
	if dv.currentOwner == nil {
		return nil, fmt.Errorf("current owner is nil")
	}
	resource, ok := dv.currentOwner.GetResource(key)
	if !ok {
		resource = &Resource{Properties: map[string]any{}}
	}

	tmpl, err := ce.interpolateValue(dv, rt)
	if err != nil {
		return nil, fmt.Errorf("could not interpolate resource %s: %w", key, err)
	}

	resTmpl := tmpl.(template.ResourceTemplate)
	typeParts := strings.Split(resTmpl.Type, ":")
	if len(typeParts) != 2 && resTmpl.Type != "" {
		return nil, fmt.Errorf("invalid resource type: %s", resTmpl.Type)
	}

	if len(typeParts) == 2 {
		provider := typeParts[0]
		resourceType := typeParts[1]

		id := construct.ResourceId{
			Provider:  provider,
			Type:      resourceType,
			Namespace: resTmpl.Namespace,
			Name:      resTmpl.Name,
		}
		if resource.Id == (construct.ResourceId{}) {
			resource.Id = id
		} else if resource.Id != id {
			return nil, fmt.Errorf("resource id mismatch: %s", key)
		}
	}

	// #TODO: deep merge the properties by evaluating the properties recursively
	// merge the properties
	for k, v := range resTmpl.Properties {
		// if the base resource does not have the property, set the property
		if resource.Properties[k] == nil {
			resource.Properties[k] = v
			continue
		}
		// if the property is a map, merge the map
		vt := reflect.TypeOf(v)
		switch vt.Kind() {
		case reflect.Map:
			for mk, mv := range v.(map[string]any) {
				resource.Properties[k].(map[string]any)[mk] = mv
			}
		case reflect.Slice:
			for _, mv := range v.([]any) {
				resource.Properties[k] = append(resource.Properties[k].([]any), mv)
			}
		default:
			resource.Properties[k] = v
		}
	}
	return resource, nil
}

func (ce *ConstructEvaluator) resolveEdge(dv *DynamicValueData, edge template.EdgeTemplate) (*Edge, error) {
	from, err := ce.interpolateValue(dv, edge.From)
	if err != nil {
		return nil, err
	}
	if from == nil {
		return nil, fmt.Errorf("from is nil")
	}
	to, err := ce.interpolateValue(dv, edge.To)
	if err != nil {
		return nil, err
	}
	if to == nil {
		return nil, fmt.Errorf("to is nil")
	}
	data, err := ce.interpolateValue(dv, edge.Data)
	if err != nil {
		return nil, err
	}

	return &Edge{
		From: from.(template.ResourceRef),
		To:   to.(template.ResourceRef),
		Data: data.(construct.EdgeData),
	}, nil
}

func (ce *ConstructEvaluator) evaluateOutputs(o InfraOwner) error {
	// sort the keys of the outputs alphabetically to ensure deterministic ordering
	sortKeys := func(m map[string]template.OutputTemplate) []string {
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		return keys
	}

	outputs := o.GetTemplateOutputs()
	keys := sortKeys(outputs)
	for _, key := range keys {
		ot := outputs[key]
		dv := &DynamicValueData{
			currentOwner:   o,
			propertySource: o.GetPropertySource(),
		}
		output, err := ce.interpolateValue(dv, ot)
		if err != nil {
			return fmt.Errorf("failed to interpolate value for output %s: %w", key, err)
		}

		outputTemplate, ok := output.(template.OutputTemplate)
		if !ok {
			return fmt.Errorf("invalid output template for output %s", key)
		}

		var value any
		var ref construct.PropertyRef

		r, ok := outputTemplate.Value.(template.ResourceRef)
		if !ok {
			value = outputTemplate.Value
		} else {
			serializedRef, err := ce.marshalRef(o, r)
			if err != nil {
				return fmt.Errorf("failed to serialize ref for output %s: %w", key, err)
			}

			var refString string
			if sr, ok := serializedRef.(string); ok {
				refString = sr
			} else if sr, ok := serializedRef.(fmt.Stringer); ok {
				refString = sr.String()
			} else {
				return fmt.Errorf("invalid ref string for output %s", key)
			}

			err = ref.Parse(refString)
			if err != nil {
				return fmt.Errorf("failed to parse ref string for output %s: %w", key, err)
			}
		}

		if ref != (construct.PropertyRef{}) && value != nil {
			return fmt.Errorf("output declaration must be a reference or a value for output %s", key)
		}

		o.DeclareOutput(key, OutputDeclaration{
			Name:  key,
			Ref:   ref,
			Value: value,
		})
	}
	return nil
}

func (ce *ConstructEvaluator) convertInputs(inputs map[string]model.Input) (construct.Properties, error) {
	props := make(construct.Properties)
	for k, v := range inputs {
		if ce.DryRun == 0 && v.Status != model.InputStatusResolved {
			return nil, fmt.Errorf("input %s is not resolved", k)
		}
		props[k] = v.Value
	}
	return props, nil
}

type HasInputs interface {
	ForEachInput(f func(input property.Property) error) error
	GetInputs() construct.Properties
}

func (ce *ConstructEvaluator) initializeInputs(c HasInputs, i construct.Properties) error {
	var inputErrors error
	_ = c.ForEachInput(func(input property.Property) error {
		v, err := i.GetProperty(input.Details().Path)
		if err == nil {
			if (v == nil || v == input.ZeroValue()) && input.Details().Required {
				inputErrors = errors.Join(inputErrors, fmt.Errorf("input %s is required", input.Details().Path))
				return nil
			}
			if err = input.SetProperty(c.GetInputs(), v); err != nil {
				inputErrors = errors.Join(inputErrors, err)
				return nil
			}
		} else if errors.Is(err, construct.ErrPropertyDoesNotExist) {
			if dv, err := input.GetDefaultValue(DynamicValueContext{}, nil); err == nil {
				if dv == nil {
					dv = input.ZeroValue()
				}
				if (dv == nil || dv == input.ZeroValue()) && input.Details().Required {
					inputErrors = errors.Join(inputErrors, fmt.Errorf("input %s is required", input.Details().Path))
					return nil
				}
				if dv == nil {
					return nil // no default value (e.g., for collections or other types with type arguments, i.e., generics)
				}
				if err = input.SetProperty(c.GetInputs(), dv); err != nil {
					inputErrors = errors.Join(inputErrors, err)
					return nil
				}
			}
		} else {
			inputErrors = errors.Join(inputErrors, err)
		}
		return nil
	})
	return inputErrors
}
