package reflectutil

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

/*
GetConcreteValue returns the concrete value of a reflect.Value.

This function is used to get the concrete value of a reflect.Value even if it is a pointer or interface.
Concrete values are values that are not pointers or interfaces (including maps, slices, structs, etc.).
*/
func GetConcreteValue(v reflect.Value) any {
	return GetConcreteElement(v).Interface()
}

// IsNotConcrete returns true if the reflect.Value is a pointer or interface.
func IsNotConcrete(v reflect.Value) bool {
	return v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface
}

// GetConcreteElement returns the concrete reflect.Value of a reflect.Value
// when it is a pointer or interface or the same reflect.Value when it is already concrete.
func GetConcreteElement(v reflect.Value) reflect.Value {
	for IsNotConcrete(v) {
		v = v.Elem()
	}
	return v
}

// GetField returns the reflect.Value of a field in a struct or map.
func GetField(v reflect.Value, fieldExpr string) (reflect.Value, error) {
	if v.Kind() == reflect.Invalid {
		return reflect.Value{}, fmt.Errorf("value is nil")
	}

	fields := strings.Split(fieldExpr, ".")
	for _, field := range fields {
		v = GetConcreteElement(v)

		// Handle array/slice indices
		if strings.Contains(field, "[") {
			fieldName := field[:strings.Index(field, "[")]
			indexStr := field[strings.Index(field, "[")+1 : strings.Index(field, "]")]
			index, err := strconv.Atoi(indexStr)
			if err != nil {
				return reflect.Value{}, err
			}

			if fieldName != "" {
				switch v.Kind() {
				case reflect.Map:
					v = GetConcreteElement(v.MapIndex(reflect.ValueOf(fieldName)))
				case reflect.Struct:
					v = GetConcreteElement(v.FieldByName(fieldName))
				default:
					return reflect.Value{}, fmt.Errorf("field is not a struct or map: %s", fieldName)
				}

				if !v.IsValid() {
					return reflect.Value{}, fmt.Errorf("invalid field name: %s", fieldName)
				}
			}

			switch v.Kind() {
			case reflect.Slice, reflect.Array:
				if index >= v.Len() {
					return reflect.Value{}, fmt.Errorf("index out of range: %d", index)
				}
				v = v.Index(index)
			default:
				return reflect.Value{}, fmt.Errorf("field is not a slice or array: %s", fieldName)
			}
		} else {
			switch v.Kind() {
			case reflect.Map:
				v = v.MapIndex(reflect.ValueOf(field))
				if !v.IsValid() {
					return reflect.Value{}, fmt.Errorf("invalid map key: %s", field)
				}
			case reflect.Struct:
				v = v.FieldByName(field)
				if !v.IsValid() {
					return reflect.Value{}, fmt.Errorf("invalid field name: %s", field)
				}
			case reflect.Slice, reflect.Array:
				index, err := strconv.Atoi(field)
				if err != nil {
					return reflect.Value{}, fmt.Errorf("invalid slice or array index: %s", field)
				}
				if index >= v.Len() {
					return reflect.Value{}, fmt.Errorf("index out of range: %d", index)
				}
				v = v.Index(index)
			default:
				return reflect.Value{}, fmt.Errorf("unsupported type for field: %s", field)
			}
		}
	}

	return v, nil
}

func GetTypedField[T any](v reflect.Value, fieldExpr string) (T, bool) {
	var zero T
	fieldValue, err := GetField(v, fieldExpr)
	if err != nil {
		return zero, false
	}

	return GetTypedValue[T](fieldValue)
}

func GetTypedValue[T any](v any) (T, bool) {
	var typedValue T
	var ok bool

	var tKind reflect.Kind
	var rVal reflect.Value
	if rVal, ok = v.(reflect.Value); !ok {
		rVal = reflect.ValueOf(v)
	}
	tKind = rVal.Kind()

	if tKind != reflect.Pointer && tKind != reflect.Interface {
		typedValue, ok = GetConcreteValue(rVal).(T)
	} else {
		typedValue, ok = rVal.Interface().(T)
	}

	return typedValue, ok
}
