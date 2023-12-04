package properties

import (
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

type (
	AnyProperty struct {
		SharedPropertyFields
		knowledgebase.PropertyDetails
	}
)

func (a *AnyProperty) SetProperty(resource *construct.Resource, value any) error {
	return resource.SetProperty(a.Path, value)
}

func (a *AnyProperty) AppendProperty(resource *construct.Resource, value any) error {
	propVal, err := resource.GetProperty(a.Path)
	if err != nil {
		return err
	}
	if propVal == nil {
		return a.SetProperty(resource, value)
	}
	return resource.AppendProperty(a.Path, value)
}

func (a *AnyProperty) RemoveProperty(resource *construct.Resource, value any) error {
	return resource.RemoveProperty(a.Path, value)
}

func (a *AnyProperty) Details() *knowledgebase.PropertyDetails {
	return &a.PropertyDetails
}

func (a *AnyProperty) Clone() knowledgebase.Property {
	clone := *a
	return &clone
}

func (a *AnyProperty) GetDefaultValue(ctx knowledgebase.DynamicContext, data knowledgebase.DynamicValueData) (any, error) {
	if a.DefaultValue == nil {
		return nil, nil
	}
	return a.Parse(a.DefaultValue, ctx, data)
}

func (a *AnyProperty) Parse(value any, ctx knowledgebase.DynamicContext, data knowledgebase.DynamicValueData) (any, error) {
	if val, ok := value.(string); ok {
		// first check if its a resource id
		rType := ResourceProperty{}
		id, err := rType.Parse(val, ctx, data)
		if err == nil {
			return id, nil
		}

		// check if its a property ref
		ref, err := ParsePropertyRef(val, ctx, data)
		if err == nil {
			return ref, nil
		}

		// check if its any other template string
		var result any
		err = ctx.ExecuteDecode(val, data, &result)
		if err == nil {
			return result, nil
		}
	}

	if mapVal, ok := value.(map[string]any); ok {
		m := MapProperty{KeyProperty: &StringProperty{}, ValueProperty: &AnyProperty{}}
		return m.Parse(mapVal, ctx, data)
	}

	if listVal, ok := value.([]any); ok {
		l := ListProperty{ItemProperty: &AnyProperty{}}
		return l.Parse(listVal, ctx, data)
	}

	return value, nil
}

func (a *AnyProperty) ZeroValue() any {
	return nil
}

func (a *AnyProperty) Contains(value any, contains any) bool {
	if val, ok := value.(string); ok {
		s := StringProperty{}
		return s.Contains(val, contains)
	}
	if mapVal, ok := value.(map[string]any); ok {
		m := MapProperty{KeyProperty: &StringProperty{}, ValueProperty: &AnyProperty{}}
		return m.Contains(mapVal, contains)
	}
	if listVal, ok := value.([]any); ok {
		l := ListProperty{ItemProperty: &AnyProperty{}}
		return l.Contains(listVal, contains)
	}
	return false
}

func (a *AnyProperty) Type() string {
	return "any"
}

func (a *AnyProperty) Validate(value any, properties construct.Properties) error {
	return nil
}

func (a *AnyProperty) SubProperties() knowledgebase.Properties {
	return nil
}
