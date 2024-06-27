package constructs

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	stateconverter "github.com/klothoplatform/klotho/pkg/infra/state_reader/state_converter"
	statetemplate "github.com/klothoplatform/klotho/pkg/infra/state_reader/state_template"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/klothoplatform/klotho/pkg/k2/reflectutil"
	"github.com/klothoplatform/klotho/pkg/k2/stack"
	"go.uber.org/zap"
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

func (ce *ConstructEvaluator) Evaluate(constructUrn model.URN) (constraints.Constraints, error) {
	ci, err := ce.evaluateConstruct(constructUrn)
	if err != nil {
		return constraints.Constraints{}, fmt.Errorf("error evaluating construct: %w", err)
	}

	marshaller := ConstructMarshaller{Construct: ci}
	constraintList, err := marshaller.Marshal()
	if err != nil {
		return constraints.Constraints{}, fmt.Errorf("error marshalling construct to constraints: %w", err)
	}

	cs, err := constraintList.ToConstraints()
	if err != nil {
		return constraints.Constraints{}, fmt.Errorf("error converting constraint list to constraints: %w", err)
	}

	return cs, nil
}

var interpolationPattern = regexp.MustCompile(`\$\{([^:]+):([^}]+)}`)
var isolatedInterpolationPattern = regexp.MustCompile(`^\$\{([^:]+):([^}]+)}$`)

