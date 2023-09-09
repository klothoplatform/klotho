package engine

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/klothoplatform/klotho/pkg/construct"
)

// SetMapKey is a struct that represents a key in a map
// Because values of maps are not addressable, we need to store the map and the key separately
// then after we configure the sub field, we are able to go back and store that value in the map
type SetMapKey struct {
	Map   reflect.Value
	Key   reflect.Value
	Value reflect.Value
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
	resId := &construct.ResourceId{}
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

	field, _, err := parseFieldName(res, arr[1], dag, true)
	if err != nil {
		return nil
	}
	return field.Interface()
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
					return reflect.Value{}, nil, fmt.Errorf("unable to find field %s on resource %s, field is not a map[string]", fields[i], resource.Id())
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
