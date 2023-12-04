package properties

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/klothoplatform/klotho/pkg/collectionutil"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

type (
	ListProperty struct {
		MinLength    *int
		MaxLength    *int
		ItemProperty knowledgebase.Property
		Properties   knowledgebase.Properties
		SharedPropertyFields
		knowledgebase.PropertyDetails
	}
)

func (l *ListProperty) SetProperty(resource *construct.Resource, value any) error {
	if val, ok := value.([]any); ok {
		return resource.SetProperty(l.Path, val)
	}
	return fmt.Errorf("invalid list value %v", value)
}

func (l *ListProperty) AppendProperty(resource *construct.Resource, value any) error {
	propval, err := resource.GetProperty(l.Path)
	if err != nil {
		return err
	}
	if propval == nil {
		err := l.SetProperty(resource, []any{})
		if err != nil {
			return err
		}
	}
	if l.ItemProperty != nil && !strings.HasPrefix(l.ItemProperty.Type(), "list") {
		if reflect.ValueOf(value).Kind() == reflect.Slice || reflect.ValueOf(value).Kind() == reflect.Array {
			var errs error
			for i := 0; i < reflect.ValueOf(value).Len(); i++ {
				err := resource.AppendProperty(l.Path, reflect.ValueOf(value).Index(i).Interface())
				if err != nil {
					errs = errors.Join(errs, err)
				}
			}
			return errs
		}
	}
	return resource.AppendProperty(l.Path, value)
}

func (l *ListProperty) RemoveProperty(resource *construct.Resource, value any) error {
	propval, err := resource.GetProperty(l.Path)
	if err != nil {
		return err
	}
	if propval == nil {
		return nil
	}
	if l.ItemProperty != nil && !strings.HasPrefix(l.ItemProperty.Type(), "list") {
		if reflect.ValueOf(value).Kind() == reflect.Slice || reflect.ValueOf(value).Kind() == reflect.Array {
			var errs error
			for i := 0; i < reflect.ValueOf(value).Len(); i++ {
				err := resource.RemoveProperty(l.Path, reflect.ValueOf(value).Index(i).Interface())
				if err != nil {
					errs = errors.Join(errs, err)
				}
			}
			return errs
		}
	}
	return resource.RemoveProperty(l.Path, value)
}

func (l *ListProperty) Details() *knowledgebase.PropertyDetails {
	return &l.PropertyDetails
}

func (l *ListProperty) Clone() knowledgebase.Property {
	var itemProp knowledgebase.Property
	if l.ItemProperty != nil {
		itemProp = l.ItemProperty.Clone()
	}
	var props knowledgebase.Properties
	if l.Properties != nil {
		props = l.Properties.Clone()
	}
	clone := *l
	clone.ItemProperty = itemProp
	clone.Properties = props
	return &clone
}

func (list *ListProperty) GetDefaultValue(ctx knowledgebase.DynamicContext, data knowledgebase.DynamicValueData) (any, error) {
	if list.DefaultValue == nil {
		return nil, nil
	}
	return list.Parse(list.DefaultValue, ctx, data)
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

func (l *ListProperty) SubProperties() knowledgebase.Properties {
	return l.Properties
}
