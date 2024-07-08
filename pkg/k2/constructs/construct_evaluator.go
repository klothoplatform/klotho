package constructs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"go.uber.org/zap"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	stateconverter "github.com/klothoplatform/klotho/pkg/infra/state_reader/state_converter"
	statetemplate "github.com/klothoplatform/klotho/pkg/infra/state_reader/state_template"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/klothoplatform/klotho/pkg/k2/reflectutil"
	"github.com/klothoplatform/klotho/pkg/k2/stack"
	"github.com/klothoplatform/klotho/pkg/logging"
)

type ConstructEvaluator struct {
	constructs        map[model.URN]*Construct
	stateManager      *model.StateManager
	stackStateManager *stack.StateManager
	stateConverter    stateconverter.StateConverter
}

func NewConstructEvaluator(sm *model.StateManager, ssm *stack.StateManager) (*ConstructEvaluator, error) {
	stateConverter, err := loadStateConverter()
	if err != nil {
		return nil, err
	}

	return &ConstructEvaluator{
		constructs:        make(map[model.URN]*Construct),
		stateManager:      sm,
		stackStateManager: ssm,
		stateConverter:    stateConverter,
	}, nil
}

func (ce *ConstructEvaluator) Evaluate(constructUrn model.URN, state model.State, ctx context.Context) (constraints.Constraints, error) {
	ci, err := ce.evaluateConstruct(constructUrn, state, ctx)
	if err != nil {
		return constraints.Constraints{}, fmt.Errorf("error evaluating construct %s: %w", constructUrn, err)
	}
	err = ce.evaluateBindings(ci, ctx)
	if err != nil {
		return constraints.Constraints{}, fmt.Errorf("error evaluating bindings: %w", err)
	}

	marshaller := ConstructMarshaller{ConstructEvaluator: ce}
	constraintList, err := marshaller.Marshal(constructUrn)
	if err != nil {
		return constraints.Constraints{}, fmt.Errorf("error marshalling construct to constraints: %w", err)
	}

	cs, err := constraintList.ToConstraints()
	if err != nil {
		return constraints.Constraints{}, fmt.Errorf("error converting constraint list to constraints: %w", err)
	}

	return cs, nil
}

// Matches one or more interpolation groups in a string e.g., ${inputs:foo.bar}-baz-${resource:Boz}
var interpolationPattern = regexp.MustCompile(`\$\{([^:]+):([^}]+)}`)

// Matches exactly one interpolation group e.g., ${inputs:foo.bar}
var isolatedInterpolationPattern = regexp.MustCompile(`^\$\{([^:]+):([^}]+)}$`)

var spreadPattern = regexp.MustCompile(`\.\.\.}$`)

