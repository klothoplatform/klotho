package stateconverter

import (
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/klothoplatform/klotho/pkg/construct"
	statetemplate "github.com/klothoplatform/klotho/pkg/infra/state_reader/state_template"
	"go.uber.org/zap"
)

type (
	PulumiState []Resource

	Resource struct {
		Urn     string                 `json:"urn"`
		Type    string                 `json:"type"`
		Outputs map[string]interface{} `json:"outputs"`
	}
	pulumiStateConverter struct {
		templates map[string]statetemplate.StateTemplate
	}
)

func (p pulumiStateConverter) ConvertState(reader io.Reader) (State, error) {
	var pulumiState PulumiState
	dec := json.NewDecoder(reader)
	err := dec.Decode(&pulumiState)
	if err != nil {
		return nil, err
	}
	internalModel := make(State)
	var errs error
	// Convert the Pulumi state to the internal model
	for _, resource := range pulumiState {
		mapping, ok := p.templates[resource.Type]
		if !ok {
			zap.S().Debugf("no mapping found for resource type %s", resource.Type)
			continue
		}
		resource, err := p.convertResource(resource, mapping)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		internalModel[resource.ID] = resource.Properties
	}
	return internalModel, errs
}

func (p pulumiStateConverter) convertResource(resource Resource, template statetemplate.StateTemplate) (
	*construct.Resource,
	error,
) {

	// Get the type from the resource
	parts := strings.Split(resource.Urn, ":")
	name := parts[len(parts)-1]
	id := construct.ResourceId{
		Provider: strings.Split(template.QualifiedTypeName, ":")[0],
		Type:     strings.Split(template.QualifiedTypeName, ":")[1],
		Name:     name,
	}
	properties := make(construct.Properties)
	for k, v := range resource.Outputs {
		if mapping, ok := template.PropertyMappings[k]; ok {
			if strings.Contains(mapping, "#") {
				// TODO: Determine how to cross correlate references/resource properties.
				// an example of this is the subnets vpcId field (value = vpc-123456789), to where internally its modeld as a "resource".
				continue
			}
			properties[mapping] = v
		}
	}
	// Convert the keys to camel case
	klothoResource := &construct.Resource{
		ID:         id,
		Properties: convertKeysToCamelCase(properties),
	}
	return klothoResource, nil
}
