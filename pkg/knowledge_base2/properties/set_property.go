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
		MinLength    *int
		MaxLength    *int
		ItemProperty knowledgebase.Property
		Properties   knowledgebase.Properties
		SharedPropertyFields
		knowledgebase.PropertyDetails
	}
)

func (s *SetProperty) SetProperty(resource *construct.Resource, value any) error {
	if val, ok := value.(set.HashedSet[string, any]); ok {
		return resource.SetProperty(s.Path, val)
	}
	return fmt.Errorf("invalid set value %v", value)
}

func (s *SetProperty) AppendProperty(resource *construct.Resource, value any) error {
	propVal, err := resource.GetProperty(s.Path)
	if err != nil {
		return err
	}
	if propVal == nil {
		if val, ok := value.(set.HashedSet[string, any]); ok {
			return s.SetProperty(resource, val)
		}
	}
	return resource.AppendProperty(s.Path, value)
}

func (s *SetProperty) RemoveProperty(resource *construct.Resource, value any) error {
	propVal, err := resource.GetProperty(s.Path)
	if err != nil {
		return err
	}
	if propVal == nil {
		return nil
	}
	propSet, ok := propVal.(set.HashedSet[string, any])
	if !ok {
		return errors.New("invalid set value")
	}
	if val, ok := value.(set.HashedSet[string, any]); ok {
		for _, v := range val.ToSlice() {
			propSet.Remove(v)
		}
	} else {
		return fmt.Errorf("invalid set value %v", value)
	}
	return s.SetProperty(resource, propSet)
}

func (s *SetProperty) Details() *knowledgebase.PropertyDetails {
	return &s.PropertyDetails
}

func (s *SetProperty) Clone() knowledgebase.Property {
	var itemProp knowledgebase.Property
	if s.ItemProperty != nil {
		itemProp = s.ItemProperty.Clone()
	}
	var props knowledgebase.Properties
	if s.Properties != nil {
		props = s.Properties.Clone()
	}
	clone := *s
	clone.ItemProperty = itemProp
	clone.Properties = props
	return &clone
}

func (s *SetProperty) GetDefaultValue(ctx knowledgebase.DynamicContext, data knowledgebase.DynamicValueData) (any, error) {
	if s.DefaultValue == nil {
		return nil, nil
	}
	return s.Parse(s.DefaultValue, ctx, data)
}

func (s *SetProperty) Parse(value any, ctx knowledgebase.DynamicContext, data knowledgebase.DynamicValueData) (any, error) {
	var result = set.HashedSet[string, any]{
		Hasher: func(s any) string {
			return fmt.Sprintf("%v", s)
		},
	}

	var vals []any
	if valSet, ok := value.(set.HashedSet[string, any]); ok {
		vals = valSet.ToSlice()
	} else if val, ok := value.([]any); ok {
		vals = val
	} else {
		// before we fail, check to see if the entire value is a template
		if strVal, ok := value.(string); ok {
			err := ctx.ExecuteDecode(strVal, data, &vals)
			if err != nil {
				return nil, err
			}
		}
	}

	for _, v := range vals {
		if len(s.Properties) != 0 {
			m := MapProperty{Properties: s.Properties}
			val, err := m.Parse(v, ctx, data)
			if err != nil {
				return nil, err
			}
			result.Add(val)
		} else {
			val, err := s.ItemProperty.Parse(v, ctx, data)
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

	for _, val := range valSet.M {
		if reflect.DeepEqual(contains, val) {
			return true
		}
	}

	return false
}

func (s *SetProperty) Type() string {
	if s.ItemProperty != nil {
		return fmt.Sprintf("set(%s)", s.ItemProperty.Type())
	}
	return "set"
}

func (s *SetProperty) Validate(resource *construct.Resource, value any) error {
	setVal, ok := value.(set.HashedSet[string, any])
	if !ok {
		return fmt.Errorf("could not validate set property: invalid set value %v", value)
	}
	if s.MinLength != nil {
		if setVal.Len() < *s.MinLength {
			return fmt.Errorf("value %s is too short. minimum length is %d", setVal.M, *s.MinLength)
		}
	}
	if s.MaxLength != nil {
		if setVal.Len() > *s.MaxLength {
			return fmt.Errorf("value %s is too long. maximum length is %d", setVal.M, *s.MaxLength)
		}
	}

	var errs error
	for _, item := range setVal.ToSlice() {
		if err := s.ItemProperty.Validate(resource, item); err != nil {
			errs = errors.Join(errs, fmt.Errorf("invalid item %v: %v", item, err))
		}
	}
	if errs != nil {
		return errs
	}
	return nil
}

func (s *SetProperty) SubProperties() knowledgebase.Properties {
	return s.Properties
}

func (s *SetProperty) Item() knowledgebase.Property {
	return s.ItemProperty
}
