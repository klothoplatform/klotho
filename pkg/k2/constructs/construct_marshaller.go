package constructs

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/klothoplatform/klotho/pkg/k2/reflectutil"
)

type (
	// ConstructMarshaller is a struct that marshals a Construct into a list of constraints
	ConstructMarshaller struct {
		ConstructEvaluator *ConstructEvaluator
	}
)

// Marshal marshals a Construct into a list of constraints
func (m *ConstructMarshaller) Marshal(constructURN model.URN) (constraints.ConstraintList, error) {
	var cs constraints.ConstraintList
	c, ok := m.ConstructEvaluator.Constructs.Get(constructURN)
	if !ok {
		return nil, fmt.Errorf("could not find construct %s", constructURN)
	}

	for _, r := range c.Resources {
		resourceConstraints, err := m.marshalResource(c, r)
		if err != nil {
			err = fmt.Errorf("could not marshall resource: %w", err)
			return nil, err
		}
		cs = append(cs, resourceConstraints...)
	}
	for _, e := range c.Edges {
		edgeConstraints, err := m.marshalEdge(c, e)
		if err != nil {
			return nil, fmt.Errorf("could not marshall edge: %w", err)
		}

		cs = append(cs, edgeConstraints...)
	}

	for _, o := range c.OutputDeclarations {
		outputConstraints, err := m.marshalOutput(o)
		if err != nil {
			return nil, fmt.Errorf("could not marshall output: %w", err)
		}
		cs = append(cs, outputConstraints...)
	}

	sort.SliceStable(cs, cs.NaturalSort)

	return cs, nil
}

func (m *ConstructMarshaller) marshalResource(o InfraOwner, r *Resource) (constraints.ConstraintList, error) {
	var cs constraints.ConstraintList
	cs = append(cs, &constraints.ApplicationConstraint{
		Operator: "must_exist",
		Node:     r.Id,
	})
	for k, v := range r.Properties {

		v, err := m.marshalRefs(o, v)
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

// marshalEdge marshals an Edge into a list of constraints
func (m *ConstructMarshaller) marshalEdge(o InfraOwner, e *Edge) (constraints.ConstraintList, error) {

	var from construct.ResourceId
	ref, err := m.ConstructEvaluator.serializeRef(o, e.From)
	if err != nil {
		return nil, fmt.Errorf("could not serialize from resource id: %w", err)
	}
	if idRef, ok := ref.(construct.ResourceId); ok {
		from = idRef
	} else {
		err = from.Parse(ref.(string))
	}
	if err != nil {
		return nil, fmt.Errorf("could not parse from resource id: %w", err)
	}

	var to construct.ResourceId
	ref, err = m.ConstructEvaluator.serializeRef(o, e.To)
	if err != nil {
		return nil, fmt.Errorf("could not serialize to resource id: %w", err)
	}
	if idRef, ok := ref.(construct.ResourceId); ok {
		to = idRef
	} else {
		err = to.Parse(ref.(string))
	}
	if err != nil {
		return nil, fmt.Errorf("could not parse to resource id: %w", err)
	}
	v, err := m.marshalRefs(o, e.Data)
	if err != nil {
		return nil, fmt.Errorf("could not marshall resource properties: %w", err)
	}

	return constraints.ConstraintList{&constraints.EdgeConstraint{
		Operator: "must_exist",
		Target: constraints.Edge{
			Source: from,
			Target: to,
		},
		Data: v.(construct.EdgeData),
	}}, nil
}

// marshalOutput marshals an OutputDeclaration into a list of constraints
func (m *ConstructMarshaller) marshalOutput(o OutputDeclaration) (constraints.ConstraintList, error) {
	var cs constraints.ConstraintList

	c := &constraints.OutputConstraint{
		Operator: "must_exist",
		Name:     o.Name,
	}
	if o.Ref != (construct.PropertyRef{}) {
		c.Ref = o.Ref
	} else {
		c.Value = o.Value
	}

	cs = append(cs, c)
	return cs, nil
}

// marshalRefs replaces all ResourceRef instances in an input (rawVal) with the serialized values using the context's serializeRef method
func (m *ConstructMarshaller) marshalRefs(o InfraOwner, rawVal any) (any, error) {

	ref := reflectutil.GetConcreteElement(reflect.ValueOf(rawVal))

	switch val := rawVal.(type) {
	case ResourceRef:
		return m.ConstructEvaluator.serializeRef(o, val)
	case construct.ResourceId, construct.PropertyRef:
		return val, nil
	}

	var err error
	switch ref.Kind() {
	case reflect.Struct:
		for i := 0; i < ref.NumField(); i++ {
			field := reflectutil.GetConcreteElement(ref.Field(i))
			if field.Kind() == reflect.Struct {
				_, err = m.marshalRefs(o, field.Interface())
				if err != nil {
					return nil, err
				}
			}
			if newField, ok := field.Interface().(ResourceRef); ok {
				var serializedRef any
				serializedRef, err = m.ConstructEvaluator.serializeRef(o, newField)
				if err != nil {
					return nil, err
				}
				ref.Field(i).Set(reflect.ValueOf(serializedRef))
			}
		}
	case reflect.Map:
		for _, key := range ref.MapKeys() {
			field := reflectutil.GetConcreteElement(ref.MapIndex(key))
			switch field.Kind() {
			case reflect.Map, reflect.Struct, reflect.Interface, reflect.Slice | reflect.Array, reflect.Ptr:
				mField, err := m.marshalRefs(o, field.Interface())
				if err != nil {
					return nil, err
				}
				ref.SetMapIndex(key, reflect.ValueOf(mField))
			default:
				if field.IsValid() {
					if newField, ok := field.Interface().(ResourceRef); ok {
						var serializedRef any
						serializedRef, err = m.ConstructEvaluator.serializeRef(o, newField)
						if err != nil {
							return nil, err
						}
						ref.SetMapIndex(key, reflect.ValueOf(serializedRef))
					}
				}
			}
		}
	case reflect.Slice | reflect.Array:
		for i := 0; i < ref.Len(); i++ {
			field := reflectutil.GetConcreteElement(ref.Index(i))
			if field.Kind() == reflect.Struct {
				_, err = m.marshalRefs(o, field.Interface())
				if err != nil {
					return nil, err
				}
			}
			if field.Kind() == reflect.Map {
				_, err = m.marshalRefs(o, field.Interface())
				if err != nil {
					return nil, err
				}
			}
			if field.IsValid() {
				if newField, ok := field.Interface().(ResourceRef); ok {
					var serializedRef any
					serializedRef, err = m.ConstructEvaluator.serializeRef(o, newField)
					if err != nil {
						return nil, err
					}
					ref.Index(i).Set(reflect.ValueOf(serializedRef))
				}
			}
		}
	case reflect.Interface | reflect.Pointer:
		if ref.Elem().Kind() == reflect.Struct {
			_, err = m.marshalRefs(o, ref.Elem().Interface())
			if err != nil {
				return nil, err
			}
		}
	default:
		if ref.IsValid() {
			if newField, ok := ref.Interface().(ResourceRef); ok {
				var serializedRef any
				serializedRef, err = m.ConstructEvaluator.serializeRef(o, newField)
				if err != nil {
					return nil, err
				}
				ref.Set(reflect.ValueOf(serializedRef))
			}
		}
	}

	if ref.IsValid() {
		return ref.Interface(), nil
	}
	return nil, nil
}
