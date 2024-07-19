package reflectutil

import (
	"errors"
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
	if v.IsValid() {
		return GetConcreteElement(v).Interface()
	}
	return nil
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

// GetField returns the [reflect.Value] of a field in a struct or map.
func GetField(v reflect.Value, fieldExpr string) (reflect.Value, error) {
	if v.Kind() == reflect.Invalid {
		return reflect.Value{}, fmt.Errorf("value is nil")
	}

	fields := SplitPath(fieldExpr)
	for _, field := range fields {
		if strings.Contains(field, "[") != strings.Contains(field, "]") {
			return reflect.Value{}, errors.New("invalid path: unclosed brackets ")
		}

		v = GetConcreteElement(v)

		// Handle array/slice indices
		if strings.Contains(field, "[") && !strings.Contains(field, ".") {
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
			field = strings.TrimSuffix(strings.TrimLeft(field, ".["), "]")
			switch v.Kind() {
			case reflect.Map:
				if v.Type().Key().Kind() != reflect.String {
					return reflect.Value{}, fmt.Errorf("unsupported map key type: %s: key type must be 'String'", v.Type().Key())
				}
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

func TracePath(v reflect.Value, fieldExpr string) ([]reflect.Value, error) {
	if !v.IsValid() {
		return nil, fmt.Errorf("value is invalid")
	}

	trace := []reflect.Value{v}

	if fieldExpr == "" {
		return trace, nil
	}

	fields := strings.Split(fieldExpr, ".")

	for _, field := range fields {
		last := trace[len(trace)-1]
		next, err := GetField(last, field)
		if err != nil {
			return nil, err
		}
		trace = append(trace, next)
	}

	return trace, nil
}

// FirstOfType returns the first value in the slice that matches the specified type.
// If no matching value is found, it returns the zero value of the type and false.
func FirstOfType[T any](values []reflect.Value) (T, bool) {
	var zero T
	for _, v := range values {
		if v.CanInterface() {
			if val, ok := v.Interface().(T); ok {
				return val, true
			}
		}
	}
	return zero, false
}

func LastOfType[T any](values []reflect.Value) (T, bool) {
	// Create a new slice with reversed order
	reversed := make([]reflect.Value, len(values))
	for i, v := range values {
		reversed[len(values)-1-i] = v
	}

	// Use FirstOfType on the reversed slice
	return FirstOfType[T](reversed)
}

// IsAnyOf returns true if the [reflect.Value] is any of the specified types.
func IsAnyOf(v reflect.Value, types ...reflect.Kind) bool {
	for _, t := range types {
		if v.Kind() == t {
			return true
		}
	}
	return false
}

// SplitPath splits a path string into parts separated by '.' and '[', ']'.
// It is used to split a path string into parts that can be used to access fields in a slice, array, struct, or map.
// Bracketed components are treated as a single part, including the brackets.
func SplitPath(path string) []string {
	var parts []string
	bracket := 0
	lastPartIdx := 0
	for i := 0; i < len(path); i++ {
		switch path[i] {
		case '.':
			if bracket == 0 {
				if i > lastPartIdx {
					parts = append(parts, path[lastPartIdx:i])
				}
				lastPartIdx = i
			}

		case '[':
			if bracket == 0 {
				if i > lastPartIdx {
					parts = append(parts, path[lastPartIdx:i])
				}
				lastPartIdx = i
			}
			bracket++

		case ']':
			bracket--
			if bracket == 0 {
				parts = append(parts, path[lastPartIdx:i+1])
				lastPartIdx = i + 1
			}
		}
		if i == len(path)-1 && lastPartIdx <= i {
			parts = append(parts, path[lastPartIdx:])
		}
	}
	return parts
}
