package engine

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (e *Engine) ConfigureResource(graph *construct.ResourceGraph, resource construct.Resource) error {
	template := e.ResourceTemplates[construct.ResourceId{Provider: resource.Id().Provider, Type: resource.Id().Type}]
	if template != nil {
		err := TemplateConfigure(resource, *template, graph)
		if err != nil {
			return err

		}
	}

	err := graph.CallConfigure(resource, nil)
	if err != nil {
		return err

	}

	return nil
}

func TemplateConfigure(resource construct.Resource, template knowledgebase.ResourceTemplate, dag *construct.ResourceGraph) error {
	for _, config := range template.Configuration {
		field, _, err := parseFieldName(resource, config.Field, dag, true)
		if err != nil {
			return err
		}
		if (!field.IsValid() || !field.IsZero()) || config.ZeroValueAllowed {
			//since pointers will be non zero but could still be nil we need to check that case before proceeding
			if field.Kind() == reflect.Ptr && !field.IsNil() && !field.Elem().IsZero() {
				continue
			} else if field.Kind() != reflect.Ptr {
				continue
			}
		}
		err = ConfigureField(resource, config.Field, config.Value, config.ZeroValueAllowed, dag)
		if err != nil {
			return err
		}
	}
	return nil
}

// SetMapKey is a struct that represents a key in a map
// Because values of maps are not addressable, we need to store the map and the key separately
// then after we configure the sub field, we are able to go back and store that value in the map
type SetMapKey struct {
	Map   reflect.Value
	Key   reflect.Value
	Value reflect.Value
}

func (e *Engine) configureResource(context *SolveContext, r construct.Resource) {
	oldId := r.Id()

	configureComplete := make(map[string]struct{})
	for _, rc := range e.Context.Constraints[constraints.ResourceConstraintScope] {
		rc := rc.(*constraints.ResourceConstraint)
		if rc.Target != r.Id() {
			continue
		}
		config := knowledgebase.Configuration{Field: rc.Property, Value: rc.Value}
		configRule := knowledgebase.ConfigurationRule{Config: config, Resource: rc.Target}

		field, err := GetFieldByName(reflect.ValueOf(r), rc.Property)
		if err != nil {
			context.Errors = append(context.Errors, &ResourceConfigurationError{
				Resource:   r,
				Cause:      err,
				Config:     config,
				Constraint: rc,
			})
			continue
		}
		configureComplete[field.Name] = struct{}{}
		e.handleDecision(
			context,
			Decision{
				Level: LevelInfo,
				Result: &DecisionResult{
					Config:   &configRule,
					Resource: r,
				},
				Action: ActionConfigure,
				Cause:  &Cause{Constraint: rc},
			},
		)
	}

	tmpl := e.GetTemplateForResource(r)
	if tmpl == nil {
		return
	}
	for _, cfg := range tmpl.Configuration {
		if _, done := configureComplete[cfg.Field]; done {
			continue
		}
		if cfg.ValueTemplate != "" {
			ctx := knowledgebase.ConfigTemplateContext{DAG: context.ResourceGraph}
			data := knowledgebase.ConfigTemplateData{
				Resource: r.Id(),
			}
			var err error
			cfg, err = ctx.ResolveConfig(cfg, data)
			if err != nil {
				context.Errors = append(context.Errors, &ResourceConfigurationError{
					Resource: r,
					Cause:    err,
					Config:   cfg,
				})
				continue
			}
		}
		configRule := &knowledgebase.ConfigurationRule{Config: cfg, Resource: r.Id()}
		e.handleDecision(
			context,
			Decision{
				Level: LevelInfo,
				Result: &DecisionResult{
					Config:   configRule,
					Resource: r,
				},
				Action: ActionConfigure,
				Cause:  &Cause{ResourceConfiguration: r},
			},
		)
	}
	// Re-run make operational in case the configuration changed the requirements
	e.MakeResourceOperational(context, r)

	// If the ID changes, primarily caused by a namespace being added, update all the references.
	if oldId != r.Id() {
		err := context.ResourceGraph.ReplaceConstructId(oldId, r)
		if err != nil {
			context.Errors = append(context.Errors, &ResourceConfigurationError{
				Resource: r,
				Cause:    err,
			})
		}
	}
}