/*
	 interpolateValue interpolates a value based on the context of the construct
		rawValue is the value to interpolate. The format of a raw value is ${<prefix>:<key>} where prefix is the type of value to interpolate and key is the key to interpolate

		The key can be a path to a value in the context. For example, ${inputs:foo.bar} will interpolate the value of the key bar in the foo input.
		The target of a dot-separated path can be a map or a struct.
		The path can also include brackets to access an array. For example, ${inputs:foo[0].bar} will interpolate the value of the key bar in the first element of the foo input array.
	    A rawValue can contain a combination of interpolation expressions and literals. For example, "${inputs:foo.bar}-baz-${resource:Boz}" is a valid rawValue.
*/
func (ce *ConstructEvaluator) interpolateValue(c InterpolationSource, rawValue any, ctx InterpolationContext) (any, error) {
	if ref, ok := rawValue.(ResourceRef); ok {
		switch ref.Type {
		case ResourceRefTypeInterpolated:
			return ce.interpolateValue(c, ref.ResourceKey, ctx)
		case ResourceRefTypeTemplate:
			ref.ConstructURN = ctx.Construct.URN
			return ref, nil
		default:
			return rawValue, nil
		}
	}

	v := reflectutil.GetConcreteElement(reflect.ValueOf(rawValue))
	rawValue = v.Interface()

	switch v.Kind() {
	case reflect.String:
		return ce.interpolateString(c.GetPropertySource(), v.String(), ctx)
	case reflect.Slice:
		length := v.Len()
		var interpolated []any
		for i := 0; i < length; i++ {
			// handle spread operator by injecting the spread value into the array at the current index
			originalValue := reflectutil.GetConcreteValue(v.Index(i))
			if originalString, ok := originalValue.(string); ok && spreadPattern.MatchString(originalString) {
				unspreadPath := originalString[:len(originalString)-4] + "}"
				spreadValue, err := ce.interpolateValue(c, unspreadPath, ctx)
				if err != nil {
					return nil, err
				}
				if reflect.TypeOf(spreadValue).Kind() != reflect.Slice {
					return nil, errors.New("spread value must be a slice")
				}
				for i := 0; i < reflect.ValueOf(spreadValue).Len(); i++ {
					interpolated = append(interpolated, reflect.ValueOf(spreadValue).Index(i).Interface())
				}
				continue
			}
			value, err := ce.interpolateValue(c, v.Index(i).Interface(), ctx)
			if err != nil {
				return nil, err
			}
			interpolated = append(interpolated, value)
		}
		return interpolated, nil
	case reflect.Map:
		keys := v.MapKeys()
		interpolated := make(map[string]any)
		for _, k := range keys {
			key, err := ce.interpolateValue(c, k.Interface(), ctx)
			if err != nil {
				return nil, err
			}
			value, err := ce.interpolateValue(c, v.MapIndex(k).Interface(), ctx)
			if err != nil {
				return nil, err
			}
			interpolated[fmt.Sprint(key)] = value
		}
		return interpolated, nil
	case reflect.Struct:
		// Create a new instance of the struct
		newStruct := reflect.New(v.Type()).Elem()

		// Interpolate each field
		for i := 0; i < v.NumField(); i++ {
			fieldName := v.Type().Field(i).Name
			fieldValue, err := ce.interpolateValue(c, v.Field(i).Interface(), ctx)
			if err != nil {
				return nil, err
			}
			// Set the interpolated value to the field in the new struct
			if fieldValue != nil {
				newStruct.FieldByName(fieldName).Set(reflect.ValueOf(fieldValue))
			}
		}

		// Return the new struct
		return newStruct.Interface(), nil
	default:
		return rawValue, nil
	}
}

func (ce *ConstructEvaluator) interpolateString(ps *PropertySource, rawValue string, ctx InterpolationContext) (any, error) {

	// if the rawValue is an isolated interpolation expression, interpolate it and return the raw value
	if isolatedInterpolationPattern.MatchString(rawValue) {
		return ce.interpolateExpression(ps, rawValue, ctx)
	}

	var err error

	// Replace each match in the rawValue (mixed expressions are always interpolated as strings)
	interpolated := interpolationPattern.ReplaceAllStringFunc(rawValue, func(match string) string {
		var val any
		val, err = ce.interpolateExpression(ps, match, ctx)
		return fmt.Sprint(val)
	})
	if err != nil {
		return nil, err
	}

	return interpolated, nil
}

