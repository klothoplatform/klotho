package engine

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/klothoplatform/klotho/pkg/core"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// SetMapKey is a struct that represents a key in a map
// Because values of maps are not addressable, we need to store the map and the key separately
// then after we configure the sub field, we are able to go back and store that value in the map
type SetMapKey struct {
	Map   reflect.Value
	Key   reflect.Value
	Value reflect.Value
}

// ConfigureField is a function that takes a resource, a field name, and a value and sets the field on the resource to the value
// It also takes a graph so that it can resolve references
// It returns an error if the field cannot be set
func ConfigureField(resource core.Resource, fieldName string, value interface{}, zeroValueAllowed bool, graph *core.ResourceGraph) error {
	field, setMapKey, err := parseFieldName(resource, fieldName, graph)
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
		if reflect.ValueOf(value).Kind() != reflect.Map && !field.Type().Implements(reflect.TypeOf((*core.Resource)(nil)).Elem()) && field.Type() != reflect.TypeOf(core.ResourceId{}) {
			return fmt.Errorf("config template is not the correct type for field %s and resource %s. expected it to be a map, but got %s", fieldName, resource.Id(), reflect.TypeOf(value))
		}
		err := configureField(value, field, graph, zeroValueAllowed)
		if err != nil {
			return err
		}
	default:
		if reflect.TypeOf(value) != field.Type() && reflect.TypeOf(value).String() == "core.ResourceId" {
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
	return nil
}

func configureField(val interface{}, field reflect.Value, dag *core.ResourceGraph, zeroValueAllowed bool) error {
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
		if field.Type().Implements(reflect.TypeOf((*core.Resource)(nil)).Elem()) && reflect.ValueOf(val).Type().Kind() == reflect.String {
			res := getFieldFromIdString(val.(string), dag)
			// if the return type is a resource id we need to get the correlating resource object
			if id, ok := res.(core.ResourceId); ok {
				res = dag.GetResource(id)
			}
			if res == nil && !zeroValueAllowed {
				return fmt.Errorf("resource %s does not exist in the graph", val)
			} else if zeroValueAllowed && res == nil {
				return nil
			}
			field.Elem().Set(reflect.ValueOf(res).Elem())
			return nil
		} else if field.Type().Implements(reflect.TypeOf((*core.Resource)(nil)).Elem()) && reflect.ValueOf(val).Type().String() == "core.ResourceId" {
			id := val.(core.ResourceId)
			res := getFieldFromIdString(id.String(), dag)
			// if the return type is a resource id we need to get the correlating resource object
			if id, ok := res.(core.ResourceId); ok {
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
			field.Set(reflect.ValueOf(fieldFromString))
			if reflect.TypeOf(fieldFromString) != field.Type() {
				return fmt.Errorf("the type represented by %s not the correct type for field %s. expected it to be %s, but got %s", val, field, field.Type(), reflect.TypeOf(fieldFromString))
			}
			return nil
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
		if field.Type() == reflect.TypeOf(core.ResourceId{}) && reflect.ValueOf(val).Type().Kind() == reflect.String {
			id := core.ResourceId{}
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
		if field.Kind() == reflect.String && reflect.TypeOf(val).Kind() != reflect.String && reflect.TypeOf(val).Elem().String() == "core.ResourceId" {
			id := val.(*core.ResourceId)
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

func getIdAndFields(id core.ResourceId) (core.ResourceId, string) {
	arr := strings.Split(id.String(), "#")
	resId := &core.ResourceId{}
	err := resId.UnmarshalText([]byte(arr[0]))
	if err != nil {
		return core.ResourceId{}, ""
	}
	if len(arr) == 1 {
		return *resId, ""
	}
	return *resId, arr[1]
}

func getFieldFromIdString(id string, dag *core.ResourceGraph) any {
	arr := strings.Split(id, "#")
	resId := &core.ResourceId{}
	err := resId.UnmarshalText([]byte(arr[0]))
	if err != nil {
		return nil
	}
	if len(arr) == 1 {
		return *resId
	}
	res := dag.GetResource(*resId)
	if res == nil {
		return nil
	}

	field, _, err := parseFieldName(res, arr[1], dag)
	if err != nil {
		return nil
	}
	return field.Interface()
}

// ParseFieldName parses a field name and returns the value of the field
// Example: "spec.template.spec.containers[0].image" will return the value of the image field of the first container in the template
func parseFieldName(resource core.Resource, fieldName string, dag *core.ResourceGraph) (reflect.Value, *SetMapKey, error) {
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
		if i == 0 {
			field = reflect.ValueOf(resource).Elem().FieldByName(currFieldName)
		} else {
			if field.Kind() == reflect.Ptr {
				field = field.Elem().FieldByName(currFieldName)
			} else {
				field = field.FieldByName(currFieldName)
			}
		}
		if !field.IsValid() {
			return reflect.Value{}, nil, fmt.Errorf("unable to find field %s on resource %s, field is not valid", fields[i], resource.Id())
		} else if field.IsZero() && field.Kind() == reflect.Ptr {
			newField := reflect.New(field.Type().Elem())
			field.Set(newField)
			field = newField
		}
		if key != "" {
			if field.Kind() == reflect.Map {
				// Right now we only support string keys on maps, so error if we see a mismatch
				if field.Type().Key().Kind() != reflect.String {
					return reflect.Value{}, nil, fmt.Errorf("unable to find field %s on resource %s, field is not a map[string]", fields[i], resource.Id())
				}
				// create the map if it is currently nil
				if field.IsNil() {
					field.Set(reflect.MakeMap(field.Type()))
				}

				resId := &core.ResourceId{}
				err := resId.UnmarshalText([]byte(key))
				if err == nil {
					// if the key is a resource id, then we need to get the field from the resource
					field := getFieldFromIdString(resId.String(), dag)
					if field == nil {
						return reflect.Value{}, nil, fmt.Errorf("unable to find field %s on resource %s when getting field from id string", key, resId.String())
					}
					key = fmt.Sprintf("%v", field)
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
					return reflect.Value{}, nil, fmt.Errorf("unable to find field %s on resource %s, could not convert index to int", fields[i], resource.Id())

				}
				if index >= field.Len() {
					return reflect.Value{}, nil, fmt.Errorf("unable to find field %s on resource %s, length of array is less than index", fields[i], resource.Id())
				}
				field = field.Index(index)
			} else {
				return reflect.Value{}, nil, fmt.Errorf("unable to find field %s on resource %s, field type does not support key or index", fields[i], resource.Id())
			}
		}
	}
	return field, setMapKey, nil
}
