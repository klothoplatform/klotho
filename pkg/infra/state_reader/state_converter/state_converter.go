package stateconverter

import (
	"github.com/iancoleman/strcase"
	"github.com/klothoplatform/klotho/pkg/construct"
	statetemplate "github.com/klothoplatform/klotho/pkg/infra/state_reader/state_template"
)

type (
	State map[construct.ResourceId]construct.Properties

	StateConverter interface {
		// ConvertState converts the state to the Klotho state
		ConvertState([]byte) (State, error)
	}
)

func NewStateConverter(provider string, templates map[string]statetemplate.StateTemplate) StateConverter {
	return &pulumiStateConverter{templates: templates}
}

func convertKeysToCamelCase(data construct.Properties) construct.Properties {
	result := make(map[string]interface{})
	for key, value := range data {
		camelCaseKey := strcase.ToCamel(key)
		switch v := value.(type) {
		case map[string]interface{}:
			result[camelCaseKey] = convertKeysToCamelCase(v)
		default:
			result[camelCaseKey] = v
		}
	}
	return result
}
