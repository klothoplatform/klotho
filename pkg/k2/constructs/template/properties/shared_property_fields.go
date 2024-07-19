package properties

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/klothoplatform/klotho/pkg/construct"
)

type (
	SharedPropertyFields struct {
		DefaultValue   any `json:"default_value" yaml:"default_value"`
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
