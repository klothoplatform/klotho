package statereader

import (
	"github.com/klothoplatform/klotho/pkg/construct"
	stateconverter "github.com/klothoplatform/klotho/pkg/infra/state_reader/state_converter"
	statetemplate "github.com/klothoplatform/klotho/pkg/infra/state_reader/state_template"
	"github.com/klothoplatform/klotho/pkg/knowledgebase"
)

type (
	// StateReader is an interface for reading state from a state store
	StateReader interface {
		// ReadState reads the state from the state store
		ReadState([]byte, construct.Graph) (construct.Graph, error)
	}

	stateReader struct {
		templates map[string]statetemplate.StateTemplate
		kb        knowledgebase.TemplateKB
		converter stateconverter.StateConverter
	}
)

func NewPulumiReader(templates map[string]statetemplate.StateTemplate, kb knowledgebase.TemplateKB) StateReader {
	return &stateReader{templates: templates, kb: kb, converter: stateconverter.NewStateConverter("pulumi", templates)}
}

func (p stateReader) ReadState(state []byte, graph construct.Graph) (construct.Graph, error) {
	returnGraph := construct.NewGraph()
	internalState, err := p.converter.ConvertState(state)
	if err != nil {
		return nil, err
	}
	for id, properties := range internalState {
		var resource *construct.Resource
		if graph != nil {
			resource, err = graph.Vertex(id)
			if err != nil {
				return nil, err
			}
		}
		if resource == nil {
			resource = &construct.Resource{
				ID:         id,
				Properties: make(construct.Properties),
			}
		}
		for key, value := range properties {
			err := resource.SetProperty(key, value)
			if err != nil {
				return nil, err
			}
		}
		err = returnGraph.AddVertex(resource)
		if err != nil {
			return nil, err
		}
	}
	return returnGraph, nil
}
