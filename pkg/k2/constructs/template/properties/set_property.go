package properties

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/k2/constructs/template/property"

	"github.com/klothoplatform/klotho/pkg/set"
)

type (
	SetProperty struct {
		MinLength    *int
		MaxLength    *int
		ItemProperty property.Property
		Properties   property.PropertyMap
		SharedPropertyFields
		property.PropertyDetails
	}
)

func (s *SetProperty) SetProperty(properties construct.Properties, value any) error {
	switch val := value.(type) {
	case set.HashedSet[string, any]:
		return properties.SetProperty(s.Path, val)
	}

	if val, ok := value.(set.HashedSet[string, any]); ok {
		return properties.SetProperty(s.Path, val)
	}
	return fmt.Errorf("invalid set value %v", value)
}

func (s *SetProperty) AppendProperty(properties construct.Properties, value any) error {
	propVal, err := properties.GetProperty(s.Path)
	if err != nil && !errors.Is(err, construct.ErrPropertyDoesNotExist) {
		return err
	}
	if propVal == nil {
		if val, ok := value.(set.HashedSet[string, any]); ok {
			return s.SetProperty(properties, val)
		}
	}
	return properties.AppendProperty(s.Path, value)
}

func (s *SetProperty) RemoveProperty(properties construct.Properties, value any) error {
	propVal, err := properties.GetProperty(s.Path)
	if errors.Is(err, construct.ErrPropertyDoesNotExist) {
		return nil
	}
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
	return s.SetProperty(properties, propSet)
}

func (s *SetProperty) Details() *property.PropertyDetails {
	return &s.PropertyDetails
}

func (s *SetProperty) Clone() property.Property {
	var itemProp property.Property
	if s.ItemProperty != nil {
		itemProp = s.ItemProperty.Clone()
	}
	var props property.PropertyMap
	if s.Properties != nil {
		props = s.Properties.Clone()
	}
	clone := *s
	clone.ItemProperty = itemProp
	clone.Properties = props
	return &clone
}

func (s *SetProperty) GetDefaultValue(ctx property.ExecutionContext, data any) (any, error) {
	if s.DefaultValue == nil {
		return nil, nil
	}
	return s.Parse(s.DefaultValue, ctx, data)
}

func (s *SetProperty) Parse(value any, ctx property.ExecutionContext, data any) (any, error) {
	var result = set.HashedSet[string, any]{
		Hasher: func(s any) string {
			return fmt.Sprintf("%v", s)
		},
		Less: func(s1, s2 string) bool {
			return s1 < s2
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
			err := ctx.ExecuteUnmarshal(strVal, data, &vals)
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

func (s *SetProperty) Validate(properties construct.Properties, value any) error {
	if value == nil {
		if s.Required {
			return fmt.Errorf(property.ErrRequiredProperty, s.Path)
		}
		return nil
	}
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

	// Only validate values if its a primitive list, otherwise let the sub properties handle their own validation
	if s.ItemProperty != nil {
		var errs error
		hasSanitized := false
		validSet := set.HashedSet[string, any]{Hasher: setVal.Hasher}
		for _, item := range setVal.ToSlice() {
			if err := s.ItemProperty.Validate(properties, item); err != nil {
				var sanitizeErr *property.SanitizeError
				if errors.As(err, &sanitizeErr) {
					validSet.Add(sanitizeErr.Sanitized)
					hasSanitized = true
				} else {
					errs = errors.Join(errs, fmt.Errorf("invalid item %v: %v", item, err))
				}
			} else {
				validSet.Add(item)
			}
		}
		if errs != nil {
			return errs
		}
		if hasSanitized {
			return &property.SanitizeError{
				Input:     setVal,
				Sanitized: validSet,
			}
		}
	}
	return nil
}

func (s *SetProperty) SubProperties() property.PropertyMap {
	return s.Properties
}

func (s *SetProperty) Item() property.Property {
	return s.ItemProperty
}