func (ce *ConstructEvaluator) interpolateExpression(ps *PropertySource, match string, ctx InterpolationContext) (any, error) {
	if ps == nil {
		return nil, errors.New("property source is nil")
	}

	// Split the match into prefix and key
	parts := interpolationPattern.FindStringSubmatch(match)
	prefix := parts[1]
	key := parts[2]

	// Check if the prefix is allowed
	allowed := false
	for _, p := range ctx.AllowedKeys {
		if p == InterpolationSourceKey(prefix) || p == FromInterpolation && strings.HasPrefix(prefix, "from.") || p == ToInterpolation && strings.HasPrefix(prefix, "to.") {
			allowed = true
			break
		}
	}
	if !allowed {
		return "", fmt.Errorf("interpolation prefix '%s' is not allowed in the current context", prefix)
	}

	// Choose the correct root property from the source based on the prefix
	var p any
	ok := false
	if prefix == "inputs" || prefix == "resources" || prefix == "edges" || prefix == "meta" ||
		strings.HasPrefix(prefix, "from.") ||
		strings.HasPrefix(prefix, "to.") {
		p, ok = ps.GetProperty(prefix)
		if !ok {
			return nil, fmt.Errorf("could not get %s", prefix)

		}
	} else {
		return nil, fmt.Errorf("invalid prefix: %s", prefix)
	}

	// return an IaC reference if the key matches the IaC reference pattern
	if iacRefPattern.MatchString(key) {
		return ResourceRef{
			ResourceKey:  iacRefPattern.FindStringSubmatch(key)[1],
			Property:     iacRefPattern.FindStringSubmatch(key)[2],
			Type:         ResourceRefTypeIaC,
			ConstructURN: ctx.Construct.URN,
		}, nil
	}

	prefixParts := strings.Split(prefix, ".")

	// special cases for resources allowing for accessing the name of a resource directly instead of using .Id.Name
	if prefix == "resources" || prefixParts[len(prefixParts)-1] == "resources" {
		keyParts := strings.SplitN(key, ".", 2)
		resourceKey := keyParts[0]
		if len(keyParts) > 1 {
			if path := keyParts[1]; path == "Name" {
				return p.(map[string]*Resource)[resourceKey].Id.Name, nil
			}

		}
	}

	// associate any ResourceRefs with the URN of the property source they're being interpolated from
	// if the prefix is "from" or "to", the URN of the property source is the "urn" field of that level in the property source
	urnKey := "urn"
	if prefixParts[0] == "from" || prefixParts[0] == "to" {
		urnKey = fmt.Sprintf("%s.urn", prefixParts[0])
	}
	psURN, ok := GetTypedProperty[model.URN](ps, urnKey)
	if !ok {
		psURN = ctx.Construct.URN
	}

	// Retrieve the value from the designated property source
	value, err := getValueFromSource(p, key, false)
	if err != nil {
		zap.S().Debugf("could not get value from source: %s", err)
		return nil, nil
	}

	keyAndRef := strings.Split(key, "#")
	var refProperty string
	if len(keyAndRef) == 2 {
		refProperty = keyAndRef[1]
	}

	// If the value is a Resource, return a ResourceRef
	if r, ok := value.(*Resource); ok {
		if prefix == "inputs" {
			return ResourceRef{
				ResourceKey:  r.Id.String(),
				Property:     refProperty,
				Type:         ResourceRefTypeIaC,
				ConstructURN: psURN,
			}, nil
		}

		return ResourceRef{
			ResourceKey:  key,
			Property:     refProperty,
			Type:         ResourceRefTypeTemplate,
			ConstructURN: psURN,
		}, nil
	}

	if r, ok := value.(ResourceRef); ok {
		r.ConstructURN = psURN
		return r, nil
	}

	// Replace the match with the value
	return value, nil
}

// iacRefPattern is a regular expression pattern that matches an IaC reference
// IaC references are in the format <resource-key>#<property>

var iacRefPattern = regexp.MustCompile(`^([a-zA-Z0-9_-]+)#([a-zA-Z0-9._-]+)$`)

