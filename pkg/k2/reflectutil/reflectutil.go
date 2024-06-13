package reflectutil

import "reflect"

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
