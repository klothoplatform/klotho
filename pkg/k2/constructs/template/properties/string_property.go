package properties

import (
	"errors"
	"fmt"
	"strings"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/k2/constructs/template/property"

	"github.com/klothoplatform/klotho/pkg/collectionutil"
)

type (
	StringProperty struct {
		SanitizeTmpl  *property.SanitizeTmpl
		AllowedValues []string
		SharedPropertyFields
		property.PropertyDetails
	}
)

func (str *StringProperty) SetProperty(properties construct.Properties, value any) error {
	if val, ok := value.(string); ok {
		return properties.SetProperty(str.Path, val)
	} else if val, ok := value.(construct.PropertyRef); ok {
		return properties.SetProperty(str.Path, val)
	}
	return fmt.Errorf("could not set string property: invalid string value %v", value)
}

func (str *StringProperty) AppendProperty(properties construct.Properties, value any) error {
	return str.SetProperty(properties, value)
}

func (str *StringProperty) RemoveProperty(properties construct.Properties, value any) error {
	propVal, err := properties.GetProperty(str.Path)
	if errors.Is(err, construct.ErrPropertyDoesNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if propVal == nil {
		return nil
	}
	return properties.RemoveProperty(str.Path, nil)
}

func (str *StringProperty) Details() *property.PropertyDetails {
	return &str.PropertyDetails
}

func (str *StringProperty) Clone() property.Property {
	clone := *str
	return &clone
}

func (str *StringProperty) GetDefaultValue(ctx property.ExecutionContext, data any) (any, error) {
	if str.DefaultValue == nil {
		return nil, nil
	}
	return str.Parse(str.DefaultValue, ctx, data)
}

func (str *StringProperty) Parse(value any, ctx property.ExecutionContext, data any) (any, error) {
	switch val := value.(type) {
	case string:
		err := ctx.ExecuteUnmarshal(val, data, &val)
		return val, err

	case int, int32, int64, float32, float64, bool:
		return fmt.Sprintf("%v", val), nil
	}
	return nil, fmt.Errorf("could not parse string property: invalid string value %v (%[1]T)", value)
}

func (str *StringProperty) ZeroValue() any {
	return ""
}

func (str *StringProperty) Contains(value any, contains any) bool {
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

func (str *StringProperty) Type() string {
	return "string"
}

func (str *StringProperty) Validate(properties construct.Properties, value any) error {
	if value == nil {
		if str.Required {
			return fmt.Errorf(property.ErrRequiredProperty, str.Path)
		}
		return nil
	}
	stringVal, ok := value.(string)
	if !ok {
		return fmt.Errorf("value %v is not a string", value)
	}

	if len(str.AllowedValues) > 0 && !collectionutil.Contains(str.AllowedValues, stringVal) {
		return fmt.Errorf("value %s is not allowed. allowed values are %s", stringVal, str.AllowedValues)
	}

	if str.SanitizeTmpl != nil {
		return str.SanitizeTmpl.Check(stringVal)
	}
	return nil
}

func (str *StringProperty) SubProperties() property.PropertyMap {
	return nil
}