// getValueFromSource retrieves a value from a property source based on a key
// the flat parameter is used to determine if the key is a flat key or a path (mixed keys aren't supported at the moment)
// e.g (flat = true): key = "foo.bar" -> value = collection["foo."bar"], flat = false: key = "foo.bar" -> value = collection["foo"]["bar"]
func getValueFromSource(source any, key string, flat bool) (any, error) {
	value := reflect.ValueOf(source)

	keyAndRef := strings.Split(key, "#")
	if len(keyAndRef) > 2 {
		return nil, fmt.Errorf("invalid engine reference property reference: %s", key)
	}

	var refProperty string
	if len(keyAndRef) == 2 {
		refProperty = keyAndRef[1]
		key = keyAndRef[0]
	}

	// Split the key into parts if not flat
	parts := []string{key}
	if !flat {
		parts = strings.Split(key, ".")
	}

	var err error
	var lastValidValue reflect.Value
	lastValidIndex := -1

	// Traverse the map/struct/array according to the parts
	for i, part := range parts {
		// Check if the part contains brackets
		if strings.Contains(part, "[") && strings.HasSuffix(part, "]") {
			// Split the part into the key and the index
			keyAndIndex := strings.Split(strings.TrimRight(strings.TrimLeft(part, "["), "]"), "[")
			key := keyAndIndex[0]
			var index int
			index, err = strconv.Atoi(keyAndIndex[1])
			if err != nil {
				err = fmt.Errorf("could not parse index: %w", err)
				break
			}

			if r, ok := value.Interface().(*Resource); ok {
				lastValidValue = reflect.ValueOf(r.Properties)
				value, err = reflectutil.GetField(lastValidValue, part)
			} else {
				value, err = reflectutil.GetField(value, key)
			}
			if err != nil {
				err = fmt.Errorf("could not get field: %w", err)
				break
			}

			kind := value.Kind()

			switch kind {
			case reflect.Slice | reflect.Array:
				value = value.Index(index)
			case reflect.Map:
				value, err = reflectutil.GetField(value, key)
				if err != nil {
					err = fmt.Errorf("could not get field: %w", err)
					break
				}
			default:
				err = fmt.Errorf("invalid type: %s", kind)
			}
		} else {
			// The part does not contain brackets
			if value.Kind() == reflect.Map {
				v := value.MapIndex(reflect.ValueOf(part))
				if v.IsValid() {
					value = v
				} else {
					err = fmt.Errorf("could not get value for key: %s", key)
					break
				}
			} else if r, ok := value.Interface().(*Resource); ok {
				if len(parts) == 1 {
					return ResourceRef{
						ResourceKey: part,
						Property:    refProperty,
						Type:        ResourceRefTypeTemplate,
					}, nil
				} else {
					lastValidValue = reflect.ValueOf(r.Properties)
					value, err = reflectutil.GetField(lastValidValue, part)
					if err != nil {
						err = fmt.Errorf("could not get field: %w", err)
						break
					}
				}
			} else {
				var rVal reflect.Value
				rVal, err = reflectutil.GetField(value, part)
				if err != nil {
					err = fmt.Errorf("could not get field: %w", err)
					break
				}
				value = rVal
			}
		}
		if err != nil {
			break
		}
		if i == len(parts)-1 {
			return value.Interface(), nil
		}

		lastValidValue = value
		lastValidIndex = i
	}

	if lastValidIndex > -1 {
		return getValueFromSource(lastValidValue.Interface(), strings.Join(parts[lastValidIndex+1:], "."), true)
	}

	return value.Interface(), err
}