var resourceIdType = reflect.TypeOf(construct.ResourceId{})

// ConfigureField is a function that takes a resource, a field name, and a value and sets the field on the resource to the value
// It also takes a graph so that it can resolve references
// It returns an error if the field cannot be set
func ConfigureField(resource construct.Resource, fieldName string, value interface{}, zeroValueAllowed bool, graph *construct.ResourceGraph) error {
	field, setMapKey, err := parseFieldName(resource, fieldName, graph, true)
	if err != nil {
		return err
	}
	if setMapKey != nil && field.Type() == setMapKey.Value.Type() {
		field = reflect.New(field.Type()).Elem()
		setMapKey.Value = field
	}
	switch field.Kind() {
	case reflect.Slice, reflect.Array:
		if reflect.ValueOf(value).Kind() != reflect.Slice {
			return fmt.Errorf("config template is not the correct type for field %s and resource %s. expected it to be a list, but got %s", fieldName, resource.Id(), reflect.TypeOf(value))
		}
		err := configureField(value, field, graph, zeroValueAllowed)
		if err != nil {
			return err
		}
	case reflect.Pointer, reflect.Struct:
		// Since there can be pointers to primitive types and others, we will ensure that those still work
		if field.Kind() == reflect.Pointer && field.Elem().Kind() != reflect.Struct {
			if reflect.TypeOf(value) != field.Type() && reflect.TypeOf(value) == resourceIdType {
				return fmt.Errorf("config template is not the correct type for field %s and resource %s. expected it to be %s, but got %s", fieldName, resource.Id(), field.Type(), reflect.TypeOf(value))
			}
		} else if reflect.ValueOf(value).Kind() != reflect.Map && !field.Type().Implements(reflect.TypeOf((*construct.Resource)(nil)).Elem()) && field.Type() != reflect.TypeOf(construct.ResourceId{}) {
			return fmt.Errorf("config template is not the correct type for field %s and resource %s. expected it to be a map, but got %s", fieldName, resource.Id(), reflect.TypeOf(value))
		}
		err := configureField(value, field, graph, zeroValueAllowed)
		if err != nil {
			return err
		}
	default:
		if reflect.TypeOf(value) != field.Type() && reflect.TypeOf(value) == resourceIdType {
			return fmt.Errorf("config template is not the correct type for field %s and resource %s. expected it to be %s, but got %s", fieldName, resource.Id(), field.Type(), reflect.TypeOf(value))
		}
		err := configureField(value, field, graph, zeroValueAllowed)
		if err != nil {
			return err
		}
	}
	if setMapKey != nil {
		setMapKey.Map.SetMapIndex(setMapKey.Key, setMapKey.Value)
	}
	zap.S().Debugf("configured %s#%s to value '%v'", resource.Id(), fieldName, value)
	return nil
}

