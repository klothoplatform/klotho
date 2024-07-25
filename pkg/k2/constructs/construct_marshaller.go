package constructs

import (
	"fmt"
	"reflect"

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
	c, ok := m.ConstructEvaluator.constructs.Get(constructURN)
	if !ok {
		return nil, fmt.Errorf("could not find construct %s", constructURN)
	}

	for _, r := range c.Resources {
		resourceConstraints, err := m.marshalResource(c, r)
		if err != nil {
			return nil, fmt.Errorf("could not marshal resource: %w", err)
		}
		cs = append(cs, resourceConstraints...)
	}

	for _, e := range c.Edges {
		edgeConstraints, err := m.marshalEdge(c, e)
		if err != nil {
			return nil, fmt.Errorf("could not marshal edge: %w", err)
		}
		cs = append(cs, edgeConstraints...)
	}

	for _, o := range c.OutputDeclarations {
		outputConstraints, err := m.marshalOutput(o)
		if err != nil {
			return nil, fmt.Errorf("could not marshal output: %w", err)
		}
		cs = append(cs, outputConstraints...)
	}

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
func (m *ConstructMarshaller) marshalRefs(o InfraOwner, rawVal any) (any, error) {
	// Handle ResourceRef types directly
	if ref, ok := rawVal.(ResourceRef); ok {
		switch ref.Type {
		case ResourceRefTypeInterpolated:
			return m.marshalRefs(o, ref.ResourceKey)
		case ResourceRefTypeTemplate:
			ref.ConstructURN = o.GetURN()
			return ref, nil
		default:
			return rawVal, nil
		}
	}

	// Get the concrete value
	ref := reflect.ValueOf(rawVal)
	if ref.Kind() == reflect.Ptr {
		if ref.IsNil() {
			return rawVal, nil
		}
		ref = ref.Elem()
	}

	if !ref.IsValid() {
		return rawVal, nil
	}

	switch ref.Kind() {
	case reflect.Struct:
		for i := 0; i < ref.NumField(); i++ {
			field := ref.Field(i)
			if !field.CanSet() {
				continue
			}
			fieldValue := reflectutil.GetConcreteElement(field)
			_, err := m.marshalRefs(o, fieldValue.Interface())
			if err != nil {
				return nil, err
			}
		}
	case reflect.Map:
		for _, key := range ref.MapKeys() {
			field := reflectutil.GetConcreteElement(ref.MapIndex(key))
			serializedField, err := m.marshalRefs(o, field.Interface())
			if err != nil {
				return nil, err
			}
			ref.SetMapIndex(key, reflect.ValueOf(serializedField))
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < ref.Len(); i++ {
			field := reflectutil.GetConcreteElement(ref.Index(i))
			serializedField, err := m.marshalRefs(o, field.Interface())
			if err != nil {
				return nil, err
			}
			ref.Index(i).Set(reflect.ValueOf(serializedField))
		}
	}

	return rawVal, nil
}
