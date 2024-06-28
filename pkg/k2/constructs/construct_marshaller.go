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
	//TODO: consider look into capturing multiple errors instead of returning the first one
	var cs constraints.ConstraintList
	c, ok := m.ConstructEvaluator.constructs[constructURN]
	if !ok {
		return nil, fmt.Errorf("could not find construct %s", constructURN)
	}

	for id, r := range c.ImportedResources {
		resourceConstraints, err := m.marshalImportedResource(id, r)
		if err != nil {
			return nil, fmt.Errorf("could not marshall imported resource: %w", err)
		}
		cs = append(cs, resourceConstraints...)
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

	return cs, nil
}

func (m *ConstructMarshaller) marshalResource(c *Construct, r *Resource) (constraints.ConstraintList, error) {
	var cs constraints.ConstraintList
	cs = append(cs, &constraints.ApplicationConstraint{
		Operator: "must_exist",
		Node:     r.Id,
	})
	for k, v := range r.Properties {

		v, err := m.marshalRefs(v)
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
func (m *ConstructMarshaller) marshalEdge(c *Construct, e *Edge) (constraints.ConstraintList, error) {

	var from construct.ResourceId
	ref, err := m.ConstructEvaluator.serializeRef(e.From)
	if err != nil {
		return nil, fmt.Errorf("could not serialize from resource id: %w", err)
	}
	err = from.Parse(ref.(string))
	if err != nil {
		return nil, fmt.Errorf("could not parse from resource id: %w", err)
	}

	var to construct.ResourceId
	ref, err = m.ConstructEvaluator.serializeRef(e.To)
	if err != nil {
		return nil, fmt.Errorf("could not serialize to resource id: %w", err)
	}
	err = to.Parse(ref.(string))
	if err != nil {
		return nil, fmt.Errorf("could not parse to resource id: %w", err)
	}
	v, err := m.marshalRefs(e.Data)
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

// marshalImportedResource marshals an imported resource into a list of constraints
func (m *ConstructMarshaller) marshalImportedResource(r construct.ResourceId, props map[string]any) (constraints.ConstraintList, error) {
	// We don't need to marshal ResourceRefs of imported resources when generating constraints,
	// as they were already marshaled when during evaluation of the construct that declared the imported resource.
	var cs constraints.ConstraintList
	cs = append(cs, &constraints.ApplicationConstraint{
		Operator: "import",
		Node:     r,
	})
	for k, v := range props {
		cs = append(cs, &constraints.ResourceConstraint{
			Operator: "equals",
			Target:   r,
			Property: k,
			Value:    v,
		})
	}

	return cs, nil
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
func (m *ConstructMarshaller) marshalRefs(rawVal any) (any, error) {
	if ref, ok := rawVal.(ResourceRef); ok {
		return m.ConstructEvaluator.serializeRef(ref)
	}

	ref := reflectutil.GetConcreteElement(reflect.ValueOf(rawVal))

	var err error
	switch ref.Kind() {
	case reflect.Struct:
		for i := 0; i < ref.NumField(); i++ {
			field := reflectutil.GetConcreteElement(ref.Field(i))
			if field.Kind() == reflect.Struct {
				_, err = m.marshalRefs(field.Interface())
				if err != nil {
					return nil, err
				}
			}
			if newField, ok := field.Interface().(ResourceRef); ok {
				var serializedRef any
				serializedRef, err = m.ConstructEvaluator.serializeRef(newField)
				if err != nil {
					return nil, err
				}
				ref.Field(i).Set(reflect.ValueOf(serializedRef))
			}
		}
	case reflect.Map:
		for _, key := range ref.MapKeys() {
			field := reflectutil.GetConcreteElement(ref.MapIndex(key))
			if field.Kind() == reflect.Struct {
				_, err = m.marshalRefs(field.Interface())
				if err != nil {
					return nil, err
				}
			}
			if field.IsValid() {
				if newField, ok := field.Interface().(ResourceRef); ok {
					var serializedRef any
					serializedRef, err = m.ConstructEvaluator.serializeRef(newField)
					if err != nil {
						return nil, err
					}
					ref.SetMapIndex(key, reflect.ValueOf(serializedRef))
				}
			}

		}
	case reflect.Slice | reflect.Array:
		for i := 0; i < ref.Len(); i++ {
			field := reflectutil.GetConcreteElement(ref.Index(i))
			if field.Kind() == reflect.Struct {
				_, err = m.marshalRefs(field.Interface())
				if err != nil {
					return nil, err
				}
			}
			if field.Kind() == reflect.Map {
				_, err = m.marshalRefs(field.Interface())
				if err != nil {
					return nil, err
				}
			}
			if field.IsValid() {
				if newField, ok := field.Interface().(ResourceRef); ok {
					var serializedRef any
					serializedRef, err = m.ConstructEvaluator.serializeRef(newField)
					if err != nil {
						return nil, err
					}
					ref.Index(i).Set(reflect.ValueOf(serializedRef))
				}
			}
		}
	case reflect.Interface | reflect.Pointer:
		if ref.Elem().Kind() == reflect.Struct {
			_, err = m.marshalRefs(ref.Elem().Interface())
			if err != nil {
				return nil, err
			}
		}
	default:
		if ref.IsValid() {
			if newField, ok := ref.Interface().(ResourceRef); ok {
				var serializedRef any
				serializedRef, err = m.ConstructEvaluator.serializeRef(newField)
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
