package statetemplate

import (
	"embed"

	"gopkg.in/yaml.v3"
)

type (
	// StateTemplate is a template for reading state from a state store
	StateTemplate struct {
		// QualifiedTypeName is the qualified type name of the resource
		QualifiedTypeName string `json:"qualified_type_name" yaml:"qualified_type_name"`
		// IaCQualifiedType is the qualified type of the IaC resource
		IaCQualifiedType string `json:"iac_qualified_type" yaml:"iac_qualified_type"`
		// PropertyMappings is a map of property mappings
		PropertyMappings map[string]string `json:"property_mappings" yaml:"property_mappings"`
	}
)

//go:embed mappings/*/*.yaml
var PulumiTemplates embed.FS

func LoadStateTemplates(provider string) (map[string]StateTemplate, error) {
	stateTemplates := make(map[string]StateTemplate)
	files, err := PulumiTemplates.ReadDir("mappings/" + provider)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		data, err := PulumiTemplates.ReadFile("mappings/" + provider + "/" + file.Name())
		if err != nil {
			return nil, err
		}
		var stateTemplate StateTemplate
		err = yaml.Unmarshal(data, &stateTemplate)
		if err != nil {
			return nil, err
		}
		stateTemplates[stateTemplate.IaCQualifiedType] = stateTemplate
	}
	return stateTemplates, nil
}
