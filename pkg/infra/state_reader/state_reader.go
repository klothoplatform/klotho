package statereader

import (
	"errors"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	stateconverter "github.com/klothoplatform/klotho/pkg/infra/state_reader/state_converter"
	statetemplate "github.com/klothoplatform/klotho/pkg/infra/state_reader/state_template"
	"github.com/klothoplatform/klotho/pkg/knowledgebase"
)

type (
	DriftError struct {
		Err      error
		ref      construct.PropertyRef
		oldValue interface{}
		newValue interface{}
	}

	// StateReader is an interface for reading state from a state store
	StateReader interface {
		// ReadState reads the state from the state store
		ReadState([]byte, construct.Graph) (construct.Graph, error)

		// DetectDrift detects drift between the state and the IaC
		DetectDrift([]byte, construct.Graph) []DriftError

		// ConvertToImports converts the state to imports
		ConvertToImports([]*construct.Resource) []constraints.Constraint
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
	if graph != nil {
		driftErrors := p.DetectDrift(state, graph)
		if len(driftErrors) > 0 {
			var joinedDriftErrors error
			for _, err := range driftErrors {
				joinedDriftErrors = errors.Join(joinedDriftErrors, err.Err)
			}
			return nil, joinedDriftErrors
		}
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
		returnGraph.AddVertex(resource)
	}
	return returnGraph, nil
}

func (p stateReader) DetectDrift(state []byte, graph construct.Graph) []DriftError {
	return nil
}

func (p stateReader) ConvertToImports(resources []*construct.Resource) []constraints.Constraint {
	importConstraints := make([]constraints.Constraint, 0)
	for _, res := range resources {
		// Convert the resource to an import
		importConstraints = append(importConstraints, &constraints.ApplicationConstraint{
			Operator: constraints.ImportConstraintOperator,
			Node:     res.ID,
		})
		for key, value := range res.Properties {
			importConstraints = append(importConstraints, &constraints.ResourceConstraint{
				Operator: constraints.ImportConstraintOperator,
				Target:   res.ID,
				Property: key,
				Value:    value,
			})
		}
	}
	return importConstraints
}
