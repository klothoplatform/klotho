package constructs

import (
	"bytes"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/klothoplatform/klotho/pkg/construct"
)

type (
	ConstructContext struct {
		Id                ConstructId
		ConstructTemplate ConstructTemplate
		Meta              map[string]any
		Inputs            map[string]any
		Resources         map[string]*Resource
		Edges             []*Edge
		Outputs           map[string]any
	}

	ResourceRef struct {
		ResourceKey string
		Property    string
		Type        ResourceRefType
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
)

var (
	ResourceInterpolationContext  = InterpolationContext{InputsInterpolation, ResourcesInterpolation, ResourcesInterpolation}
	EdgeInterpolationContext      = InterpolationContext{InputsInterpolation, ResourcesInterpolation, EdgesInterpolation}
	OutputInterpolationContext    = InterpolationContext{InputsInterpolation, ResourcesInterpolation, EdgesInterpolation, MetaInterpolation}
	InputRuleInterpolationContext = InterpolationContext{InputsInterpolation, ResourcesInterpolation, EdgesInterpolation, MetaInterpolation}
)

// NewConstructContext creates a new ConstructContext instance
func NewConstructContext(constructId ConstructId, inputs map[string]any) *ConstructContext {
	return &ConstructContext{
		Id:                constructId,
		ConstructTemplate: loadConstructTemplate(constructId.TemplateId),
		Meta:              map[string]any{},
		Inputs:            inputs,
		Resources:         map[string]*Resource{},
		Edges:             []*Edge{},
		Outputs:           map[string]any{},
	}
}

/* InterpolateValue interpolates a value based on the context of the construct
	rawValue is the value to interpolate. The format of a raw value is ${<prefix>:<key>} where prefix is the type of value to interpolate and key is the key to interpolate

	The key can be a path to a value in the context. For example, ${inputs:foo.bar} will interpolate the value of the key bar in the foo input.
	The target of a dot-separated path can be a map or a struct.
	The path can also include brackets to access an array. For example, ${inputs:foo[0].bar} will interpolate the value of the key bar in the first element of the foo input array.

	Allowable prefixes are:
	- inputs: Interpolates a value from the inputs of the construct
	- resources: Interpolates a value from the resources of the construct
    - edges: Interpolates a value from the edges of the construct
    - meta: Interpolates a value from the metadata of the construct

    A rawValue can contain a combination of interpolation expressions and literals. For example, "${inputs:foo.bar}-baz-${resource:Boz}" is a valid rawValue.
*/

var interpolationPattern = regexp.MustCompile(`\$\{([^:]+):([^}]+)}`)
var isolatedInterpolationPattern = regexp.MustCompile(`^\$\{([^:]+):([^}]+)}$`)

func (c *ConstructContext) InterpolateValue(rawValue any, ctx InterpolationContext) (any, error) {
	if ref, ok := rawValue.(ResourceRef); ok {
		if ref.Type == ResourceRefTypeInterpolated {
			return c.InterpolateValue(ref.ResourceKey, ctx)
		}
		return rawValue, nil
	}

	v := reflect.ValueOf(rawValue)

	// If the value is a pointer, dereference it
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
		rawValue = v.Interface()
	}

	switch v.Kind() {
	case reflect.String:
		return c.interpolateString(v.String(), ctx)
	case reflect.Slice:
		length := v.Len()
		interpolated := make([]interface{}, length)
		for i := 0; i < length; i++ {
			value, err := c.InterpolateValue(v.Index(i).Interface(), ctx)
			if err != nil {
				return nil, err
			}
			interpolated[i] = value
		}
		return interpolated, nil
	case reflect.Map:
		keys := v.MapKeys()
		interpolated := make(map[string]interface{})
		for _, k := range keys {
			key, err := c.InterpolateValue(k.Interface(), ctx)
			if err != nil {
				return nil, err
			}
			value, err := c.InterpolateValue(v.MapIndex(k).Interface(), ctx)
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
			fieldValue, err := c.InterpolateValue(v.Field(i).Interface(), ctx)
			if err != nil {
				return nil, err
			}
			// Set the interpolated value to the field in the new struct
			newStruct.FieldByName(fieldName).Set(reflect.ValueOf(fieldValue))
		}

		// Return the new struct
		return newStruct.Interface(), nil
	default:
		return rawValue, nil
	}
}

func (c *ConstructContext) interpolateString(rawValue string, ctx InterpolationContext) (any, error) {

	// if the rawValue is an isolated interpolation expression, interpolate it and return the raw value
	if isolatedInterpolationPattern.MatchString(rawValue) {
		return c.interpolateExpression(rawValue, ctx), nil
	}

	// Replace each match in the rawValue
	interpolated := interpolationPattern.ReplaceAllStringFunc(rawValue, func(match string) string {
		return fmt.Sprint(c.interpolateExpression(match, ctx))
	})

	return interpolated, nil
}

