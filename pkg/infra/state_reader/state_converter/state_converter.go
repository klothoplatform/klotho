package stateconverter

import (
	"io"

	"github.com/iancoleman/strcase"
	"github.com/klothoplatform/klotho/pkg/construct"
	statetemplate "github.com/klothoplatform/klotho/pkg/infra/state_reader/state_template"
)

//go:generate mockgen -source=./state_converter.go --destination=../state_converter_mock_test.go --package=statereader

type (
	State map[construct.ResourceId]construct.Properties

	StateConverter interface {
		// ConvertState converts the state to the Klotho state
		ConvertState(io.Reader) (State, error)
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
			resultingProperties := convertKeysToCamelCase(v)
			// convert properties to map[string]any
			mapResult := make(map[string]interface{})
			for k, v := range resultingProperties {
				mapResult[k] = v
			}
			result[camelCaseKey] = mapResult
		default:
			result[camelCaseKey] = v
		}
	}
	return result
}
