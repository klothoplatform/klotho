package properties

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/k2/constructs/template/property"
)

type (
	AnyProperty struct {
		SharedPropertyFields
		property.PropertyDetails
	}
)

func (a *AnyProperty) SetProperty(properties construct.Properties, value any) error {
	return properties.SetProperty(a.Path, value)
}

func (a *AnyProperty) AppendProperty(properties construct.Properties, value any) error {
	propVal, err := properties.GetProperty(a.Path)
	if err != nil {
		return err
	}
	if propVal == nil {
		return a.SetProperty(properties, value)
	}
	return properties.AppendProperty(a.Path, value)
}

func (a *AnyProperty) RemoveProperty(properties construct.Properties, value any) error {
	return properties.RemoveProperty(a.Path, value)
}

func (a *AnyProperty) Details() *property.PropertyDetails {
	return &a.PropertyDetails
}

func (a *AnyProperty) Clone() property.Property {
	clone := *a
	return &clone
}

func (a *AnyProperty) GetDefaultValue(ctx property.ExecutionContext, data any) (any, error) {
	if a.DefaultValue == nil {
		return nil, nil
	}
	return a.Parse(a.DefaultValue, ctx, data)
}

func (a *AnyProperty) Parse(value any, ctx property.ExecutionContext, data any) (any, error) {
	if val, ok := value.(string); ok {
		// check if its any other template string
		var result any
		err := ctx.ExecuteUnmarshal(val, data, &result)
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

func (a *AnyProperty) Validate(properties construct.Properties, value any) error {
	if a.Required && value == nil {
		return fmt.Errorf(property.ErrRequiredProperty, a.Path)
	}
	return nil
}

func (a *AnyProperty) SubProperties() property.PropertyMap {
	return nil
}
