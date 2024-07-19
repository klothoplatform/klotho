package properties

import (
	"errors"
	"fmt"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/k2/constructs/template/property"
	"github.com/klothoplatform/klotho/pkg/k2/model"
)

type ConstructTemplateIdList []property.ConstructType

func (l ConstructTemplateIdList) MatchesAny(urn model.URN) bool {
	var id property.ConstructType
	err := id.FromURN(urn)
	if err != nil {
		return false
	}
	for _, t := range l {
		if t == id {
			return true
		}
	}
	return false

}

type (
	ConstructProperty struct {
		AllowedTypes ConstructTemplateIdList
		SharedPropertyFields
		property.PropertyDetails
	}
)

func (r *ConstructProperty) SetProperty(properties construct.Properties, value any) error {
	if val, ok := value.(model.URN); ok {
		return properties.SetProperty(r.Path, val)
	}
	return fmt.Errorf("invalid construct URN %v", value)
}

func (r *ConstructProperty) AppendProperty(properties construct.Properties, value any) error {
	return r.SetProperty(properties, value)
}

func (r *ConstructProperty) RemoveProperty(properties construct.Properties, value any) error {
	propVal, err := properties.GetProperty(r.Path)
	if errors.Is(err, construct.ErrPropertyDoesNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if propVal == nil {
		return nil
	}
	propId, ok := propVal.(model.URN)
	if !ok {
		return fmt.Errorf("error attempting to remove construct property: invalid property value %v", propVal)
	}
	valId, ok := value.(model.URN)
	if !ok {
		return fmt.Errorf("error attempting to remove construct property: invalid construct value %v", value)
	}
	if !propId.Equals(valId) {
		return fmt.Errorf("error attempting to remove construct property: construct value %v does not match property value %v", value, propVal)
	}
	return properties.RemoveProperty(r.Path, value)
}

func (r *ConstructProperty) Details() *property.PropertyDetails {
	return &r.PropertyDetails
}
func (r *ConstructProperty) Clone() property.Property {
	clone := *r
	return &clone
}

func (r *ConstructProperty) GetDefaultValue(ctx property.ExecutionContext, data any) (any, error) {
	if r.DefaultValue == nil {
		return nil, nil
	}
	return r.Parse(r.DefaultValue, ctx, data)
}

func (r *ConstructProperty) Parse(value any, ctx property.ExecutionContext, data any) (any, error) {
	if val, ok := value.(string); ok {
		urn, err := ExecuteUnmarshalAsURN(ctx, val, data)
		if err != nil {
			return nil, fmt.Errorf("invalid construct URN %v", val)
		}
		if !urn.IsResource() || urn.Type != "construct" {
			return nil, fmt.Errorf("invalid construct URN %v", urn)
		}
		if len(r.AllowedTypes) > 0 && !r.AllowedTypes.MatchesAny(urn) {
			return nil, fmt.Errorf("construct value %v does not match allowed types %s", value, r.AllowedTypes)
		}
		return urn, err
	}

	if val, ok := value.(map[string]interface{}); ok {
		id := model.URN{
			AccountID:        val["account"].(string),
			Project:          val["project"].(string),
			Environment:      val["environment"].(string),
			Application:      val["application"].(string),
			Type:             val["type"].(string),
			Subtype:          val["subtype"].(string),
			ParentResourceID: val["parentResourceId"].(string),
			ResourceID:       val["resourceId"].(string),
		}

		if len(r.AllowedTypes) > 0 && !r.AllowedTypes.MatchesAny(id) {
			return nil, fmt.Errorf("construct value %v does not match type %s", value, r.AllowedTypes)
		}
		return id, nil
	}
	if val, ok := value.(model.URN); ok {
		if len(r.AllowedTypes) > 0 && !r.AllowedTypes.MatchesAny(val) {
			return nil, fmt.Errorf("construct value %v does not match type %s", value, r.AllowedTypes)
		}
		return val, nil
	}

	return nil, fmt.Errorf("invalid construct value %v", value)
}

func (r *ConstructProperty) ZeroValue() any {
	return model.URN{}
}

func (r *ConstructProperty) Contains(value any, contains any) bool {
	if val, ok := value.(model.URN); ok {
		if cont, ok := contains.(model.URN); ok {
			return val.Equals(cont)
		}
	}
	return false
}

func (r *ConstructProperty) Type() string {
	if len(r.AllowedTypes) > 0 {
		typeString := ""
		for i, t := range r.AllowedTypes {
			typeString += t.String()
			if i < len(r.AllowedTypes)-1 {
				typeString += ", "
			}
		}
		return fmt.Sprintf("construct(%s)", typeString)
	}
	return "construct"
}

func (r *ConstructProperty) Validate(properties construct.Properties, value any) error {
	if value == nil {
		if r.Required {
			return fmt.Errorf(property.ErrRequiredProperty, r.Path)
		}
		return nil
	}
	id, ok := value.(model.URN)
	if !ok {
		return fmt.Errorf("invalid construct URN %v", value)
	}
	if len(r.AllowedTypes) > 0 && !r.AllowedTypes.MatchesAny(id) {
		return fmt.Errorf("value %v does not match allowed types %s", value, r.AllowedTypes)
	}
	return nil
}

func (r *ConstructProperty) SubProperties() property.PropertyMap {
	return nil
}
