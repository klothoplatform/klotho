package stateconverter

import (
	"encoding/json"
	"errors"
	"io"
	"slices"
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

func (p pulumiStateConverter) ConvertResource(resource Resource) (
	*construct.Resource,
	error,
) {
	mapping, ok := p.templates[resource.Type]
	if !ok {
		zap.S().Debugf("no mapping found for resource type %s", resource.Type)
		return nil, nil
	}
	return p.convertResource(resource, mapping)
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
			properties[mapping] = v
		}
	}

	//TODO: find a better way to handle subnet types
	if id.QualifiedTypeName() == "aws:subnet" {
		if rawCidr, ok := properties["CidrBlock"]; ok {
			if cidr, ok := rawCidr.(string); ok && slices.Contains([]string{"10.0.0.0/18", "10.0.64.0/18"}, cidr) {
				properties["Type"] = "public"
			} else {
				properties["Type"] = "private"
			}
		}
	}

	// Convert the keys to camel case
	klothoResource := &construct.Resource{
		ID:         id,
		Properties: properties,
	}
	return klothoResource, nil
}