/*
	 interpolateValue interpolates a value based on the context of the construct
		rawValue is the value to interpolate. The format of a raw value is ${<prefix>:<key>} where prefix is the type of value to interpolate and key is the key to interpolate

		The key can be a path to a value in the context. For example, ${inputs:foo.bar} will interpolate the value of the key bar in the foo input.
		The target of a dot-separated path can be a map or a struct.
		The path can also include brackets to access an array. For example, ${inputs:foo[0].bar} will interpolate the value of the key bar in the first element of the foo input array.

		Allowable prefixes are:
		- stack: Interpolates a value from a construct's IaC (pulumi) stack
		- inputs: Interpolates a value from the inputs of the construct
		- resources: Interpolates a value from the resources of the construct
	    - edges: Interpolates a value from the edges of the construct
	    - meta: Interpolates a value from the metadata of the construct

	    A rawValue can contain a combination of interpolation expressions and literals. For example, "${inputs:foo.bar}-baz-${resource:Boz}" is a valid rawValue.
*/
func (ce *ConstructEvaluator) interpolateValue(c *Construct, rawValue any, ctx InterpolationContext) (any, error) {
	if ref, ok := rawValue.(ResourceRef); ok {
		if ref.Type == ResourceRefTypeInterpolated {
			return ce.interpolateValue(c, ref.ResourceKey, ctx)
		}
		return rawValue, nil
	}

	v := reflectutil.GetConcreteElement(reflect.ValueOf(rawValue))
	rawValue = v.Interface()

	switch v.Kind() {
	case reflect.String:
		return ce.interpolateString(c, v.String(), ctx)
	case reflect.Slice:
		length := v.Len()
		interpolated := make([]any, length)
		for i := 0; i < length; i++ {
			value, err := ce.interpolateValue(c, v.Index(i).Interface(), ctx)
			if err != nil {
				return nil, err
			}
			interpolated[i] = value
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

func (ce *ConstructEvaluator) interpolateString(c *Construct, rawValue string, ctx InterpolationContext) (any, error) {

	// if the rawValue is an isolated interpolation expression, interpolate it and return the raw value
	if isolatedInterpolationPattern.MatchString(rawValue) {
		return ce.interpolateExpression(c, rawValue, ctx), nil
	}

	// Replace each match in the rawValue
	interpolated := interpolationPattern.ReplaceAllStringFunc(rawValue, func(match string) string {
		return fmt.Sprint(ce.interpolateExpression(c, match, ctx))
	})

	return interpolated, nil
}

func (ce *ConstructEvaluator) interpolateExpression(c *Construct, match string, ctx InterpolationContext) any {
	// Split the match into prefix and key
	parts := interpolationPattern.FindStringSubmatch(match)
	prefix := parts[1]
	key := parts[2]

	// Check if the prefix is allowed
	allowed := false
	for _, p := range ctx {
		if p == InterpolationSource(prefix) {
			allowed = true
			break
		}
	}
	if !allowed {
		return ""
	}

	// Choose the correct map based on the prefix
	var m any
	switch prefix {
	case "inputs":
		m = c.Inputs
	case "resources":
		m = c.Resources
	case "edges":
		m = c.Edges
	case "meta":
		m = c.Meta
	default:
		return ""
	}

	// return an IaC reference if the key matches the IaC reference pattern
	if iacRefPattern.MatchString(key) {
		return ResourceRef{
			ResourceKey: iacRefPattern.FindStringSubmatch(key)[1],
			Property:    iacRefPattern.FindStringSubmatch(key)[2],
			Type:        ResourceRefTypeIaC,
		}
	}

	// special cases for resources
	if prefix == "resources" {
		keyParts := strings.SplitN(key, ".", 2)
		resourceKey := keyParts[0]
		if len(keyParts) > 1 {
			if path := keyParts[1]; path == "Name" {
				return m.(map[string]*Resource)[resourceKey].Id.Name
			}

		}
	}

	// Retrieve the value from the map
	value := getValueFromCollection(m, key)

	keyAndRef := strings.Split(key, "#")
	var refProperty string
	if len(keyAndRef) == 2 {
		refProperty = keyAndRef[1]
	}

	// If the value is a Resource, return a ResourceRef
	if r, ok := value.(*Resource); ok {
		if prefix == "inputs" {
			return ResourceRef{
				ResourceKey: r.Id.String(),
				Property:    refProperty,
				Type:        ResourceRefTypeIaC,
			}
		}

		return ResourceRef{
			ResourceKey: key,
			Property:    refProperty,
			Type:        ResourceRefTypeTemplate,
		}
	}

	// Replace the match with the value
	return value
}

// iacRefPattern is a regular expression pattern that matches an IaC reference
// IaC references are in the format <resource-key>#<property>

var iacRefPattern = regexp.MustCompile(`^([a-zA-Z0-9_-]+)#([a-zA-Z0-9._-]+)$`)

func getValueFromCollection(collection any, key string) any {
	value := collection

	keyAndRef := strings.Split(key, "#")
	if len(keyAndRef) > 2 {
		return nil
	}

	var refProperty string
	if len(keyAndRef) == 2 {
		refProperty = keyAndRef[1]
		key = keyAndRef[0]
	}

	// Split the key into parts
	parts := strings.Split(key, ".")

	// Traverse the map/struct/array according to the parts
	for _, part := range parts {
		// Check if the part contains brackets
		if strings.Contains(part, "[") && strings.Contains(part, "]") {
			// Split the part into the key and the index
			keyAndIndex := strings.Split(strings.TrimRight(strings.TrimLeft(part, "["), "]"), "[")
			key := keyAndIndex[0]
			index, err := strconv.Atoi(keyAndIndex[1])
			if err != nil {
				return nil
			}

			value = collection.(map[string]any)[key]
			kind := reflect.TypeOf(value).Kind()

			switch kind {
			case reflect.Slice | reflect.Array:
				value = reflect.ValueOf(value).Index(index).Interface()
			case reflect.Map:
				value = value.(map[string]any)[fmt.Sprint(index)]
			default:
				return nil
			}
		} else {
			// The part does not contain brackets
			mr := reflect.ValueOf(value)
			if mr.Kind() == reflect.Map {
				v := mr.MapIndex(reflect.ValueOf(part))
				if v.IsValid() {
					value = v.Interface()
				} else {
					return nil
				}
			} else if r, ok := value.(*Resource); ok {
				if len(parts) == 1 {
					return ResourceRef{
						ResourceKey: part,
						Property:    refProperty,
						Type:        ResourceRefTypeTemplate,
					}
				} else {
					value = r.Properties[part]
				}
			} else {
				rVal, err := reflectutil.GetField(reflect.ValueOf(value), part)
				if err != nil {
					return nil
				}
				value = rVal.Interface()
			}
		}
	}

	return value
}

// parse inputs
func (ce *ConstructEvaluator) parseInputs(c *Construct) {
	for key, value := range c.ConstructTemplate.Inputs {
		if _, hasVal := c.Inputs[key]; !hasVal && value.Default != nil {
			c.Inputs[key] = value.Default
		}
	}
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
func (ce *ConstructEvaluator) evaluateInputRules(c *Construct) {
	for _, rule := range c.ConstructTemplate.InputRules {
		if err := ce.evaluateInputRule(c, rule); err != nil {
			panic(err)
		}
	}
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
func (ce *ConstructEvaluator) evaluateConstruct(constructUrn model.URN) (*Construct, error) {

	cState, ok := ce.stateManager.GetConstructState(constructUrn.ResourceID)
	if !ok {
		return nil, fmt.Errorf("could not get state state for construct: %s", constructUrn)
	}

	inputs := make(map[string]any)
	for k, v := range cState.Inputs {
		if v.Status != "" && v.Status != model.InputStatusResolved {
			return nil, fmt.Errorf("input '%s' is not resolved", k)
		}

		if iURN, ok := v.Value.(model.URN); ok {
			ic, ok := ce.constructs[iURN]
			if !ok {
				return nil, fmt.Errorf("could not find construct %s", iURN)
			}
			inputs[k] = ic
		} else {
			inputs[k] = v.Value
		}
	}

	c, err := NewConstruct(constructUrn, inputs)
	if err != nil {
		return nil, fmt.Errorf("could not create construct: %w", err)
	}
	ce.constructs[constructUrn] = c

	ce.parseInputs(c)
	err = ce.importResources(c)
	if err != nil {
		return nil, fmt.Errorf("could not import resources: %w", err)
	}
	ce.evaluateResources(c)
	ce.evaluateEdges(c)
	ce.evaluateInputRules(c)
	ce.evaluateOutputs(c)

	return c, nil
}

func (ce *ConstructEvaluator) evaluateEdges(c *Construct) {
	for _, edge := range c.ConstructTemplate.Edges {
		c.Edges = append(c.Edges, ce.resolveEdge(c, edge))
	}
}

func (ce *ConstructEvaluator) evaluateResources(c *Construct) {

	c.ConstructTemplate.ResourcesIterator().ForEach(func(key string, resource ResourceTemplate) {
		c.Resources[key] = ce.resolveResource(c, key, resource)
	})
}

// getInputFunc generates a template function that returns the value of the current construct's input with the given key
func getInputFunc(c *Construct) func(string) any {
	return func(key string) any {
		return c.Inputs[key]
	}
}

func (ce *ConstructEvaluator) templateFunctions(c *Construct) template.FuncMap {
	funcs := template.FuncMap{}
	funcs["inputs"] = getInputFunc(c)
	return funcs
}

func (ce *ConstructEvaluator) evaluateInputRule(c *Construct, rule InputRuleTemplate) error {
	tmpl, err := template.New("input_rule").Funcs(ce.templateFunctions(c)).Parse(rule.If)
	if err != nil {
		return fmt.Errorf("could not parse template: %w", err)
	}
	var rawResult bytes.Buffer
	if err := tmpl.Execute(&rawResult, nil); err != nil {
		panic(err)
	}

	// TODO: look into additional handling for nil rawResult
	boolResult, _ := strconv.ParseBool(rawResult.String())
	executeThen := boolResult

	var body ConditionalExpressionTemplate
	if executeThen {
		body = rule.Then
	} else {
		body = rule.Else
	}

	// add raw resources to the context
	for key, resource := range body.Resources {
		c.Resources[key] = ce.resolveResource(c, key, resource)
	}

	for key, resource := range body.Resources {
		rp, err := ce.interpolateValue(c, resource, InputRuleInterpolationContext)
		if err != nil {
			panic(err)
		}
		r := rp.(ResourceTemplate)
		c.Resources[key] = ce.resolveResource(c, key, r)
	}

	for _, edge := range body.Edges {
		c.Edges = append(c.Edges, ce.resolveEdge(c, edge))

	}

	return nil
}

func (ce *ConstructEvaluator) resolveResource(c *Construct, key string, rt ResourceTemplate) *Resource {

	// update the resource if it already exists
	resource := c.Resources[key]
	if resource == nil {
		resource = &Resource{Properties: map[string]any{}}
	}

	tmpl, err := ce.interpolateValue(c, rt, ResourceInterpolationContext)
	if err != nil {
		panic(err)
	}

	resTmpl := tmpl.(ResourceTemplate)
	typeParts := strings.Split(resTmpl.Type, ":")
	if len(typeParts) != 2 && resTmpl.Type != "" {
		panic("Invalid resource type: " + resTmpl.Type)
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
			panic("Resource id mismatch")
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

	return resource
}

func (ce *ConstructEvaluator) resolveEdge(c *Construct, edge EdgeTemplate) *Edge {
	from, err := ce.interpolateValue(c, edge.From, EdgeInterpolationContext)
	if err != nil {
		panic(err)
	}
	to, err := ce.interpolateValue(c, edge.To, EdgeInterpolationContext)
	if err != nil {
		panic(err)
	}
	data, err := ce.interpolateValue(c, edge.Data, EdgeInterpolationContext)
	if err != nil {
		panic(err)
	}

	return &Edge{
		From: from.(ResourceRef),
		To:   to.(ResourceRef),
		Data: data.(map[string]any),
	}
}

func (ce *ConstructEvaluator) evaluateOutputs(c *Construct) {
	for key, output := range c.ConstructTemplate.Outputs {
		output, err := ce.interpolateValue(c, output, OutputInterpolationContext)
		if err != nil {
			panic(err)
		}
		outputTemplate, ok := output.(OutputTemplate)
		if !ok {
			panic("invalid output template")
		}
		var value any
		var ref construct.PropertyRef
		r, ok := outputTemplate.Value.(ResourceRef)
		if !ok {
			value = outputTemplate.Value
		} else {
			serializedRef, err := c.SerializeRef(r)
			if err != nil {
				panic(err)
			}

			refString, ok := serializedRef.(string)
			if !ok {
				panic("invalid ref")
			}
			err = ref.Parse(refString)
			if err != nil {
				panic(err)
			}
		}

		if ref != (construct.PropertyRef{}) && value != nil {
			panic("output declaration must be a reference or a value")
		}

		c.OutputDeclarations[key] = OutputDeclaration{
			Name:  key,
			Ref:   ref,
			Value: value,
		}
	}
}

var constructTypePattern = regexp.MustCompile(`^Construct<([\w.-]+)>$`)

func (ce *ConstructEvaluator) importResources(c *Construct) error {
	for iName, i := range c.ConstructTemplate.Inputs {
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

		resolvedInput, ok := c.Inputs[iName]
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
			c.ImportedResources[rId] = cState.Properties
			zap.S().Infof("imported resource %s", rId)
		}
	}
	return nil
}

func loadStateConverter() (stateconverter.StateConverter, error) {
	templates, err := statetemplate.LoadStateTemplates("pulumi")
	if err != nil {
		return nil, err
	}

	return stateconverter.NewStateConverter("pulumi", templates), nil
}
