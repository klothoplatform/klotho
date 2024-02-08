package knowledgebase

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"gopkg.in/yaml.v3"
)

type (
	OperationalRule struct {
		If                 string              `json:"if" yaml:"if"`
		Steps              []OperationalStep   `json:"steps" yaml:"steps"`
		ConfigurationRules []ConfigurationRule `json:"configuration_rules" yaml:"configuration_rules"`
	}

	EdgeRule struct {
		If                 string                `json:"if" yaml:"if"`
		Steps              []EdgeOperationalStep `json:"steps" yaml:"steps"`
		ConfigurationRules []ConfigurationRule   `json:"configuration_rules" yaml:"configuration_rules"`
	}

	AdditionalRule struct {
		If    string            `json:"if" yaml:"if"`
		Steps []OperationalStep `json:"steps" yaml:"steps"`
	}

	PropertyRule struct {
		If    string          `json:"if" yaml:"if"`
		Step  OperationalStep `json:"step" yaml:"step"`
		Value any             `json:"value" yaml:"value"`
	}

	EdgeOperationalStep struct {
		Resource string `json:"resource" yaml:"resource"`
		OperationalStep
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
		// FailIfMissing fails if the step is not satisfied when being evaluated. If this flag is set, the step cannot create dependencies
		FailIfMissing bool `json:"fail_if_missing" yaml:"fail_if_missing"`
		// Unique defines if the resource that is created should be unique
		Unique bool `json:"unique" yaml:"unique"`
		// UseRef defines if the rule should set the field to the property reference instead of the resource itself
		UsePropertyRef string `json:"use_property_ref" yaml:"use_property_ref"`
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
		// Value defines the value that should be set on the resource
		Value any `json:"value" yaml:"value"`
	}

	ResourceSelector struct {
		Selector   string         `json:"selector" yaml:"selector"`
		Properties map[string]any `json:"properties" yaml:"properties"`
		// NumPreferred defines the amount of resources that should be preferred to satisfy the selector.
		// This number is only used if num needed on the step is not met
		NumPreferred int `json:"num_preferred" yaml:"num_preferred"`
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

func (rule AdditionalRule) Hash() (string, error) {
	// Convert the struct to a byte slice.
	// Note that the struct must be able to be converted to JSON,
	// so all fields must be exported (i.e., start with a capital letter).
	byteSlice, err := json.Marshal(rule)
	if err != nil {
		return "", err
	}

	// Hash the byte slice.
	hash := sha256.Sum256(byteSlice)

	// Convert the hash to a hexadecimal string.
	hashString := hex.EncodeToString(hash[:])

	return hashString, nil
}

func (d Direction) Edge(resource, dep construct.ResourceId) construct.SimpleEdge {
	if d == DirectionUpstream {
		return construct.SimpleEdge{Source: dep, Target: resource}
	}
	return construct.SimpleEdge{Source: resource, Target: dep}
}

// IsMatch checks if the resource selector is a match for the given resource
func (p ResourceSelector) IsMatch(ctx DynamicValueContext, data DynamicValueData, res *construct.Resource) (bool, error) {
	return p.matches(ctx, data, res, false)
}

// CanUse checks if the `res` can be used to satisfy the resource selector. This differs from [IsMatch] because it will
// also consider unset properties to be able to be used. This is primarily used for when empty resources are created
// during path expansion, other resources' selectors can be used to configure those empty resources.
func (p ResourceSelector) CanUse(ctx DynamicValueContext, data DynamicValueData, res *construct.Resource) (bool, error) {
	return p.matches(ctx, data, res, true)
}

func (p ResourceSelector) matches(
	ctx DynamicValueContext,
	data DynamicValueData,
	res *construct.Resource,
	allowEmpty bool,
) (bool, error) {
	ids, err := p.ExtractResourceIds(ctx, data)
	if err != nil {
		return false, fmt.Errorf("error extracting resource ids in resource selector: %w", err)
	}
	matchesType := false
	for _, id := range ids {
		// We only check if the resource selector is a match in terms of properties and classifications (not the actual id)
		// We do this because if we have explicit ids in the selector and someone changes the id of a side effect resource
		// we would no longer think it is a side effect since the id would no longer match.
		// To combat this we just check against type
		sel := construct.ResourceId{Provider: id.Provider, Type: id.Type}
		if sel.Matches(res.ID) {
			matchesType = true
			break
		}
	}
	if !matchesType {
		return false, nil
	}

	template, err := ctx.KB().GetResourceTemplate(res.ID)
	if err != nil {
		return false, fmt.Errorf("error getting resource template in resource selector: %w", err)
	}
	for k, v := range p.Properties {
		property, err := res.GetProperty(k)
		if err != nil {
			return false, err
		}
		selectorPropertyVal, err := TransformToPropertyValue(res.ID, k, v, ctx, data)
		if err != nil {
			return false, fmt.Errorf("error transforming property value in resource selector: %w", err)
		}
		if !reflect.DeepEqual(property, selectorPropertyVal) {
			if !(allowEmpty && property == nil) {
				return false, nil
			}
		}
		if !template.ResourceContainsClassifications(p.Classifications) {
			return false, nil
		}
	}
	return true, nil
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
		err := ctx.ExecuteDecode(p.Selector, data, &selectors)
		if err != nil {
			errs = errors.Join(errs, err)
			if errs != nil {
				return nil, errs
			}
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
