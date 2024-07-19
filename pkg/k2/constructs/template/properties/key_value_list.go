package properties

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/k2/constructs/template/property"
)

type (
	KeyValueListProperty struct {
		MinLength     *int
		MaxLength     *int
		KeyProperty   property.Property
		ValueProperty property.Property
		SharedPropertyFields
		property.PropertyDetails
	}

	KeyValuePair struct {
		Key   any `json:"key"`
		Value any `json:"value"`
	}
)

func (kvl *KeyValueListProperty) SetProperty(properties construct.Properties, value any) error {
	list, err := kvl.mapToList(value)
	if err != nil {
		return err
	}
	return properties.SetProperty(kvl.Path, list)
}

func (kvl *KeyValueListProperty) AppendProperty(properties construct.Properties, value any) error {
	list, err := kvl.mapToList(value)
	if err != nil {
		return err
	}
	propVal, err := properties.GetProperty(kvl.Path)
	if err != nil && !errors.Is(err, construct.ErrPropertyDoesNotExist) {
		return err
	}
	if propVal == nil {
		return properties.SetProperty(kvl.Path, list)
	}
	existingList, ok := propVal.([]any)
	if !ok {
		return fmt.Errorf("invalid existing property value %v", propVal)
	}
	return properties.SetProperty(kvl.Path, append(existingList, list...))
}

func (kvl *KeyValueListProperty) RemoveProperty(properties construct.Properties, value any) error {
	propVal, err := properties.GetProperty(kvl.Path)
	if errors.Is(err, construct.ErrPropertyDoesNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if propVal == nil {
		return nil
	}
	existingList, ok := propVal.([]any)
	if !ok {
		return fmt.Errorf("invalid existing property value %v", propVal)
	}
	removeList, err := kvl.mapToList(value)
	if err != nil {
		return err
	}
	filteredList := make([]any, 0, len(existingList))
	for _, item := range existingList {
		if !kvl.containsKeyValuePair(removeList, item) {
			filteredList = append(filteredList, item)
		}
	}
	return properties.SetProperty(kvl.Path, filteredList)
}

func (kvl *KeyValueListProperty) Details() *property.PropertyDetails {
	return &kvl.PropertyDetails
}

func (kvl *KeyValueListProperty) Clone() property.Property {
	clone := *kvl
	if kvl.KeyProperty != nil {
		clone.KeyProperty = kvl.KeyProperty.Clone()
	}
	if kvl.ValueProperty != nil {
		clone.ValueProperty = kvl.ValueProperty.Clone()
	}
	return &clone
}

func (kvl *KeyValueListProperty) GetDefaultValue(ctx property.ExecutionContext, data any) (any, error) {
	if kvl.DefaultValue == nil {
		return nil, nil
	}
	return kvl.Parse(kvl.DefaultValue, ctx, data)
}

func (kvl *KeyValueListProperty) Parse(value any, ctx property.ExecutionContext, data any) (any, error) {
	list, err := kvl.mapToList(value)
	if err != nil {
		return nil, err
	}
	result := make([]any, 0, len(list))
	for _, item := range list {
		pair, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid key-value pair %v", item)
		}

		key, err := kvl.KeyProperty.Parse(pair[kvl.KeyPropertyName()], ctx, data)
		if err != nil {
			return nil, fmt.Errorf("error parsing key: %w", err)
		}
		value, err := kvl.ValueProperty.Parse(pair[kvl.ValuePropertyName()], ctx, data)
		if err != nil {
			return nil, fmt.Errorf("error parsing value: %w", err)
		}
		result = append(result, map[string]any{
			kvl.KeyPropertyName():   key,
			kvl.ValuePropertyName(): value,
		})
	}
	return result, nil
}

func (kvl *KeyValueListProperty) KeyPropertyName() string {
	return kvl.KeyProperty.Details().Name
}

func (kvl *KeyValueListProperty) ValuePropertyName() string {
	return kvl.ValueProperty.Details().Name
}

func (kvl *KeyValueListProperty) ZeroValue() any {
	return nil
}

func (kvl *KeyValueListProperty) Contains(value any, contains any) bool {
	list, err := kvl.mapToList(value)
	if err != nil {
		return false
	}
	containsList, err := kvl.mapToList(contains)
	if err != nil {
		return false
	}
	for _, item := range containsList {
		if kvl.containsKeyValuePair(list, item) {
			return true
		}
	}
	return false
}

func (kvl *KeyValueListProperty) Type() string {
	return fmt.Sprintf("keyvaluelist(%s,%s)", kvl.KeyProperty.Type(), kvl.ValueProperty.Type())
}

func (kvl *KeyValueListProperty) Validate(properties construct.Properties, value any) error {
	if value == nil {
		if kvl.Required {
			return fmt.Errorf(property.ErrRequiredProperty, kvl.Path)
		}
		return nil
	}
	list, err := kvl.mapToList(value)
	if err != nil {
		return err
	}
	if kvl.MinLength != nil && len(list) < *kvl.MinLength {
		return fmt.Errorf("list value %v is too short. min length is %d", value, *kvl.MinLength)
	}
	if kvl.MaxLength != nil && len(list) > *kvl.MaxLength {
		return fmt.Errorf("list value %v is too long. max length is %d", value, *kvl.MaxLength)
	}
	var errs error
	for _, item := range list {
		pair, ok := item.(map[string]any)
		if !ok {
			errs = errors.Join(errs, fmt.Errorf("invalid key-value pair %v", item))
			continue
		}
		if err := kvl.KeyProperty.Validate(properties, pair[kvl.KeyPropertyName()]); err != nil {
			errs = errors.Join(errs, fmt.Errorf("invalid key %v: %w", pair[kvl.KeyPropertyName()], err))
		}
		if err := kvl.ValueProperty.Validate(properties, pair[kvl.ValuePropertyName()]); err != nil {
			errs = errors.Join(errs, fmt.Errorf("invalid value %v: %w", pair[kvl.ValuePropertyName()], err))
		}
	}
	return errs
}

func (kvl *KeyValueListProperty) SubProperties() property.PropertyMap {
	return nil
}

func (kvl *KeyValueListProperty) mapToList(value any) ([]any, error) {
	switch v := value.(type) {
	case []any:
		return v, nil
	case map[string]any:
		result := make([]any, 0, len(v))
		for key, val := range v {
			result = append(result, map[string]any{
				kvl.KeyPropertyName():   key,
				kvl.ValuePropertyName(): val,
			})
		}
		return result, nil
	default:
		return nil, fmt.Errorf("invalid input type for KeyValueListProperty: %T", value)
	}
}

func (kvl *KeyValueListProperty) containsKeyValuePair(list []any, item any) bool {
	for _, listItem := range list {
		if reflect.DeepEqual(listItem, item) {
			return true
		}
	}
	return false
}
