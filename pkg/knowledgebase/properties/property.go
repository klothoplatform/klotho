package properties

import (
	"bytes"
	"fmt"
	"text/template"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledgebase"
)

type (
	SharedPropertyFields struct {
		DefaultValue   any
		ValidityChecks []PropertyValidityCheck
	}

	PropertyValidityCheck struct {
		template *template.Template
	}
	ValidityCheckData struct {
		Properties construct.Properties `json:"properties" yaml:"properties"`
		Value      any                  `json:"value" yaml:"value"`
	}
)

func ParsePropertyRef(value any, ctx knowledgebase.DynamicContext, data knowledgebase.DynamicValueData) (construct.PropertyRef, error) {
	if val, ok := value.(string); ok {
		result := construct.PropertyRef{}
		err := ctx.ExecuteDecode(val, data, &result)
		return result, err
	}
	if val, ok := value.(map[string]interface{}); ok {
		rp := ResourceProperty{}
		id, err := rp.Parse(val["resource"], ctx, data)
		if err != nil {
			return construct.PropertyRef{}, err
		}
		return construct.PropertyRef{
			Property: val["property"].(string),
			Resource: id.(construct.ResourceId),
		}, nil
	}
	if val, ok := value.(construct.PropertyRef); ok {
		return val, nil
	}
	return construct.PropertyRef{}, fmt.Errorf("invalid property reference value %v", value)
}

func ValidatePropertyRef(value construct.PropertyRef, propertyType string, ctx knowledgebase.DynamicContext) (refVal any, err error) {
	resource, err := ctx.DAG().Vertex(value.Resource)
	if err != nil {
		return nil, fmt.Errorf("error getting resource %s, while validating property ref: %w", value.Resource, err)
	}
	if resource == nil {
		return nil, fmt.Errorf("resource %s does not exist", value.Resource)
	}
	rt, err := ctx.KB().GetResourceTemplate(value.Resource)
	if err != nil {
		return nil, err
	}
	prop := rt.GetProperty(value.Property)
	if prop == nil {
		return nil, fmt.Errorf("property %s does not exist on resource %s", value.Property, value.Resource)
	}
	if prop.Type() != propertyType {
		return nil, fmt.Errorf("property %s on resource %s is not of type %s", value.Property, value.Resource, propertyType)
	}
	if prop.Details().DeployTime {
		return nil, nil
	}
	propVal, err := resource.GetProperty(value.Property)
	if err != nil {
		return nil, fmt.Errorf("error getting property %s on resource %s, while validating property ref: %w", value.Property, value.Resource, err)
	}

	// recurse down in case of a nested property ref
	for propValRef, ok := propVal.(construct.PropertyRef); ok; propValRef, ok = propVal.(construct.PropertyRef) {
		propVal, err = ValidatePropertyRef(propValRef, propertyType, ctx)
		if err != nil {
			return nil, err
		}
		if propVal == nil {
			return nil, nil
		}
	}

	return propVal, nil
}

func (p *PropertyValidityCheck) Validate(value any, properties construct.Properties) error {
	var buff bytes.Buffer
	data := ValidityCheckData{
		Properties: properties,
		Value:      value,
	}
	err := p.template.Execute(&buff, data)
	if err != nil {
		return err
	}
	result := buff.String()
	if result != "" {
		return fmt.Errorf("invalid value %v: %s", value, result)
	}
	return nil
}
