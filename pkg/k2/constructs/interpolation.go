package constructs

import (
	"errors"
	"fmt"
	template2 "github.com/klothoplatform/klotho/pkg/k2/constructs/template"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/klothoplatform/klotho/pkg/reflectutil"
	"go.uber.org/zap"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// Matches one or more interpolation groups in a string e.g., ${inputs:foo.bar}-baz-${resource:Boz}
var interpolationPattern = regexp.MustCompile(`\$\{([^:]+):([^}]+)}`)

// Matches exactly one interpolation group e.g., ${inputs:foo.bar}
var isolatedInterpolationPattern = regexp.MustCompile(`^\$\{([^:]+):([^}]+)}$`)

var spreadPattern = regexp.MustCompile(`\.\.\.}$`)

// interpolateValue interpolates a value based on the context of the construct
//
// The format of a raw value is ${<prefix>:<key>} where prefix is the type of value to interpolate and key is the key to interpolate
//
// The key can be a path to a value in the context.
// For example, ${inputs:foo.bar} will interpolate the value of the key bar in the foo input.
//
// The target of a dot-separated path can be a map or a struct.
// The path can also include brackets to access an array or an element whose key contains a dot.
// For example, ${inputs:foo[0].bar} will interpolate the value of the key bar in the first element of the foo input array.
//
// The path can also include a spread operator to expand an array into the current array.
// For example, ${inputs:foo...} will expand the foo input array into the current array.
//
// A rawValue can contain a combination of interpolation expressions, literals, and go templates.
// For example, "${inputs:foo.bar}-baz-${resource:Boz}" is a valid rawValue.
func (ce *ConstructEvaluator) interpolateValue(dv *DynamicValueData, rawValue any) (any, error) {
	if ref, ok := rawValue.(template2.ResourceRef); ok {
		switch ref.Type {
		case template2.ResourceRefTypeInterpolated:
			return ce.interpolateValue(dv, ref.ResourceKey)
		case template2.ResourceRefTypeTemplate:
			ref.ConstructURN = dv.currentOwner.GetURN()
			rk, err := ce.interpolateValue(dv, ref.ResourceKey)
			if err != nil {
				return nil, err
			}
			ref.ResourceKey = fmt.Sprintf("%s", rk)
			return ref, nil
		default:
			return rawValue, nil
		}
	}

	v := reflectutil.GetConcreteElement(reflect.ValueOf(rawValue))
	if !v.IsValid() {
		return rawValue, nil
	}
	rawValue = v.Interface()

	switch v.Kind() {
	case reflect.String:
		resolvedVal, err := ce.interpolateString(dv, v.String())
		if err != nil {
			return nil, err
		}
		return resolvedVal, nil
	case reflect.Slice:
		length := v.Len()
		var interpolated []any
		for i := 0; i < length; i++ {
			// handle spread operator by injecting the spread value into the array at the current index
			originalValue := reflectutil.GetConcreteValue(v.Index(i))
			if originalString, ok := originalValue.(string); ok && spreadPattern.MatchString(originalString) {
				unspreadPath := originalString[:len(originalString)-4] + "}"
				spreadValue, err := ce.interpolateValue(dv, unspreadPath)
				if err != nil {
					return nil, err
				}

				if spreadValue == nil {
					continue
				}
				if reflect.TypeOf(spreadValue).Kind() != reflect.Slice {
					return nil, errors.New("spread value must be a slice")
				}

				for i := 0; i < reflect.ValueOf(spreadValue).Len(); i++ {
					interpolated = append(interpolated, reflect.ValueOf(spreadValue).Index(i).Interface())
				}
				continue
			}
			value, err := ce.interpolateValue(dv, v.Index(i).Interface())
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
			key, err := ce.interpolateValue(dv, k.Interface())
			if err != nil {
				return nil, err
			}
			value, err := ce.interpolateValue(dv, v.MapIndex(k).Interface())
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
			fieldValue, err := ce.interpolateValue(dv, v.Field(i).Interface())
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

func (ce *ConstructEvaluator) interpolateString(dv *DynamicValueData, rawValue string) (any, error) {
	// handle go template expressions
	if strings.Contains(rawValue, "{{") {
		ctx := DynamicValueContext{constructs: ce.Constructs}
		err := ctx.ExecuteUnmarshal(rawValue, dv, &rawValue)
		if err != nil {
			return nil, err
		}
	}

	ps := dv.propertySource
	if ps == nil {
		ps = dv.currentOwner.GetPropertySource()
	}

	// if the rawValue is an isolated interpolation expression, interpolate it and return the raw value
	if isolatedInterpolationPattern.MatchString(rawValue) {
		return ce.interpolateExpression(dv.currentOwner, ps, rawValue)
	}

	var err error

	// Replace each match in the rawValue (mixed expressions are always interpolated as strings)
	interpolated := interpolationPattern.ReplaceAllStringFunc(rawValue, func(match string) string {
		var val any
		val, err = ce.interpolateExpression(dv.currentOwner, ps, match)
		return fmt.Sprint(val)
	})
	if err != nil {
		return nil, err
	}

	return interpolated, nil
}

func (ce *ConstructEvaluator) interpolateExpression(owner InfraOwner, ps *template2.PropertySource, match string) (any, error) {
	if ps == nil {
		return nil, errors.New("property source is nil")
	}

	// Split the match into prefix and key
	parts := interpolationPattern.FindStringSubmatch(match)
	prefix := parts[1]
	key := parts[2]

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

	prefixParts := strings.Split(prefix, ".")

	// associate any ResourceRefs with the URN of the property source they're being interpolated from
	// if the prefix is "from" or "to", the URN of the property source is the "urn" field of that level in the property source
	var refUrn model.URN

	if strings.HasSuffix(prefix, "resources") {
		urnKey := "urn"
		if prefixParts[0] == "from" || prefixParts[0] == "to" {
			urnKey = fmt.Sprintf("%s.urn", prefixParts[0])
		}
		psURN, ok := template2.GetTypedProperty[model.URN](ps, urnKey)
		if !ok {
			psURN = owner.GetURN()
		}
		refUrn = psURN
	} else {
		propTrace, err := reflectutil.TracePath(reflect.ValueOf(p), key)
		if err == nil {
			refConstruct, ok := reflectutil.LastOfType[*Construct](propTrace)
			if ok {
				refUrn = refConstruct.URN
			}
		}
		if refUrn.Equals(model.URN{}) {
			refUrn = owner.GetURN()
		}
	}

	// return an IaC reference if the key matches the IaC reference pattern
	if iacRefPattern.MatchString(key) {
		return template2.ResourceRef{
			ResourceKey:  iacRefPattern.FindStringSubmatch(key)[1],
			Property:     iacRefPattern.FindStringSubmatch(key)[2],
			Type:         template2.ResourceRefTypeIaC,
			ConstructURN: refUrn,
		}, nil
	}

	// special cases for resources allowing for accessing the name of a resource directly instead of using .Id.Name
	if prefix == "resources" || prefixParts[len(prefixParts)-1] == "resources" {
		keyParts := reflectutil.SplitPath(key)
		resourceKey := strings.Trim(keyParts[0], ".[]")
		if len(keyParts) > 1 {
			if path := keyParts[1]; path == ".Name" {
				return p.(map[string]*Resource)[resourceKey].Id.Name, nil
			}

		}
	}

	// Retrieve the value from the designated property source
	value, err := ce.getValueFromSource(p, key, false)
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
		return template2.ResourceRef{
			ResourceKey:  r.Id.String(),
			Property:     refProperty,
			Type:         template2.ResourceRefTypeIaC,
			ConstructURN: refUrn,
		}, nil
	}

	if r, ok := value.(template2.ResourceRef); ok {
		r.ConstructURN = refUrn
		return r, nil
	}

	// Replace the match with the value
	return value, nil
}

// iacRefPattern is a regular expression pattern that matches an IaC reference
// IaC references are in the format <resource-key>#<property>
var iacRefPattern = regexp.MustCompile(`^([a-zA-Z0-9_-]+)#([a-zA-Z0-9._-]+)$`)

// indexPattern is a regular expression pattern that matches an array index in the format `[index]`
var indexPattern = regexp.MustCompile(`^\[\d+]$`)

// getValueFromSource retrieves a value from a property source based on a key
// the flat parameter is used to determine if the key is a flat key or a path (mixed keys aren't supported at the moment)
// e.g (flat = true): key = "foo.bar" -> value = collection["foo."bar"], flat = false: key = "foo.bar" -> value = collection["foo"]["bar"]
func (ce *ConstructEvaluator) getValueFromSource(source any, key string, flat bool) (any, error) {
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
		parts = reflectutil.SplitPath(key)
	}
	for i, part := range parts {
		parts[i] = strings.TrimPrefix(part, ".")
	}

	var err error
	var lastValidValue reflect.Value
	lastValidIndex := -1

	// Traverse the map/struct/array according to the parts
	for i, part := range parts {
		// Check if the part is an array index
		if indexPattern.MatchString(part) {
			// Split the part into the key and the index
			part = strings.TrimSuffix(strings.TrimPrefix(part, "["), "]")
			var index int
			index, err = strconv.Atoi(part)
			if err != nil {
				err = fmt.Errorf("could not parse index: %w", err)
				break
			}

			value = reflectutil.GetConcreteElement(value)
			kind := value.Kind()

			switch kind {
			case reflect.Slice | reflect.Array:
				if index >= value.Len() {
					err = fmt.Errorf("index out of bounds: %d", index)
					break
				}
				value = value.Index(index)
			default:
				err = fmt.Errorf("invalid type: %s", kind)
			}
		} else {
			// The part is not an array index
			part = strings.TrimSuffix(strings.TrimPrefix(part, "["), "]")

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
					return template2.ResourceRef{
						ResourceKey: part,
						Property:    refProperty,
						Type:        template2.ResourceRefTypeTemplate,
					}, nil
				} else {
					// if the parent is a resource, children are implicitly properties of the resource
					lastValidValue = reflect.ValueOf(r.Properties)
					value, err = reflectutil.GetField(lastValidValue, part)
					if err != nil {
						err = fmt.Errorf("could not get field: %w", err)
						break
					}
				}
			} else if u, ok := value.Interface().(model.URN); ok {
				if c, ok := ce.Constructs.Get(u); ok {
					lastValidValue = reflect.ValueOf(c)
					value, err = reflectutil.GetField(lastValidValue, part)
					if err != nil {
						err = fmt.Errorf("could not get field: %w", err)
						break
					}
				} else {
					err = fmt.Errorf("could not get construct: %s", u)
					break
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
		return ce.getValueFromSource(lastValidValue.Interface(), strings.Join(parts[lastValidIndex+1:], "."), true)
	}

	return value.Interface(), err
}
