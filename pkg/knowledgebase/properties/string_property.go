package properties

import (
	"fmt"
	"strings"

	"github.com/klothoplatform/klotho/pkg/collectionutil"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledgebase"
)

type (
	StringProperty struct {
		SanitizeTmpl  *knowledgebase.SanitizeTmpl
		AllowedValues []string
		SharedPropertyFields
		knowledgebase.PropertyDetails
	}
)

func (str *StringProperty) SetProperty(resource *construct.Resource, value any) error {
	if val, ok := value.(string); ok {
		return resource.SetProperty(str.Path, val)
	} else if val, ok := value.(construct.PropertyRef); ok {
		return resource.SetProperty(str.Path, val)
	}
	return fmt.Errorf("could not set string property: invalid string value %v", value)
}

func (str *StringProperty) AppendProperty(resource *construct.Resource, value any) error {
	return str.SetProperty(resource, value)
}

func (str *StringProperty) RemoveProperty(resource *construct.Resource, value any) error {
	propVal, err := resource.GetProperty(str.Path)
	if err != nil {
		return err
	}
	if propVal == nil {
		return nil
	}
	return resource.RemoveProperty(str.Path, nil)
}

func (s *StringProperty) Details() *knowledgebase.PropertyDetails {
	return &s.PropertyDetails
}

func (s *StringProperty) Clone() knowledgebase.Property {
	clone := *s
	return &clone
}

func (s *StringProperty) GetDefaultValue(ctx knowledgebase.DynamicContext, data knowledgebase.DynamicValueData) (any, error) {
	if s.DefaultValue == nil {
		return nil, nil
	}
	return s.Parse(s.DefaultValue, ctx, data)
}

func (str *StringProperty) Parse(value any, ctx knowledgebase.DynamicContext, data knowledgebase.DynamicValueData) (any, error) {
	// Here we have to try to parse to a property ref first, since a string representation of a property ref would match string parsing
	val, err := ParsePropertyRef(value, ctx, data)
	if err == nil {
		return val, nil
	}
	if val, ok := value.(string); ok {
		var result string
		err := ctx.ExecuteDecode(val, data, &result)
		return result, err
	}
	return nil, fmt.Errorf("could not parse string property: invalid string value %v", value)
}

func (s *StringProperty) ZeroValue() any {
	return ""
}

func (s *StringProperty) Contains(value any, contains any) bool {
	vString, ok := value.(string)
	if !ok {
		return false
	}
	cString, ok := contains.(string)
	if !ok {
		return false
	}
	return strings.Contains(vString, cString)
}

func (s *StringProperty) Type() string {
	return "string"
}

func (s *StringProperty) Validate(resource *construct.Resource, value any, ctx knowledgebase.DynamicContext) error {
	if value == nil {
		if s.Required {
			return fmt.Errorf(knowledgebase.ErrRequiredProperty, s.Path, resource.ID)
		}
		return nil
	}
	stringVal, ok := value.(string)
	if !ok {
		propertyRef, ok := value.(construct.PropertyRef)
		if !ok {
			return fmt.Errorf("could not validate property: invalid string value %v", value)
		}
		refVal, err := ValidatePropertyRef(propertyRef, s.Type(), ctx)
		if err != nil {
			return err
		}
		if refVal == nil {
			return nil
		}
		stringVal, ok = refVal.(string)
		if !ok {
			return fmt.Errorf("could not validate property: invalid string value %v", value)
		}
	}

	if s.AllowedValues != nil && len(s.AllowedValues) > 0 && !collectionutil.Contains(s.AllowedValues, stringVal) {
		return fmt.Errorf("value %s is not allowed. allowed values are %s", stringVal, s.AllowedValues)
	}

	if s.SanitizeTmpl != nil {
		return s.SanitizeTmpl.Check(stringVal)
	}
	return nil
}

func (s *StringProperty) SubProperties() knowledgebase.Properties {
	return nil
}