func configureField(val interface{}, field reflect.Value, dag *construct.ResourceGraph, zeroValueAllowed bool) error {
	if !reflect.ValueOf(val).IsValid() {
		return nil
	} else if reflect.ValueOf(val).IsZero() {
		return nil
	}

	if field.Kind() == reflect.Ptr && field.IsNil() {
		field.Set(reflect.New(reflect.TypeOf(field.Interface()).Elem()))
	}
	// We want to check if the field is a core Resource and if so we want to ensure that strings which represent ids
	// and resource ids are properly being cast to the correct type
	if field.Kind() == reflect.Ptr {
		if field.Type().Implements(reflect.TypeOf((*construct.Resource)(nil)).Elem()) && reflect.ValueOf(val).Type().Kind() == reflect.String {
			res := getFieldFromIdString(val.(string), dag)
			// if the return type is a resource id we need to get the correlating resource object
			if id, ok := res.(construct.ResourceId); ok {
				res = dag.GetResource(id)
			}
			if res == nil && !zeroValueAllowed {
				return fmt.Errorf("resource %s does not exist in the graph", val)
			} else if zeroValueAllowed && res == nil {
				return nil
			}
			field.Elem().Set(reflect.ValueOf(res).Elem())
			return nil
		} else if field.Type().Implements(reflect.TypeOf((*construct.Resource)(nil)).Elem()) && reflect.ValueOf(val).Type() == resourceIdType {
			id := val.(construct.ResourceId)
			res := getFieldFromIdString(id.String(), dag)
			// if the return type is a resource id we need to get the correlating resource object
			if id, ok := res.(construct.ResourceId); ok {
				res = dag.GetResource(id)
			}
			if res == nil && !zeroValueAllowed {
				return fmt.Errorf("resource %s does not exist in the graph", id)
			} else if zeroValueAllowed && res == nil {
				return nil
			}
			field.Elem().Set(reflect.ValueOf(res).Elem())
			return nil
		}
		field = field.Elem()
	}
	// see if we are getting a field from a resource ID # notation. If so we are going to assume the type is the same and set it and return
	if reflect.TypeOf(val).Kind() == reflect.String {
		fieldFromString := getFieldFromIdString(val.(string), dag)
		if fieldFromString != nil {
			fval := reflect.ValueOf(fieldFromString)
			if fval.Type().AssignableTo(field.Type()) {
				field.Set(reflect.ValueOf(fieldFromString))
				return nil
			}
		}
	}

	switch field.Kind() {
	case reflect.Slice, reflect.Array:
		arr := field
		// TODO: Add check to ensure we arent adding duplicate entries
		for i := 0; i < reflect.ValueOf(val).Len(); i++ {
			val := reflect.ValueOf(val).Index(i).Interface()
			if field.Type().Elem().Kind() == reflect.Struct {
				// create struct element from the map values passed in
				subField := reflect.New(field.Type().Elem()).Interface()
				err := configureField(val, reflect.ValueOf(subField), dag, zeroValueAllowed)
				if err != nil {
					return err
				}
				val = subField
			} else if field.Type().Elem().Kind() == reflect.Ptr {
				// create pointer element from the map values passed in
				subField := reflect.New(field.Type().Elem().Elem()).Interface()
				err := configureField(val, reflect.ValueOf(subField).Elem(), dag, zeroValueAllowed)
				if err != nil {
					return err
				}
				val = subField
			}
			// if val is a pointer we want to make sure that we transition it back to an element if the array is not a pointer array
			if reflect.ValueOf(val).Kind() == reflect.Ptr && field.Type().Elem().Kind() != reflect.Ptr {
				val = reflect.ValueOf(val).Elem().Interface()
			}
			// Check to see if this already exists in the array
			duplicate := false
			for i := 0; i < field.Len(); i++ {
				if reflect.DeepEqual(field.Index(i).Interface(), val) {
					duplicate = true
				}
			}
			if duplicate {
				continue
			}
			arr = reflect.Append(arr, reflect.ValueOf(val))

		}
		field.Set(arr)

	case reflect.Struct:
		// if the field represents an IntOrString, we need to parse the value instead of setting each field on the struct individually
		if _, ok := field.Interface().(intstr.IntOrString); ok {
			val = intstr.Parse(fmt.Sprintf("%v", val))
			field.Set(reflect.ValueOf(val))
			return nil
		}
		if field.Type() == reflect.TypeOf(construct.ResourceId{}) && reflect.ValueOf(val).Type().Kind() == reflect.String {
			id := construct.ResourceId{}
			err := id.UnmarshalText([]byte(val.(string)))
			if err != nil {
				return err
			}
			field.Set(reflect.ValueOf(id))
			return nil
		}
		if field.Kind() == reflect.Ptr && field.IsNil() {
			field.Set(reflect.New(reflect.TypeOf(field.Interface()).Elem()))
		}
		if field.Kind() == reflect.Ptr {
			field = field.Elem()
		}
		for _, key := range reflect.ValueOf(val).MapKeys() {
			for i := 0; i < field.NumField(); i++ {
				if field.Type().Field(i).Name == key.String() {
					err := configureField(reflect.ValueOf(val).MapIndex(key).Interface(), field.Field(i), dag, zeroValueAllowed)
					if err != nil {
						return err
					}
				}
			}
		}
	case reflect.Map:
		// if the field is a map[string]string, we need to unbox the map[string]interface{} into a map[string]string
		requiresMapStringString := false
		if _, ok := field.Interface().(map[string]string); ok {
			requiresMapStringString = true
		}
		if unboxed, ok := val.(map[string]interface{}); requiresMapStringString && ok {
			mapStringString := make(map[string]string)
			for k, v := range unboxed {
				mapStringString[k] = fmt.Sprintf("%v", v)
			}
			for _, key := range reflect.ValueOf(mapStringString).MapKeys() {
				field.SetMapIndex(reflect.ValueOf(key).Convert(field.Type().Key()), reflect.ValueOf(mapStringString).MapIndex(key).Convert(field.Type().Elem()))
			}
		} else {
			for _, key := range reflect.ValueOf(val).MapKeys() {
				field.SetMapIndex(key, reflect.ValueOf(val).MapIndex(key))
			}
		}
	case reflect.TypeOf(intstr.IntOrString{}).Kind():
		val = intstr.Parse(fmt.Sprintf("%s", val))
		field.Set(reflect.ValueOf(val))
	case reflect.Int32:
		field.Set(reflect.ValueOf(int32(val.(int))))
	case reflect.String:
		if !reflect.TypeOf(val).AssignableTo(field.Type()) {
			// convert string to the correct enum type if the string value is not assignable to the field
			v := reflect.New(field.Type()).Elem()
			v.SetString(val.(string))
			field.Set(v)
		} else {
			// just set the string
			field.Set(reflect.ValueOf(val))
		}
	default:
		if field.Kind() == reflect.String && reflect.TypeOf(val).Kind() != reflect.String && reflect.TypeOf(val).Elem() == resourceIdType {
			id := val.(*construct.ResourceId)
			strVal := getFieldFromIdString(id.String(), dag)
			if strVal != nil {
				field.Set(reflect.ValueOf(strVal))
				return nil
			}
		}
		field.Set(reflect.ValueOf(val))
	}
	return nil

}

