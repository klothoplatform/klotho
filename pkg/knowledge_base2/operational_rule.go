package knowledgebase2

import (
	"errors"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
)

type (
	OperationalRule struct {
		If                 string               `json:"if" yaml:"if"`
		Steps              []*OperationalStep   `json:"steps" yaml:"steps"`
		ConfigurationRules []*ConfigurationRule `json:"configuration_rules" yaml:"configuration_rules"`
	}

	// OperationalRule defines a rule that must pass checks and actions which must be carried out to make a resource operational
	OperationalStep struct {
		Resource string `json:"resource" yaml:"resource"`
		// Direction defines the direction of the rule. The direction options are upstream or downstream
		Direction Direction `json:"direction" yaml:"direction"`
		// Resources defines the resource types that the rule should be enforced on. Resource types must be specified if classifications is not specified
		Resources []string `json:"resources" yaml:"resources"`
		// Classifications defines the classifications that the rule should be enforced on. Classifications must be specified if resource types is not specified
		Classifications []string `json:"classifications" yaml:"classifications"`
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

	// Direction defines the direction of the rule. The direction options are upstream or downstream
	Direction string

	SelectionOperator string
)

const (
	Upstream   Direction = "upstream"
	Downstream Direction = "downstream"

	SpreadSelectionOperator  SelectionOperator = "spread"
	ClusterSelectionOperator SelectionOperator = "cluster"
	ClosestSelectionOperator SelectionOperator = "closest"
)

func (step OperationalStep) ExtractResourcesAndTypes(ctx ConfigTemplateContext, data ConfigTemplateData) (
	resources []construct.ResourceId,
	resource_types []construct.ResourceId,
	errs error) {
	for _, resStr := range step.Resources {
		var selectors construct.ResourceList
		selector, err := ctx.ExecuteDecodeAsResourceId(resStr, data)
		if err != nil {
			// The output of the decode may be a list of resources, so attempt to parse to that
			err = ctx.ExecuteDecode(resStr, data, &selectors)
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
