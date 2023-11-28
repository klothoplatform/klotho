package properties

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/collectionutil"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

type (
	ResourceProperty struct {
		DefaultValue *construct.ResourceId  `json:"default_value" yaml:"default_value"`
		AllowedTypes construct.ResourceList `yaml:"allowed_types"`
		Namespace    bool                   `yaml:"namespace"`
		SharedPropertyFields
		*knowledgebase.PropertyDetails
	}
)

func (r *ResourceProperty) SetProperty(resource *construct.Resource, value any) error {
	if val, ok := value.(construct.ResourceId); ok {
		return resource.SetProperty(r.Path, val)
	} else if val, ok := value.(construct.PropertyRef); ok {
		return resource.SetProperty(r.Path, val)
	}
	return fmt.Errorf("invalid resource value %v", value)
}

func (r *ResourceProperty) AppendProperty(resource *construct.Resource, value any) error {
	return r.SetProperty(resource, value)
}

func (r *ResourceProperty) RemoveProperty(resource *construct.Resource, value any) error {
	delete(resource.Properties, r.Path)
	return nil
}

func (r *ResourceProperty) Details() *knowledgebase.PropertyDetails {
	return r.PropertyDetails
}
func (r *ResourceProperty) Clone() knowledgebase.Property {
	return &ResourceProperty{
		DefaultValue: r.DefaultValue,
		AllowedTypes: r.AllowedTypes,
		Namespace:    r.Namespace,
		SharedPropertyFields: SharedPropertyFields{
			DefaultValueTemplate: r.DefaultValueTemplate,
			ValidityChecks:       r.ValidityChecks,
		},
		PropertyDetails: &knowledgebase.PropertyDetails{
			Name:                  r.Name,
			Path:                  r.Path,
			Required:              r.Required,
			ConfigurationDisabled: r.ConfigurationDisabled,
			DeployTime:            r.DeployTime,
			OperationalRule:       r.OperationalRule,
			Namespace:             r.Namespace,
		},
	}
}

func (r *ResourceProperty) GetDefaultValue(ctx knowledgebase.DynamicValueContext, data knowledgebase.DynamicValueData) (any, error) {
	if r.DefaultValue != nil {
		return *r.DefaultValue, nil
	} else if r.DefaultValueTemplate != nil {
		var result construct.ResourceId
		err := ctx.ExecuteTemplateDecode(r.DefaultValueTemplate, data, &result)
		if err != nil {
			return decodeAsPropertyRef(r.DefaultValueTemplate, ctx, data)
		}
		return result, nil
	}
	return nil, nil
}

func (r *ResourceProperty) Parse(value any, ctx knowledgebase.DynamicContext, data knowledgebase.DynamicValueData) (any, error) {
	if val, ok := value.(string); ok {
		id, err := knowledgebase.ExecuteDecodeAsResourceId(ctx, val, data)
		if !id.IsZero() && !r.AllowedTypes.MatchesAny(id) {
			return nil, fmt.Errorf("resource value %v does not match allowed types %s", value, r.AllowedTypes)
		}
		return id, err
	}
	if val, ok := value.(map[string]interface{}); ok {
		id := construct.ResourceId{
			Type:     val["type"].(string),
			Name:     val["name"].(string),
			Provider: val["provider"].(string),
		}
		if namespace, ok := val["namespace"]; ok {
			id.Namespace = namespace.(string)
		}
		if !r.AllowedTypes.MatchesAny(id) {
			return nil, fmt.Errorf("resource value %v does not match type %s", value, r.AllowedTypes)
		}
		return id, nil
	}
	if val, ok := value.(construct.ResourceId); ok {
		if !r.AllowedTypes.MatchesAny(val) {
			return nil, fmt.Errorf("resource value %v does not match type %s", value, r.AllowedTypes)
		}
		return val, nil
	}
	val, err := ParsePropertyRef(value, ctx, data)
	if err == nil {
		if ptype, ok := val.(construct.PropertyRef); ok {
			if !r.AllowedTypes.MatchesAny(ptype.Resource) {
				return nil, fmt.Errorf("resource value %v does not match type %s", value, r.AllowedTypes)
			}
		}
		return val, nil
	}
	return nil, fmt.Errorf("invalid resource value %v", value)
}

func (r *ResourceProperty) ZeroValue() any {
	return construct.ResourceId{}
}

func (r *ResourceProperty) Contains(value any, contains any) bool {
	return false
}

// set path
func (r *ResourceProperty) SetPath(path string) {
	r.Path = path
}

func (r *ResourceProperty) PropertyName() string {
	return r.Name
}
func (r *ResourceProperty) Type() string {
	if len(r.AllowedTypes) > 0 {
		typeString := ""
		for i, t := range r.AllowedTypes {
			typeString += t.String()
			if i < len(r.AllowedTypes)-1 {
				typeString += ", "
			}
		}
		return fmt.Sprintf("resource(%s)", typeString)
	}
	return "resource"
}

func (r *ResourceProperty) IsRequired() bool {
	return r.Required
}

func (r *ResourceProperty) Validate(value any, properties construct.Properties) error {
	id, ok := value.(construct.ResourceId)
	if !ok {
		return fmt.Errorf("invalid resource value %v", value)
	}
	if !collectionutil.Contains(r.AllowedTypes, id) {
		return fmt.Errorf("resource value %v does not match allowed types %s", value, r.AllowedTypes)
	}
	return nil
}

func (r *ResourceProperty) SubProperties() map[string]knowledgebase.Property {
	return nil
}
