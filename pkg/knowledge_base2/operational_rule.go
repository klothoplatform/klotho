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
		SelectionOperator SelectionOperator `json:"selection_operator" yaml:"selection_operator"`
	}

	ConfigurationRule struct {
		Resource string        `json:"resource" yaml:"resource"`
		Config   Configuration `json:"configuration" yaml:"configuration"`
	}

	// Configuration defines how to act on any intrinsic values of a resource to make it operational
	Configuration struct {
		// Field defines a field that should be set on the resource
		Field string `json:"field" yaml:"field"`
		// Fields defines a set of field that should be set on the resource
		Fields []string `json:"fields" yaml:"fields"`
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

// IsMatch checks if the resource selector is a match for the given resource
func (p ResourceSelector) IsMatch(ctx DynamicValueContext, data DynamicValueData, res *construct.Resource) bool {
	ids, err := p.ExtractResourceIds(ctx, data)
	if err != nil {
		return false
	}
	var resourceTypes construct.ResourceList
	for _, id := range ids {
		resourceTypes = append(resourceTypes, construct.ResourceId{Provider: id.Provider, Type: id.Type})
	}

	// We only check if the resource selector is a match in terms of properties and classifications (not the actual id)
	// We do this because if we have explicit ids in the selector and someone changes the id of a side effect resource
	// we would no longer think it is a side effect since the id would no longer match.
	// To combat this we just check against type
	if len(resourceTypes) > 0 {
		matchesType := false
		for _, resourceType := range resourceTypes {
			if resourceType.Matches(res.ID) {
				matchesType = true
			}
		}
		if !matchesType {
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
		template, err := ctx.KB().GetResourceTemplate(res.ID)
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

func (p ResourceSelector) ExtractResourceIds(ctx DynamicValueContext, data DynamicValueData) (ids construct.ResourceList, errs error) {
	var selectors construct.ResourceList
	if p.Selector != "" {
		selector, err := ExecuteDecodeAsResourceId(ctx, p.Selector, data)
		if err != nil {
			// The output of the decode may be a list of resources, so attempt to parse to that
			err = ctx.ExecuteDecode(p.Selector, data, &selectors)
			if err != nil {
				errs = errors.Join(errs, err)
				if errs != nil {
					return nil, errs
				}
			}
		} else {
			selectors = append(selectors, selector)
		}
	} else {
		for _, res := range ctx.KB().ListResources() {
			selectors = append(selectors, res.Id())
		}
	}

	for _, id := range selectors {
		resTmpl, err := ctx.KB().GetResourceTemplate(id)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		if resTmpl == nil {
			errs = errors.Join(errs, fmt.Errorf("could not find resource template for %s", id))
			continue
		}
		if resTmpl.ResourceContainsClassifications(p.Classifications) {
			ids = append(ids, id)
		}
	}
	return
}
