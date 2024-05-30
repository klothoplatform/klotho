package constructs

import (
	"fmt"
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	"reflect"
)

type (
	// ConstructMarshaller is a struct that marshals a Construct into a list of constraints
	ConstructMarshaller struct {
		Construct *Construct
		Context   *ConstructContext
	}
)

func (m *ConstructMarshaller) Marshal() ([]constraints.Constraint, error) {
	//TODO: consider look into capturing multiple errors instead of returning the first one
	var cs []constraints.Constraint
	for _, r := range m.Construct.Resources {
		resourceConstraints, err := m.marshalResource(r)
		if err != nil {
			err = fmt.Errorf("could not marshall resource: %w", err)
		}
		cs = append(cs, resourceConstraints...)
	}
	for _, e := range m.Construct.Edges {
		edgeConstraints, err := m.marshalEdge(e)
		if err != nil {
			err = fmt.Errorf("could not marshall edge: %w", err)
		}

		cs = append(cs, edgeConstraints...)
	}

	return cs, nil
}

func (m *ConstructMarshaller) marshalResource(r *Resource) ([]constraints.Constraint, error) {
	var cs []constraints.Constraint
	cs = append(cs, &constraints.ApplicationConstraint{
		Operator: "add",
		Node:     r.Id,
	})
	// TODO: implement more granular constraints
	for k, v := range r.Properties {

		v, err := marshallRefs(v, m.Context)
		if err != nil {
			return nil, fmt.Errorf("could not marshall resource properties: %w", err)
		}
		cs = append(cs, &constraints.ResourceConstraint{
			Operator: "equals",
			Target:   r.Id,
			Property: k,
			Value:    v,
		})
	}

	return cs, nil
}

func (m *ConstructMarshaller) marshalEdge(e *Edge) ([]constraints.Constraint, error) {

	var from construct.ResourceId
	ref, err := m.Context.serializeRef(e.From)
	if err != nil {
		return nil, fmt.Errorf("could not serialize from resource id: %w", err)
	}
	err = from.Parse(ref.(string))
	if err != nil {
		return nil, fmt.Errorf("could not parse from resource id: %w", err)
	}

	var to construct.ResourceId
	ref, err = m.Context.serializeRef(e.To)
	if err != nil {
		return nil, fmt.Errorf("could not serialize to resource id: %w", err)
	}
	err = to.Parse(ref.(string))
	if err != nil {
		return nil, fmt.Errorf("could not parse to resource id: %w", err)
	}
	v, err := marshallRefs(e.Data, m.Context)
	if err != nil {
		return nil, fmt.Errorf("could not marshall resource properties: %w", err)
	}

	return []constraints.Constraint{&constraints.EdgeConstraint{
		Operator: "must_contain",
		Target: constraints.Edge{
			Source: from,
			Target: to,
		},
		Data: v.(map[string]any),
	}}, nil
}

// MarshallRefs replaces all ResourceRef instances in an input (rawVal) with the serialized values using the context's serializeRef method
func marshallRefs(rawVal any, ctx *ConstructContext) (any, error) {
	if ref, ok := rawVal.(ResourceRef); ok {
		return ctx.serializeRef(ref)
	}

	ref := reflect.ValueOf(rawVal)
	if ref.Kind() == reflect.Ptr {
		ref = ref.Elem()
	}

	var err error
	switch ref.Kind() {
	case reflect.Struct:
		for i := 0; i < ref.NumField(); i++ {
			field := ref.Field(i)
			if field.Kind() == reflect.Ptr {
				field = field.Elem()
			}
			if field.Kind() == reflect.Struct {
				_, err = marshallRefs(field.Interface(), ctx)
			}
			if newField, ok := field.Interface().(ResourceRef); ok {
				var serializedRef any
				serializedRef, err = ctx.serializeRef(newField)
				if err != nil {
					return nil, err
				}
				ref.Field(i).Set(reflect.ValueOf(serializedRef))
			}
		}
	case reflect.Map:
		for _, key := range ref.MapKeys() {
			field := ref.MapIndex(key)
			if field.Kind() == reflect.Ptr {
				field = field.Elem()
			}
			if field.Kind() == reflect.Struct {
				_, err = marshallRefs(field.Interface(), ctx)
			}
			if newField, ok := field.Interface().(ResourceRef); ok {
				var serializedRef any
				serializedRef, err = ctx.serializeRef(newField)
				if err != nil {
					return nil, err
				}
				ref.SetMapIndex(key, reflect.ValueOf(serializedRef))
			}
		}
	case reflect.Slice | reflect.Array:
		for i := 0; i < ref.Len(); i++ {
			field := ref.Index(i)
			if field.Kind() == reflect.Ptr {
				field = field.Elem()
			}
			if field.Kind() == reflect.Struct {
				_, err = marshallRefs(field.Interface(), ctx)
			}
			if newField, ok := field.Interface().(ResourceRef); ok {
				var serializedRef any
				serializedRef, err = ctx.serializeRef(newField)
				if err != nil {
					return nil, err
				}
				ref.Index(i).Set(reflect.ValueOf(serializedRef))
			}
		}
	case reflect.Interface:
		if ref.Elem().Kind() == reflect.Struct {
			_, err = marshallRefs(ref.Elem().Interface(), ctx)
		}
	default:
		if newField, ok := ref.Interface().(ResourceRef); ok {
			var serializedRef any
			serializedRef, err = ctx.serializeRef(newField)
			if err != nil {
				return nil, err
			}
			ref.Set(reflect.ValueOf(serializedRef))
		}
	}

	if err != nil {
		return nil, err
	}

	return ref.Interface(), nil
}

