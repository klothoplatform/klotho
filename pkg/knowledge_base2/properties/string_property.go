package properties

import (
	"fmt"
	"strings"

	"github.com/klothoplatform/klotho/pkg/collectionutil"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

type (
	StringProperty struct {
		SanitizeTmpl  *knowledgebase.SanitizeTmpl `yaml:"sanitize"`
		AllowedValues []string                    `yaml:"allowed_values"`
		DefaultValue  *string                     `json:"default_value" yaml:"default_value"`
		SharedPropertyFields
		*knowledgebase.PropertyDetails
	}
)

func (str *StringProperty) SetProperty(resource *construct.Resource, value any) error {
	if val, ok := value.(string); ok {
		return resource.SetProperty(str.Path, val)
	} else if val, ok := value.(construct.PropertyRef); ok {
		return resource.SetProperty(str.Path, val)
	}
	return fmt.Errorf("invalid string value %v", value)
}

func (str *StringProperty) AppendProperty(resource *construct.Resource, value any) error {
	return str.SetProperty(resource, value)
}

func (str *StringProperty) RemoveProperty(resource *construct.Resource, value any) error {
	delete(resource.Properties, str.Path)
	return nil
}

func (s *StringProperty) Details() *knowledgebase.PropertyDetails {
	return s.PropertyDetails
}

func (s *StringProperty) Clone() knowledgebase.Property {
	return &StringProperty{
		DefaultValue:  s.DefaultValue,
		AllowedValues: s.AllowedValues,
		SanitizeTmpl:  s.SanitizeTmpl,
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

func (s *StringProperty) GetDefaultValue(ctx knowledgebase.DynamicValueContext, data knowledgebase.DynamicValueData) (any, error) {
	if s.DefaultValue != nil {
		return *s.DefaultValue, nil
	} else if s.DefaultValueTemplate != nil {
		var result construct.ResourceId
		err := ctx.ExecuteTemplateDecode(s.DefaultValueTemplate, data, &result)
		if err != nil {
			return decodeAsPropertyRef(s.DefaultValueTemplate, ctx, data)
		}
		return result, nil
	}
	return nil, nil
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
	return nil, fmt.Errorf("invalid string value %v", value)
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

func (s *StringProperty) Validate(value any, properties construct.Properties) error {
	stringVal, ok := value.(string)
	if !ok {
		return fmt.Errorf("invalid string value %v", value)
	}
	if s.AllowedValues != nil {
		if !collectionutil.Contains(s.AllowedValues, stringVal) {
			return fmt.Errorf("value %s is not allowed. allowed values are %s", stringVal, s.AllowedValues)
		}
	}

	if s.SanitizeTmpl != nil {
		oldVal := stringVal
		_, err := s.SanitizeTmpl.Execute(stringVal)
		if err != nil {
			return err
		}
		if oldVal != stringVal {
			return fmt.Errorf("value %s was sanitized to %s", oldVal, stringVal)
		}
	}
	return nil
}

func (s *StringProperty) SubProperties() map[string]knowledgebase.Property {
	return nil
}