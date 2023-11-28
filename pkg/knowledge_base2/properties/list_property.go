package properties

import (
	"errors"
	"fmt"
	"reflect"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

type (
	ListProperty struct {
		DefaultValue []any                  `json:"default_value" yaml:"default_value"`
		MinLength    *int                   `yaml:"min_length"`
		MaxLength    *int                   `yaml:"max_length"`
		ItemProperty knowledgebase.Property `yaml:"item_property"`
		Properties   knowledgebase.Properties
		SharedPropertyFields
		*knowledgebase.PropertyDetails
	}
)

func (l *ListProperty) SetProperty(resource *construct.Resource, value any) error {
	if val, ok := value.([]any); ok {
		return resource.SetProperty(l.Path, val)
	}
	return fmt.Errorf("invalid list value %v", value)
}

func (l *ListProperty) AppendProperty(resource *construct.Resource, value any) error {
	return resource.AppendProperty(l.Path, value)
}

func (l *ListProperty) RemoveProperty(resource *construct.Resource, value any) error {
	return resource.RemoveProperty(l.Path, value)
}

func (l *ListProperty) Details() *knowledgebase.PropertyDetails {
	return l.PropertyDetails
}

func (l *ListProperty) Clone() knowledgebase.Property {
	return &ListProperty{
		DefaultValue: l.DefaultValue,
		MinLength:    l.MinLength,
		MaxLength:    l.MaxLength,
		ItemProperty: l.ItemProperty.Clone(),
		Properties:   l.Properties.Clone(),
		SharedPropertyFields: SharedPropertyFields{
			DefaultValueTemplate: l.DefaultValueTemplate,
			ValidityChecks:       l.ValidityChecks,
		},
		PropertyDetails: &knowledgebase.PropertyDetails{
			Name:                  l.Name,
			Path:                  l.Path,
			Required:              l.Required,
			ConfigurationDisabled: l.ConfigurationDisabled,
			DeployTime:            l.DeployTime,
			OperationalRule:       l.OperationalRule,
			Namespace:             l.Namespace,
		},
	}
}

func (list *ListProperty) GetDefaultValue(ctx knowledgebase.DynamicValueContext, data knowledgebase.DynamicValueData) (any, error) {
	if list.DefaultValue != nil {
		return list.DefaultValue, nil
	} else if list.DefaultValueTemplate != nil {
		var result []any
		err := ctx.ExecuteTemplateDecode(list.DefaultValueTemplate, data, &result)
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	return nil, nil
}

func (list *ListProperty) Parse(value any, ctx knowledgebase.DynamicContext, data knowledgebase.DynamicValueData) (any, error) {

	var result []any
	val, ok := value.([]any)
	if !ok {
		// before we fail, check to see if the entire value is a template
		if strVal, ok := value.(string); ok {
			var result []any
			err := ctx.ExecuteDecode(strVal, data, &result)
			return result, err
		}
		return nil, fmt.Errorf("invalid list value %v", value)
	}

	for _, v := range val {
		if len(list.Properties) != 0 {
			m := MapProperty{Properties: list.Properties}
			val, err := m.Parse(v, ctx, data)
			if err != nil {
				return nil, err
			}
			result = append(result, val)
		} else {
			val, err := list.ItemProperty.Parse(v, ctx, data)
			if err != nil {
				return nil, err
			}
			result = append(result, val)
		}
	}
	return result, nil
}

func (l *ListProperty) ZeroValue() any {
	return nil
}

func (l *ListProperty) Contains(value any, contains any) bool {
	list, ok := value.([]any)
	if !ok {
		return false
	}
	containsList, ok := contains.([]any)
	if !ok {
		return false
	}
	for _, v := range list {
		for _, cv := range containsList {
			if reflect.DeepEqual(v, cv) {
				return true
			}
		}
	}
	return false
}

func (l *ListProperty) Type() string {
	if l.ItemProperty != nil {
		return fmt.Sprintf("list(%s)", l.ItemProperty.Type())
	}
	return "list"
}

func (l *ListProperty) Validate(value any, properties construct.Properties) error {
	listVal, ok := value.([]any)
	if !ok {
		return fmt.Errorf("invalid map value %v", value)
	}
	if l.MinLength != nil {
		if len(listVal) < *l.MinLength {
			return fmt.Errorf("list value %v is too short. min length is %d", value, *l.MinLength)
		}
	}
	if l.MaxLength != nil {
		if len(listVal) > *l.MaxLength {
			return fmt.Errorf("list value %v is too long. max length is %d", value, *l.MaxLength)
		}
	}
	var errs error

	for _, v := range listVal {
		if len(l.Properties) != 0 {
			m := MapProperty{Properties: l.Properties}
			err := m.Validate(v, properties)
			if err != nil {
				errs = errors.New(errs.Error() + "\n" + err.Error())
			}
		} else {
			err := l.ItemProperty.Validate(v, properties)
			if err != nil {
				errs = errors.New(errs.Error() + "\n" + err.Error())
			}
		}
	}
	if errs != nil {
		return errs
	}
	return nil
}

func (l *ListProperty) SubProperties() map[string]knowledgebase.Property {
	return l.Properties
}
