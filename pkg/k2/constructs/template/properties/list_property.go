package properties

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/k2/constructs/template/property"

	"github.com/klothoplatform/klotho/pkg/collectionutil"
)

type (
	ListProperty struct {
		MinLength    *int
		MaxLength    *int
		ItemProperty property.Property
		Properties   property.PropertyMap
		SharedPropertyFields
		property.PropertyDetails
	}
)

func (l *ListProperty) SetProperty(properties construct.Properties, value any) error {
	if val, ok := value.([]any); ok {
		return properties.SetProperty(l.Path, val)
	}
	return fmt.Errorf("invalid list value %v", value)
}

func (l *ListProperty) AppendProperty(properties construct.Properties, value any) error {
	propVal, err := properties.GetProperty(l.Path)
	if err != nil && !errors.Is(err, construct.ErrPropertyDoesNotExist) {
		return err
	}
	if propVal == nil {
		err := l.SetProperty(properties, []any{})
		if err != nil {
			return err
		}
	}
	if l.ItemProperty != nil && !strings.HasPrefix(l.ItemProperty.Type(), "list") {
		if reflect.ValueOf(value).Kind() == reflect.Slice || reflect.ValueOf(value).Kind() == reflect.Array {
			var errs error
			for i := 0; i < reflect.ValueOf(value).Len(); i++ {
				err := properties.AppendProperty(l.Path, reflect.ValueOf(value).Index(i).Interface())
				if err != nil {
					errs = errors.Join(errs, err)
				}
			}
			return errs
		}
	}
	return properties.AppendProperty(l.Path, value)
}

func (l *ListProperty) RemoveProperty(properties construct.Properties, value any) error {
	propVal, err := properties.GetProperty(l.Path)
	if errors.Is(err, construct.ErrPropertyDoesNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if propVal == nil {
		return nil
	}
	if l.ItemProperty != nil && !strings.HasPrefix(l.ItemProperty.Type(), "list") {
		if reflect.ValueOf(value).Kind() == reflect.Slice || reflect.ValueOf(value).Kind() == reflect.Array {
			var errs error
			for i := 0; i < reflect.ValueOf(value).Len(); i++ {
				err := properties.RemoveProperty(l.Path, reflect.ValueOf(value).Index(i).Interface())
				if err != nil {
					errs = errors.Join(errs, err)
				}
			}
			return errs
		}
	}
	return properties.RemoveProperty(l.Path, value)
}

func (l *ListProperty) Details() *property.PropertyDetails {
	return &l.PropertyDetails
}

func (l *ListProperty) Clone() property.Property {
	var itemProp property.Property
	if l.ItemProperty != nil {
		itemProp = l.ItemProperty.Clone()
	}
	var props property.PropertyMap
	if l.Properties != nil {
		props = l.Properties.Clone()
	}
	clone := *l
	clone.ItemProperty = itemProp
	clone.Properties = props
	return &clone
}

func (list *ListProperty) GetDefaultValue(ctx property.ExecutionContext, data any) (any, error) {
	if list.DefaultValue == nil {
		return nil, nil
	}
	return list.Parse(list.DefaultValue, ctx, data)
}

func (list *ListProperty) Parse(value any, ctx property.ExecutionContext, data any) (any, error) {

	var result []any
	val, ok := value.([]any)
	if !ok {
		// before we fail, check to see if the entire value is a template
		if strVal, ok := value.(string); ok {
			var result []any
			err := ctx.ExecuteUnmarshal(strVal, data, &result)
			if err != nil {
				return nil, fmt.Errorf("invalid list value %v: %w", value, err)
			}
			val = result
		} else {
			return nil, fmt.Errorf("invalid list value %v", value)
		}
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
		return collectionutil.Contains(list, contains)
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

func (l *ListProperty) Validate(properties construct.Properties, value any) error {
	if value == nil {
		if l.Required {
			return fmt.Errorf(property.ErrRequiredProperty, l.Path)
		}
		return nil
	}

	listVal, ok := value.([]any)
	if !ok {
		return fmt.Errorf("invalid list value %v", value)
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

	validList := make([]any, len(listVal))
	var errs error
	hasSanitized := false
	for i, v := range listVal {
		if l.ItemProperty != nil {
			err := l.ItemProperty.Validate(properties, v)
			if err != nil {
				var sanitizeErr *property.SanitizeError
				if errors.As(err, &sanitizeErr) {
					validList[i] = sanitizeErr.Sanitized
					hasSanitized = true
				} else {
					errs = errors.Join(errs, err)
				}
			} else {
				validList[i] = v
			}
		} else {
			vmap, ok := v.(map[string]any)
			if !ok {
				return fmt.Errorf("invalid value for list index %d in sub properties validation: expected map[string]any got %T", i, v)
			}
			validIndex := make(map[string]any)
			for _, prop := range l.SubProperties() {
				val, ok := vmap[prop.Details().Name]
				if !ok {
					continue
				}
				err := prop.Validate(properties, val)
				if err != nil {
					var sanitizeErr *property.SanitizeError
					if errors.As(err, &sanitizeErr) {
						validIndex[prop.Details().Name] = sanitizeErr.Sanitized
						hasSanitized = true
					} else {
						errs = errors.Join(errs, err)
					}
				} else {
					validIndex[prop.Details().Name] = val
				}
			}
			validList[i] = validIndex
		}
	}
	if errs != nil {
		return errs
	}
	if hasSanitized {
		return &property.SanitizeError{
			Input:     listVal,
			Sanitized: validList,
		}
	}

	return nil
}

func (l *ListProperty) SubProperties() property.PropertyMap {
	return l.Properties
}

func (l *ListProperty) Item() property.Property {
	return l.ItemProperty
}