func (c *ConstructContext) interpolateExpression(match string, ctx InterpolationContext) any {
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

	// If the value is a Resource, return a ResourceRef
	if _, ok := value.(*Resource); ok {
		return ResourceRef{
			ResourceKey: key,
			Type:        ResourceRefTypeTemplate,
		}
	}

	// Replace the match with the value
	return value
}

// iacRefPattern is a regular expression pattern that matches an IaC reference
// IaC references are in the format <resource-key>#<property>

var iacRefPattern = regexp.MustCompile(`^([a-zA-Z0-9_-]+)#([a-zA-Z0-9._-]+)$`)

func getValueFromCollection(collection any, key string) interface{} {
	var value any = collection

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
			case reflect.Slice:
				value = reflect.ValueOf(value).Index(index).Interface()
			case reflect.Map:
				value = value.(map[string]interface{})[fmt.Sprint(index)]
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
						Type:        ResourceRefTypeTemplate,
					}
				} else {
					value = r.Properties[part]
				}
			} else {
				return nil
			}
		}
	}

	return value
}

func (c *ConstructContext) interpolateValue(rawValue any, ctx InterpolationContext) (any, error) {
	switch v := rawValue.(type) {
	case string:
		return c.interpolateString(v, ctx)
	case []any:
		for i, val := range v {
			newVal, err := c.interpolateValue(val, ctx)
			if err != nil {
				return nil, err
			}
			v[i] = newVal
		}
		return v, nil
	case map[string]interface{}:
		for key, val := range v {
			newVal, err := c.interpolateValue(val, ctx)
			if err != nil {
				return nil, err
			}
			v[key] = newVal
		}
		return v, nil
	case struct{}:
		val := reflect.ValueOf(v)
		for i := 0; i < val.NumField(); i++ {
			newVal, err := c.interpolateValue(val.Field(i).Interface(), ctx)
			if err != nil {
				return nil, err
			}
			val.Field(i).Set(reflect.ValueOf(newVal))
		}
		return val.Interface(), nil
	default:
		return rawValue, nil
	}
}

// parse inputs
func (c *ConstructContext) parseInputs() {
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
func (c *ConstructContext) evaluateInputRules() {
	for _, rule := range c.ConstructTemplate.InputRules {
		if err := c.evaluateInputRule(rule); err != nil {
			panic(err)
		}
	}
}

func (c *ConstructContext) EvaluateConstruct() *Construct {
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

	c.parseInputs()
	c.evaluateResources()
	c.evaluateEdges()
	c.evaluateInputRules()

	return &Construct{
		Id:        c.Id,
		Inputs:    c.Inputs,
		Resources: c.Resources,
		Edges:     c.Edges,
		Outputs:   c.Outputs,
	}

}

func (c *ConstructContext) evaluateEdges() {
	for _, edge := range c.ConstructTemplate.Edges {
		c.Edges = append(c.Edges, c.resolveEdge(edge))
	}
}

func (c *ConstructContext) evaluateResources() {

	c.ConstructTemplate.ResourcesIterator().ForEach(func(key string, resource ResourceTemplate) {
		c.Resources[key] = c.resolveResource(key, resource)

	})
}

func (c *ConstructContext) input(key string) any {
	return c.Inputs[key]
}

func (c *ConstructContext) templateFunctions() template.FuncMap {
	funcs := template.FuncMap{}
	funcs["input"] = c.input
	return funcs
}

func (c *ConstructContext) evaluateInputRule(rule InputRuleTemplate) error {
	tmpl, err := template.New("input_rule").Funcs(c.templateFunctions()).Parse(
		fmt.Sprintf("{{ if %s }}true{{ else }}false{{ end }}", rule.If),
	)
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
		c.Resources[key] = c.resolveResource(key, resource)
	}

	for key, resource := range body.Resources {
		rp, err := c.InterpolateValue(resource, InputRuleInterpolationContext)
		if err != nil {
			panic(err)
		}
		r := rp.(ResourceTemplate)
		c.Resources[key] = c.resolveResource(key, r)
	}

	for _, edge := range body.Edges {
		c.Edges = append(c.Edges, c.resolveEdge(edge))

	}

	return nil
}

func (c *ConstructContext) resolveResource(key string, rt ResourceTemplate) *Resource {

	// update the resource if it already exists
	resource := c.Resources[key]
	if resource == nil {
		resource = &Resource{Properties: map[string]any{}}
	}

	tmpl, err := c.InterpolateValue(rt, ResourceInterpolationContext)
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

func (c *ConstructContext) resolveEdge(edge EdgeTemplate) *Edge {
	from, err := c.InterpolateValue(edge.From, EdgeInterpolationContext)
	if err != nil {
		panic(err)
	}
	to, err := c.InterpolateValue(edge.To, EdgeInterpolationContext)
	if err != nil {
		panic(err)
	}
	data, err := c.InterpolateValue(edge.Data, EdgeInterpolationContext)
	if err != nil {
		panic(err)
	}

	return &Edge{
		From: from.(ResourceRef),
		To:   to.(ResourceRef),
		Data: data.(map[string]any),
	}
}

func (c *ConstructContext) serializeRef(ref ResourceRef) (any, error) {
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