type ConstraintValueProvider interface {
	MarshallValue() any
}

// MarshallValue replaces a struct in place with the output of its MarshallValue method
func MarshallValue(value any) any {
	ref := reflect.ValueOf(value)
	if ref.Kind() == reflect.Ptr {
		ref = ref.Elem()
	}
	switch ref.Kind() {
	case reflect.Struct:
		for i := 0; i < ref.NumField(); i++ {
			field := ref.Field(i)
			if field.Kind() == reflect.Ptr {
				field = field.Elem()
			}
			if field.Kind() == reflect.Struct {
				MarshallValue(field.Interface())
			}
			if newField, ok := field.Interface().(ConstraintValueProvider); ok {
				ref.Field(i).Set(reflect.ValueOf(newField.MarshallValue()))
			}
		}
	case reflect.Map:
		for _, key := range ref.MapKeys() {
			field := ref.MapIndex(key)
			if field.Kind() == reflect.Ptr {
				field = field.Elem()
			}
			if field.Kind() == reflect.Struct {
				MarshallValue(field.Interface())
			}
			if newField, ok := field.Interface().(ConstraintValueProvider); ok {
				ref.SetMapIndex(key, reflect.ValueOf(newField.MarshallValue()))
			}
		}
	case reflect.Slice | reflect.Array:
		for i := 0; i < ref.Len(); i++ {
			field := ref.Index(i)
			if field.Kind() == reflect.Ptr {
				field = field.Elem()
			}
			if field.Kind() == reflect.Struct {
				MarshallValue(field.Interface())
			}
			if newField, ok := field.Interface().(ConstraintValueProvider); ok {
				ref.Index(i).Set(reflect.ValueOf(newField.MarshallValue()))
			}
		}
	case reflect.Interface:
		if ref.Elem().Kind() == reflect.Struct {
			MarshallValue(ref.Elem().Interface())
		}
	default:
		if newField, ok := ref.Interface().(ConstraintValueProvider); ok {
			ref.Set(reflect.ValueOf(newField.MarshallValue()))
		}
	}
	return ref.Interface()
}

func (e *Edge) toConstraints() []constraints.Constraint {
	//TODO: handle refs in edges
	cs := make([]constraints.Constraint, 0)
	cs = append(cs, &constraints.EdgeConstraint{
		Operator: "must_contain",
		//Target: constraints.Edge{
		//	Source: e.From,
		//	Target: e.To,
		//},
	})
	return cs
}

//func transformFields[T any](input any, recursive bool, transformer func(field any) (any, error)) T {
//	ref := reflect.ValueOf(input)
//	if ref.Kind() == reflect.Ptr {
//		ref = ref.Elem()
//	}
//	switch ref.Kind() {
//	case reflect.Struct:
//		for i := 0; i < ref.NumField(); i++ {
//			field := ref.Field(i)
//			if field.Kind() == reflect.Ptr {
//				field = field.Elem()
//			}
//			if field.Kind() == reflect.Struct && recursive {
//				transformFields[any](field, recursive, transformer)
//			}
//			if newField, err := transformer(field.Interface()); err == nil {
//				ref.Field(i).Set(reflect.ValueOf(newField))
//			}
//		}
//	case reflect.Map:
//		for _, key := range ref.MapKeys() {
//			field := ref.MapIndex(key)
//			if field.Kind() == reflect.Ptr {
//				field = field.Elem()
//			}
//			if field.Kind() == reflect.Struct && recursive {
//				transformFields[any](field, recursive, transformer)
//			}
//			if newField, err := transformer(field.Interface()); err == nil {
//				ref.SetMapIndex(key, reflect.ValueOf(newField))
//			}
//		}
//	case reflect.Slice | reflect.Array:
//		for i := 0; i < ref.Len(); i++ {
//			field := ref.Index(i)
//			if field.Kind() == reflect.Ptr {
//				field = field.Elem()
//			}
//			if field.Kind() == reflect.Struct && recursive {
//				transformFields[any](field, recursive, transformer)
//			}
//			if newField, err := transformer(field.Interface()); err == nil {
//				ref.Index(i).Set(reflect.ValueOf(newField))
//			}
//		}
//	case reflect.Interface:
//		if ref.Elem().Kind() == reflect.Struct && recursive {
//			transformFields[any](ref.Elem(), recursive, transformer)
//		}
//	default:
//		if ref.CanSet() {
//			if newField, err := transformer(ref.Interface()); err == nil {
//				ref.Set(reflect.ValueOf(newField))
//			}
//		} else {
//			zap.S().Warnf("Field %v is not settable", ref)
//		}
//	}
//	return ref.Interface().(T)
//}