/*
	evaluateInputRules evaluates the input rules of the construct

An input rule is a conditional expression that determines a set of resources, edges, and outputs based on the inputs of the construct
An input rule is evaluated by checking the if condition and then evaluating the then or else condition based on the result
the if condition is a go template that can access the inputs of the construct
input rules cannot use interpolation in the if condition

Example:
  - if: {{ eq input("foo") "bar" }}
    then:
    resources:
    "my-resource":
    properties:
    foo: "bar"

in the example input() is a function that returns the value of the input with the given key
*/
func (ce *ConstructEvaluator) evaluateInputRules(o InfraOwner, interpolationCtx InterpolationContext) error {
	for _, rule := range o.GetInputRules() {
		if err := ce.evaluateInputRule(o, rule, interpolationCtx); err != nil {
			return fmt.Errorf("could not evaluate input rule: %w", err)
		}
	}
	return nil
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
	Binding Conflict Resolvers
*/
func (ce *ConstructEvaluator) evaluateConstruct(constructUrn model.URN, state model.State, ctx context.Context) (*Construct, error) {

	cState, ok := state.Constructs[constructUrn.ResourceID]
	if !ok {
		return nil, fmt.Errorf("could not get state state for construct: %s", constructUrn)
	}

	inputs := make(map[string]any)

	templateId, err := ParseConstructTemplateId(constructUrn.Subtype)
	if err != nil {
		return nil, fmt.Errorf("could not parse construct template id: %w", err)

	}
	ct, err := loadConstructTemplate(templateId)
	if err != nil {
		return nil, fmt.Errorf("could not load construct template: %w", err)
	}
	for k, v := range cState.Inputs {
		inputTemplate, ok := ct.Inputs[k]
		if !ok {
			zap.S().Warnf("input %s not found in construct template", k)
		}
		v, err := ce.resolveInput(k, v, inputTemplate)
		if err != nil {
			return nil, err
		}
		inputs[k] = v
	}

	c, err := NewConstruct(constructUrn, inputs)
	if err != nil {
		return nil, fmt.Errorf("could not create construct: %w", err)
	}
	ce.constructs[constructUrn] = c

	if err = ce.initBindings(c, state); err != nil {
		return nil, fmt.Errorf("could not initialize bindings: %w", err)
	}

	if err = ce.importResources(c, ctx); err != nil {
		return nil, fmt.Errorf("could not import resources: %w", err)
	}

	if err = ce.evaluateResources(c, NewInterpolationContext(c, ResourceInterpolationContext)); err != nil {
		return nil, fmt.Errorf("could not evaluate resources: %w", err)
	}

	if err = ce.evaluateEdges(c, NewInterpolationContext(c, EdgeInterpolationContext)); err != nil {
		return nil, fmt.Errorf("could not evaluate edges: %w", err)
	}

	if err = ce.evaluateInputRules(c, NewInterpolationContext(c, InputRuleInterpolationContext)); err != nil {
		return nil, fmt.Errorf("could not evaluate input rules: %w", err)
	}

	if err = ce.evaluateOutputs(c, NewInterpolationContext(c, OutputInterpolationContext)); err != nil {
		return nil, fmt.Errorf("could not evaluate outputs: %w", err)
	}

	return c, nil
}

// resolveInput converts a model.Input to a construct.Input and adds it to the inputs map.
// If the value of the input is a URN, it resolves the URN to a construct.
// If the input's status is not "resolved", it returns an error.
func (ce *ConstructEvaluator) resolveInput(k string, v model.Input, t InputTemplate) (any, error) {
	if v.Status != "" && v.Status != model.InputStatusResolved {
		return nil, fmt.Errorf("input '%s' is not resolved", k)
	}
	var resolvedValue any
	switch {
	case strings.HasPrefix(t.Type, "Construct<"):
		cType := strings.TrimSuffix(strings.TrimPrefix(t.Type, "Construct<"), ">")

		if iURN, ok := v.Value.(model.URN); ok && iURN.IsResource() && iURN.Type == "construct" && iURN.Subtype == cType {
			ic, ok := ce.constructs[iURN]
			if !ok {
				return nil, fmt.Errorf("could not find construct %s", iURN)
			}
			resolvedValue = ic
		} else {
			return nil, fmt.Errorf("invalid construct URN: %v", v.Value)
		}
	case t.Type == "path":
		var err error
		pStr, ok := v.Value.(string)
		if !ok {
			return "", fmt.Errorf("invalid path type: expected string, got %T", v.Value)
		}
		resolvedValue, err = handlePathInput(pStr)
		if err != nil {
			return nil, err
		}
	default:
		resolvedValue = v.Value
	}
	return resolvedValue, nil
}

func handlePathInput(value string) (string, error) {
	if filepath.IsAbs(value) {
		return value, nil
	}

	// handle relative paths
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("could not get working directory")
	}
	return filepath.Join(wd, value), nil
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
				for _, b := range c.Bindings {
					bindings = append(bindings, newBindingDeclaration(*c.URN, b))
				}
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

		b, err := ce.newBinding(c.URN, d.From, d.To)
		if err != nil {
			return fmt.Errorf("could not create binding: %w", err)
		}

		inputs := make(map[string]any)
		for key, inputTemplate := range b.BindingTemplate.Inputs {
			mVal, ok := d.Inputs[key]
			if !ok {
				continue
			}
			if mVal.Status != "" && mVal.Status != model.InputStatusResolved {
				return fmt.Errorf("input '%s' is not resolved", key)
			}
			resolvedValue, err := ce.resolveInput(key, mVal, inputTemplate)
			if err != nil {
				return fmt.Errorf("could not resolve input: %w", err)
			}
			inputs[key] = resolvedValue
		}
		populateDefaultInputValues(inputs, b.BindingTemplate.Inputs)
		b.Inputs = inputs

		c.Bindings = append(c.Bindings, b)
	}
	return nil
}

