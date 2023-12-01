package properties

import (
	"bytes"
	"fmt"
	"text/template"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
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

func ParsePropertyRef(value any, ctx knowledgebase.DynamicContext, data knowledgebase.DynamicValueData) (any, error) {
	if val, ok := value.(string); ok {
		result := construct.PropertyRef{}
		err := ctx.ExecuteDecode(val, data, &result)
		return result, err
	}
	if val, ok := value.(map[string]interface{}); ok {
		rp := ResourceProperty{}
		id, err := rp.Parse(val["resource"], ctx, data)
		if err != nil {
			return nil, err
		}
		return construct.PropertyRef{
			Property: val["property"].(string),
			Resource: id.(construct.ResourceId),
		}, nil
	}
	if val, ok := value.(construct.PropertyRef); ok {
		return val, nil
	}
	return nil, fmt.Errorf("invalid property reference value %v", value)
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

func decodeAsPropertyRef(template *template.Template, ctx knowledgebase.DynamicValueContext, data knowledgebase.DynamicValueData) (any, error) {
	var ref construct.PropertyRef
	err := ctx.ExecuteTemplateDecode(template, data, &ref)
	if err != nil {
		return nil, err
	}
	return ref, nil
}