func getIdAndFields(id construct.ResourceId) (construct.ResourceId, string) {
	arr := strings.Split(id.String(), "#")
	resId := &construct.ResourceId{}
	err := resId.UnmarshalText([]byte(arr[0]))
	if err != nil {
		return construct.ResourceId{}, ""
	}
	if len(arr) == 1 {
		return *resId, ""
	}
	return *resId, arr[1]
}

func getFieldFromIdString(id string, dag *construct.ResourceGraph) any {
	arr := strings.Split(id, "#")
	var resId construct.ResourceId
	err := resId.UnmarshalText([]byte(arr[0]))
	if err != nil {
		return nil
	}
	res := dag.GetResource(resId)
	if res == nil {
		return nil
	}
	if len(arr) == 1 {
		return resId
	}

	field, _, err := parseFieldName(res, arr[1], dag, true)
	if err != nil {
		return nil
	}
	return field.Interface()
}

func GetFieldByName(s reflect.Value, fieldName string) (reflect.StructField, error) {
	for s.Kind() == reflect.Ptr {
		s = s.Elem()
	}
	t := s.Type()
	field, ok := t.FieldByName(fieldName)
	if ok {
		return field, nil
	}

	// Try to find the field by its json or yaml tag (especially to handle case [upper/lower] [Pascal/snake])
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if fieldName == strings.ToLower(f.Name) {
			// When YAML marshalling fields that don't have a tag, they're just lower cased
			// so this condition should catch those.
			return f, nil
		}
		tagName, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		if fieldName == tagName {
			return f, nil
		}
		tagName, _, _ = strings.Cut(f.Tag.Get("yaml"), ",")
		if fieldName == tagName {
			return f, nil
		}
	}

	return reflect.StructField{}, fmt.Errorf("unable to find field %s on resource %s", fieldName, s)
}