func (ce *ConstructEvaluator) evaluateBindings(c *Construct, ctx context.Context) error {
	for _, binding := range c.OrderedBindings() {
		if err := ce.evaluateBinding(c, binding, ctx); err != nil {
			return fmt.Errorf("could not evaluate binding: %w", err)
		}
	}
	if err := ce.applyBindings(c); err != nil {
		return fmt.Errorf("could not apply bindings: %w", err)
	}

	return nil
}

func (ce *ConstructEvaluator) evaluateBinding(c *Construct, b *Binding, ctx context.Context) error {
	if b.BindingTemplate.From.Name == "" || b.BindingTemplate.To.Name == "" {
		return nil // assume that this binding does not modify the current construct
	}
	var err error
	if err = ce.importResources(b, ctx); err != nil {
		return fmt.Errorf("could not import resources: %w", err)
	}

	if b.From != nil && c.URN.Equals(b.From.GetURN()) {
		// only import "to" resources if the binding is from the current construct
		if err = ce.importBindingToResources(b, ctx); err != nil {
			return fmt.Errorf("could not import binding resources: %w", err)
		}
	}

	interpolationCtx := NewInterpolationContext(c, BindingInterpolationContext)

	if err = ce.evaluateResources(b, interpolationCtx); err != nil {
		return fmt.Errorf("could not evaluate resources: %w", err)
	}

	if err = ce.evaluateEdges(b, interpolationCtx); err != nil {
		return fmt.Errorf("could not evaluate edges: %w", err)
	}

	if err = ce.evaluateInputRules(b, interpolationCtx); err != nil {
		return fmt.Errorf("could not evaluate input rules: %w", err)
	}

	if err = ce.evaluateOutputs(b, interpolationCtx); err != nil {
		return fmt.Errorf("could not evaluate outputs: %w", err)
	}
	return nil
}

func (ce *ConstructEvaluator) evaluateEdges(c InfraOwner, interpolationCtx InterpolationContext) error {
	for _, edge := range c.GetTemplateEdges() {
		e, err := ce.resolveEdge(c, edge, interpolationCtx)
		if err != nil {
			return fmt.Errorf("could not resolve edge: %w", err)
		}
		c.SetEdges(append(c.GetEdges(), e))
	}
	return nil
}

