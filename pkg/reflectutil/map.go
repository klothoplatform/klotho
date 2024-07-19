package reflectutil

import (
	"fmt"
	"reflect"
)

func MapContainsKey(m any, key interface{}) (bool, error) {
	var mapValue reflect.Value
	if mValue, ok := m.(reflect.Value); ok {
		mapValue = mValue
	} else {
		mapValue = reflect.ValueOf(m)
	}
	if mapValue.Kind() != reflect.Map {
		return false, fmt.Errorf("value is not a map")
	}

	keyValue := reflect.ValueOf(key)
	if !keyValue.IsValid() {
		return false, fmt.Errorf("invalid key")
	}

	return mapValue.MapIndex(keyValue).IsValid(), nil
}