// ParseFieldName parses a field name and returns the value of the field
// Example: "spec.template.spec.containers[0].image" will return the value of the image field of the first container in the template
//
// if you pass in configure as false, then the function will not create any new fields if they are nil and rather will return an empty reflect value
func parseFieldName(resource construct.Resource, fieldName string, dag *construct.ResourceGraph, configure bool) (reflect.Value, *SetMapKey, error) {
	fields := strings.Split(fieldName, ".")
	var field reflect.Value
	var setMapKey *SetMapKey
	for i := 0; i < len(fields); i++ {
		splitField := strings.Split(fields[i], "[")
		currFieldName := splitField[0]
		var key string
		if len(splitField) > 1 {
			key = strings.TrimSuffix(splitField[1], "]")
			key = strings.TrimPrefix(key, "\"")
			key = strings.TrimSuffix(key, "\"")
		}
		parent := field
		if i == 0 {
			parent = reflect.ValueOf(resource).Elem()
		}
		if parent.Kind() == reflect.Ptr {
			parent = field.Elem()
		}
		field = parent.FieldByName(currFieldName)
		if !field.IsValid() {
			fieldType, err := GetFieldByName(parent, currFieldName)
			if err != nil {
				return reflect.Value{}, nil, err
			}
			field = parent.FieldByIndex(fieldType.Index)
		}
		if field.IsZero() && field.Kind() == reflect.Ptr {
			if !configure {
				return reflect.Value{}, nil, nil
			}
			newField := reflect.New(field.Type().Elem())
			field.Set(newField)
			field = newField
		}
		if key != "" {
			if field.Kind() == reflect.Map {
				// Right now we only support string keys on maps, so error if we see a mismatch
				if field.Type().Key().Kind() != reflect.String {
					return reflect.Value{}, nil, fmt.Errorf("unable to find field %s on resource %s, field is not a map[string]", strings.Join(fields[:i+1], "."), resource.Id())
				}
				// create the map if it is currently nil
				if field.IsNil() {
					field.Set(reflect.MakeMap(field.Type()))
				}

				resId := &construct.ResourceId{}
				err := resId.UnmarshalText([]byte(key))
				if err == nil {
					// if the key is a resource id, then we need to get the field from the resource
					field := getFieldFromIdString(resId.String(), dag)
					if field != nil {
						key = fmt.Sprintf("%v", field)
					}
				}
				// create a copy of the value and clone the existing one. We do this because map values are not addressable
				newField := reflect.New(field.Type().Elem()).Elem()
				if field.MapIndex(reflect.ValueOf(key)).IsValid() {
					newField.Set(field.MapIndex(reflect.ValueOf(key)))
				}
				// set the map key to the new value so that we can set the mapkey after all configuration has been done
				setMapKey = &SetMapKey{Map: field, Key: reflect.ValueOf(key), Value: newField}
				field = newField
			} else if field.Kind() == reflect.Slice || field.Kind() == reflect.Array {
				index, err := strconv.Atoi(key)
				if err != nil {
					return reflect.Value{}, nil, fmt.Errorf("unable to find field %s on resource %s, could not convert index to int", strings.Join(fields[:i+1], "."), resource.Id())

				}
				if index >= field.Len() {
					return reflect.Value{}, nil, fmt.Errorf("unable to find field %s on resource %s, length of array is less than index", strings.Join(fields[:i+1], "."), resource.Id())
				}
				field = field.Index(index)
			} else {
				return reflect.Value{}, nil, fmt.Errorf("unable to find field %s on resource %s, field type does not support key or index", strings.Join(fields[:i+1], "."), resource.Id())
			}
		}
	}
	return field, setMapKey, nil
}
