package knowledgebase2

import (
	"errors"
	"fmt"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"gopkg.in/yaml.v3"
)

type (
	OperationalRule struct {
		If                 string              `json:"if" yaml:"if"`
		Steps              []OperationalStep   `json:"steps" yaml:"steps"`
		ConfigurationRules []ConfigurationRule `json:"configuration_rules" yaml:"configuration_rules"`
	}

	// OperationalRule defines a rule that must pass checks and actions which must be carried out to make a resource operational
	OperationalStep struct {
		Resource string `json:"resource" yaml:"resource"`
		// Direction defines the direction of the rule. The direction options are upstream or downstream
		Direction Direction `json:"direction" yaml:"direction"`
		// Resources defines the resource types that the rule should be enforced on. Resource types must be specified if classifications is not specified
		Resources []ResourceSelector `json:"resources" yaml:"resources"`
		// NumNeeded defines the number of resources that must satisfy the rule
		NumNeeded int `json:"num_needed" yaml:"num_needed"`

		ReplacementCondition string `json:"replacement_condition" yaml:"replacement_condition"`

		FailIfMissing bool `json:"fail_if_missing" yaml:"fail_if_missing"`
		// Unique defines if the resource that is created should be unique
		Unique bool `json:"unique" yaml:"unique"`
		// SelectionOperator defines how the rule should select a resource if one does not exist
		SelectionOperator SelectionOperator
	}

	ConfigurationRule struct {
		Resource string        `json:"resource" yaml:"resource"`
		Config   Configuration `json:"configuration" yaml:"configuration"`
	}

	// Configuration defines how to act on any intrinsic values of a resource to make it operational
	Configuration struct {
		// Fields defines a field that should be set on the resource
		Field string `json:"field" yaml:"field"`
		// Value defines the value that should be set on the resource
		Value any `json:"value" yaml:"value"`
	}

	ResourceSelector struct {
		Selector   string         `json:"selector" yaml:"selector"`
		Properties map[string]any `json:"properties" yaml:"properties"`
		// Classifications defines the classifications that the rule should be enforced on. Classifications must be specified if resource types is not specified
		Classifications []string `json:"classifications" yaml:"classifications"`
	}

	// Direction defines the direction of the rule. The direction options are upstream or downstream
	Direction string

	SelectionOperator string
)

const (
	DirectionUpstream   Direction = "upstream"
	DirectionDownstream Direction = "downstream"

	SpreadSelectionOperator  SelectionOperator = "spread"
	ClusterSelectionOperator SelectionOperator = "cluster"
	ClosestSelectionOperator SelectionOperator = ""
)

func (p ResourceSelector) IsMatch(decodedSelector construct.ResourceId, res *construct.Resource, kb TemplateKB) bool {
	if decodedSelector != (construct.ResourceId{}) {
		if !decodedSelector.Matches(res.ID) {
			return false
		}
	}
	for k, v := range p.Properties {
		property, err := res.GetProperty(k)
		if err != nil {
			return false
		}
		if property != v {
			return false
		}
		template, err := kb.GetResourceTemplate(res.ID)
		if err != nil || template == nil {
			return false
		}
		if !template.ResourceContainsClassifications(p.Classifications) {
			return false
		}
	}
	return true
}

func (p *ResourceSelector) UnmarshalYAML(n *yaml.Node) error {
	type h ResourceSelector
	var r h
	err := n.Decode(&r)
	if err != nil {
		var selectorString string
		err = n.Decode(&selectorString)
		if err == nil {
			r.Selector = selectorString
		} else {
			return fmt.Errorf("error decoding resource selector: %w", err)
		}
	}
	*p = ResourceSelector(r)
	return nil
}

func (step OperationalStep) ExtractResourcesAndTypes(
	ctx DynamicValueContext,
	data DynamicValueData) (resources []construct.ResourceId, resource_types []construct.ResourceId, errs error) {
	for _, resStr := range step.Resources {
		var selectors construct.ResourceList
		selector, err := ctx.ExecuteDecodeAsResourceId(resStr.Selector, data)
		if err != nil {
			// The output of the decode may be a list of resources, so attempt to parse to that
			err = ctx.ExecuteDecode(resStr.Selector, data, &selectors)
			if err != nil {
				errs = errors.Join(errs, err)
				continue
			}
		} else {
			selectors = append(selectors, selector)
		}

		for _, id := range selectors {
			if id.Name != "" {
				resources = append(resources, id)
			} else {
				resource_types = append(resource_types, id)
			}
		}
	}
	return
}
