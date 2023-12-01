package properties

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/collectionutil"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

type (
	ResourceProperty struct {
		AllowedTypes construct.ResourceList
		SharedPropertyFields
		knowledgebase.PropertyDetails
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
	propVal, err := resource.GetProperty(r.Path)
	if err != nil {
		return err
	}
	if propVal == nil {
		return nil
	}
	propId, ok := propVal.(construct.ResourceId)
	if !ok {
		return fmt.Errorf("error attempting to remove resource property: invalid property value %v", propVal)
	}
	valId, ok := value.(construct.ResourceId)
	if !ok {
		return fmt.Errorf("error attempting to remove resource property: invalid resource value %v", value)
	}
	if !propId.Matches(valId) {
		return fmt.Errorf("error attempting to remove resource property: resource value %v does not match property value %v", value, propVal)
	}
	delete(resource.Properties, r.Path)
	return nil
}

func (r *ResourceProperty) Details() *knowledgebase.PropertyDetails {
	return &r.PropertyDetails
}
func (r *ResourceProperty) Clone() knowledgebase.Property {
	return &ResourceProperty{
		AllowedTypes: r.AllowedTypes,
		SharedPropertyFields: SharedPropertyFields{
			DefaultValue:   r.DefaultValue,
			ValidityChecks: r.ValidityChecks,
		},
		PropertyDetails: knowledgebase.PropertyDetails{
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
	if r.DefaultValue == nil {
		return nil, nil
	}
	return r.Parse(r.DefaultValue, ctx, data)
}

func (r *ResourceProperty) Parse(value any, ctx knowledgebase.DynamicContext, data knowledgebase.DynamicValueData) (any, error) {
	if val, ok := value.(string); ok {
		id, err := knowledgebase.ExecuteDecodeAsResourceId(ctx, val, data)
		if !id.IsZero() && len(r.AllowedTypes) > 0 && !r.AllowedTypes.MatchesAny(id) {
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
		if len(r.AllowedTypes) > 0 && !r.AllowedTypes.MatchesAny(id) {
			return nil, fmt.Errorf("resource value %v does not match type %s", value, r.AllowedTypes)
		}
		return id, nil
	}
	if val, ok := value.(construct.ResourceId); ok {
		if len(r.AllowedTypes) > 0 && !r.AllowedTypes.MatchesAny(val) {
			return nil, fmt.Errorf("resource value %v does not match type %s", value, r.AllowedTypes)
		}
		return val, nil
	}
	val, err := ParsePropertyRef(value, ctx, data)
	if err == nil {
		if len(r.AllowedTypes) > 0 && !r.AllowedTypes.MatchesAny(val.Resource) {
			return nil, fmt.Errorf("resource value %v does not match type %s", value, r.AllowedTypes)
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
