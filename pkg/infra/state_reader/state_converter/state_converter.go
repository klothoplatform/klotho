package stateconverter

import (
	"io"

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
