package properties

import (
	"errors"
	"fmt"
	"reflect"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/klothoplatform/klotho/pkg/set"
)

type (
	SetProperty struct {
		DefaultValue []any                  `json:"default_value" yaml:"default_value"`
		MinLength    *int                   `yaml:"min_length"`
		MaxLength    *int                   `yaml:"max_length"`
		ItemProperty knowledgebase.Property `yaml:"item_property"`
		Properties   knowledgebase.Properties
		SharedPropertyFields
		*knowledgebase.PropertyDetails
	}
)

func (s *SetProperty) SetProperty(resource *construct.Resource, value any) error {
	if val, ok := value.(set.HashedSet[string, any]); ok {
		return resource.SetProperty(s.Path, val)
	}
	return fmt.Errorf("invalid set value %v", value)
}

func (s *SetProperty) AppendProperty(resource *construct.Resource, value any) error {
	return resource.AppendProperty(s.Path, value)
}

func (s *SetProperty) RemoveProperty(resource *construct.Resource, value any) error {
	return resource.RemoveProperty(s.Path, value)
}

func (s *SetProperty) Details() *knowledgebase.PropertyDetails {
	return s.PropertyDetails
}

func (s *SetProperty) Clone() knowledgebase.Property {
	return &SetProperty{
		DefaultValue: s.DefaultValue,
		MinLength:    s.MinLength,
		MaxLength:    s.MaxLength,
		ItemProperty: s.ItemProperty.Clone(),
		Properties:   s.Properties.Clone(),
		SharedPropertyFields: SharedPropertyFields{
			DefaultValueTemplate: s.DefaultValueTemplate,
			ValidityChecks:       s.ValidityChecks,
		},
		PropertyDetails: &knowledgebase.PropertyDetails{
			Name:                  s.Name,
			Path:                  s.Path,
			Required:              s.Required,
			ConfigurationDisabled: s.ConfigurationDisabled,
			DeployTime:            s.DeployTime,
			OperationalRule:       s.OperationalRule,
			Namespace:             s.Namespace,
		},
	}
}

func (s *SetProperty) GetDefaultValue(ctx knowledgebase.DynamicValueContext, data knowledgebase.DynamicValueData) (any, error) {
	if s.DefaultValue != nil {
		return s.DefaultValue, nil
	} else if s.DefaultValueTemplate != nil {
		var result []any
		err := ctx.ExecuteTemplateDecode(s.DefaultValueTemplate, data, &result)
		return result, err
	}
	return nil, nil
}

func (s *SetProperty) Parse(value any, ctx knowledgebase.DynamicContext, data knowledgebase.DynamicValueData) (any, error) {
	var result = set.HashedSet[string, any]{
		Hasher: func(s any) string {
			return fmt.Sprintf("%v", s)
		},
	}
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
		if len(s.Properties) != 0 {
			m := MapProperty{Properties: s.Properties}
			val, err := m.Parse(v, ctx, data)
			if err != nil {
				return nil, err
			}
			result.Add(val)
		} else {
			val, err := s.ItemProperty.Parse(val, ctx, data)
			if err != nil {
				return nil, err
			}
			result.Add(val)
		}
	}
	return result, nil
}

func (s *SetProperty) ZeroValue() any {
	return nil
}

func (s *SetProperty) Contains(value any, contains any) bool {
	valSet, ok := value.(set.HashedSet[string, any])
	if !ok {
		return false
	}
	containsSet, ok := contains.(set.HashedSet[string, any])
	if !ok {
		return false
	}
	for _, v := range containsSet.M {
		for _, val := range valSet.M {
			if reflect.DeepEqual(v, val) {
				return true
			}
		}
	}
	return false
}

// set path
func (s *SetProperty) SetPath(path string) {
	s.Path = path
}

func (s *SetProperty) PropertyName() string {
	return s.Name
}
func (s *SetProperty) Type() string {
	if s.ItemProperty != nil {
		return fmt.Sprintf("set(%s)", s.ItemProperty.Type())
	}
	return "set"
}

func (s *SetProperty) IsRequired() bool {
	return s.Required
}

func (s *SetProperty) Validate(value any, properties construct.Properties) error {
	setVal, ok := value.(set.HashedSet[string, any])
	if !ok {
		return fmt.Errorf("invalid string value %v", value)
	}
	if s.MinLength != nil {
		if setVal.Len() < *s.MinLength {
			return fmt.Errorf("value %s is too short. minimum length is %d", setVal, *s.MinLength)
		}
	}
	if s.MaxLength != nil {
		if setVal.Len() > *s.MaxLength {
			return fmt.Errorf("value %s is too long. maximum length is %d", setVal, *s.MaxLength)
		}
	}

	var errs error
	for _, item := range setVal.ToSlice() {
		if err := s.ItemProperty.Validate(item, properties); err != nil {
			errs = errors.Join(errs, fmt.Errorf("invalid item %v: %v", item, err))
		}
	}
	if errs != nil {
		return errs
	}
	return nil
}

func (s *SetProperty) SubProperties() map[string]knowledgebase.Property {
	return s.Properties
}