// applyBindings applies the bindings to the construct by merging the resources, edges, and output declarations
// of the construct's bindings with the construct's resources, edges, and output declarations
func (ce *ConstructEvaluator) applyBindings(c *Construct) error {
	for _, binding := range c.Bindings {
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
				logging.GetLogger(context.Background()).Sugar().Warnf("Output %s already exists in construct, skipping binding output", key)
			}
		}

		// Merge imported resources
		for id, properties := range binding.ImportedResources {
			res, exists := c.ImportedResources[id]
			if !exists {
				c.ImportedResources[id] = properties
			} else {
				c.ImportedResources[id] = mergeProperties(res, properties)
			}
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

func (ce *ConstructEvaluator) evaluateResources(o ResourceOwner, interpolationCtx InterpolationContext) error {
	var err error
	i := o.GetTemplateResourcesIterator()
	i.ForEach(func(key string, resource ResourceTemplate) error {
		var r *Resource
		r, err = ce.resolveResource(o, key, resource, interpolationCtx)
		if err != nil {
			return stopIteration
		}
		o.SetResource(key, r)
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func GetPropertyFunc(ps *PropertySource) func(string) any {
	return func(key string) any {
		i, ok := ps.GetProperty(fmt.Sprintf("inputs.%s", key))
		if !ok {
			return nil
		}
		return i
	}
}

func (ce *ConstructEvaluator) templateFunctions(ps *PropertySource) template.FuncMap {
	funcs := template.FuncMap{}
	funcs["inputs"] = GetPropertyFunc(ps)
	return funcs
}

func (ce *ConstructEvaluator) evaluateInputRule(o InfraOwner, rule InputRuleTemplate, interpolationCtx InterpolationContext) error {
	tmpl, err := template.New("input_rule").Funcs(ce.templateFunctions(o.GetPropertySource())).Parse(rule.If)
	if err != nil {
		return fmt.Errorf("template parsing failed: %w", err)
	}
	var rawResult bytes.Buffer
	if err := tmpl.Execute(&rawResult, nil); err != nil {
		return fmt.Errorf("template execution failed: %w", err)
	}

	boolResult, err := strconv.ParseBool(rawResult.String())
	if err != nil {
		return fmt.Errorf("result parsing failed: %w", err)
	}
	executeThen := boolResult

	var body ConditionalExpressionTemplate
	if executeThen {
		body = rule.Then
	} else {
		body = rule.Else
	}

	// add raw resources to the context
	for key, resource := range body.Resources {
		r, err := ce.resolveResource(o, key, resource, interpolationCtx)
		if err != nil {
			return fmt.Errorf("could not resolve resource %s: %w", key, err)
		}
		o.SetResource(key, r)
	}

	for key, resource := range body.Resources {
		rp, err := ce.interpolateValue(o, resource, interpolationCtx)
		if err != nil {
			return fmt.Errorf("could not interpolate resource %s: %w", key, err)
		}
		rt := rp.(ResourceTemplate)

		r, err := ce.resolveResource(o, key, rt, interpolationCtx)
		if err != nil {
			return fmt.Errorf("could not resolve resource %s : %w", key, err)
		}
		o.SetResource(key, r)
	}

	for _, edge := range body.Edges {
		e, err := ce.resolveEdge(o, edge, interpolationCtx)
		if err != nil {
			return fmt.Errorf("could not resolve edge: %w", err)
		}
		o.SetEdges(append(o.GetEdges(), e))
	}
	return nil
}

func (ce *ConstructEvaluator) resolveResource(o ResourceOwner, key string, rt ResourceTemplate, interpolationCtx InterpolationContext) (*Resource, error) {
	// update the resource if it already exists
	resource, ok := o.GetResource(key)
	if !ok {
		resource = &Resource{Properties: map[string]any{}}
	}

	tmpl, err := ce.interpolateValue(o, rt, interpolationCtx)
	if err != nil {
		return nil, fmt.Errorf("could not interpolate resource %s: %w", key, err)
	}

	resTmpl := tmpl.(ResourceTemplate)
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

func (ce *ConstructEvaluator) resolveEdge(c InfraOwner, edge EdgeTemplate, interpolationCtx InterpolationContext) (*Edge, error) {
	from, err := ce.interpolateValue(c, edge.From, interpolationCtx)
	if err != nil {
		return nil, err
	}
	to, err := ce.interpolateValue(c, edge.To, interpolationCtx)
	if err != nil {
		return nil, err
	}
	data, err := ce.interpolateValue(c, edge.Data, interpolationCtx)
	if err != nil {
		return nil, err
	}

	return &Edge{
		From: from.(ResourceRef),
		To:   to.(ResourceRef),
		Data: data.(construct.EdgeData),
	}, nil
}

func (ce *ConstructEvaluator) evaluateOutputs(o InfraOwner, interpolationCtx InterpolationContext) error {
	for key, output := range o.GetTemplateOutputs() {
		output, err := ce.interpolateValue(o, output, interpolationCtx)
		if err != nil {
			return fmt.Errorf("failed to interpolate value for output %s: %w", key, err)
		}

		outputTemplate, ok := output.(OutputTemplate)
		if !ok {
			return fmt.Errorf("invalid output template for output %s", key)
		}

		var value any
		var ref construct.PropertyRef

		r, ok := outputTemplate.Value.(ResourceRef)
		if !ok {
			value = outputTemplate.Value
		} else {
			serializedRef, err := ce.serializeRef(r)
			if err != nil {
				return fmt.Errorf("failed to serialize ref for output %s: %w", key, err)
			}

			refString, ok := serializedRef.(string)
			if !ok {
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

var constructTypePattern = regexp.MustCompile(`^Construct<([\w.-]+)>$`)

func (ce *ConstructEvaluator) importResources(o InfraOwner, ctx context.Context) error {
	log := logging.GetLogger(ctx).Sugar()
	importedResources := o.GetImportedResources()
	for iName, i := range o.GetTemplateInputs() {
		// parse construct type from input type in the form of Construct<type>
		// get the construct from the evaluator if it exists and is the correct type or return an error
		// then go through the resources of the construct and add them to the imported resources of the current construct
		// if the resource is not found, return an error
		if i.Type == "Construct" {
			return errors.New("input of type Construct must have a type specified in the form of Construct<type>")
		}
		typeMatch := constructTypePattern.FindStringSubmatch(i.Type)
		if len(typeMatch) == 0 {
			continue // skip the input if it is not a construct
		}

		resolvedInput, ok := o.GetInput(iName)
		if !ok {
			return fmt.Errorf("could not find resolved input %s", iName)
		}

		ic, ok := resolvedInput.(*Construct)
		if !ok {
			return fmt.Errorf("value %v of input %s is not a construct", iName, resolvedInput)
		}

		// TODO: DS - consider whether to include transitive resource imports
		stackState, ok := ce.stackStateManager.ConstructStackState[ic.URN]
		if !ok {
			return fmt.Errorf("could not find stack state for construct %s", ic.URN)
		}
		for rId, state := range stackState.Resources {
			cState, err := ce.stateConverter.ConvertResource(stateconverter.Resource{
				Urn:     string(state.URN),
				Type:    string(state.Type),
				Outputs: state.Outputs,
			})
			if err != nil {
				return fmt.Errorf("could not convert state: %w", err)
			}
			importedResources[rId] = cState.Properties
			log.Infof("imported resource %s", rId)
		}
	}
	return nil
}

func (ce *ConstructEvaluator) importBindingToResources(b *Binding, ctx context.Context) error {
	log := logging.GetLogger(ctx).Sugar()
	importedResources := b.GetImportedResources()
	// TODO: DS - consider whether to include transitive resource imports
	tc := b.To
	stackState, ok := ce.stackStateManager.ConstructStackState[tc.URN]
	if !ok {
		return fmt.Errorf("could not find stack state for construct %s", tc.URN)
	}
	for rId, state := range stackState.Resources {
		cState, err := ce.stateConverter.ConvertResource(stateconverter.Resource{
			Urn:     string(state.URN),
			Type:    string(state.Type),
			Outputs: state.Outputs,
		})
		if err != nil {
			return fmt.Errorf("could not convert state: %w", err)
		}
		if cState == nil {
			continue
		}
		importedResources[rId] = cState.Properties
		log.Infof("imported resource %s", rId)
	}
	return nil
}

func (ce *ConstructEvaluator) RegisterOutputValues(urn model.URN, outputs map[string]any) {
	if c, ok := ce.constructs[urn]; ok {
		c.Outputs = outputs
	}
}

func loadStateConverter() (stateconverter.StateConverter, error) {
	templates, err := statetemplate.LoadStateTemplates("pulumi")
	if err != nil {
		return nil, err
	}
	return stateconverter.NewStateConverter("pulumi", templates), nil
}
