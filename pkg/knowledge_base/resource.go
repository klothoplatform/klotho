package knowledgebase

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/construct"
)

type (
	// ResourceTemplate defines how rules are handled by the engine in terms of making sure they are functional in the graph
	ResourceTemplate struct {
		// Provider refers to the resources provider
		Provider string `json:"provider" yaml:"provider"`
		// Type refers to the unique type identifier of the resource
		Type string `json:"type" yaml:"type"`
		// Rules defines a set of rules that must pass checks and actions which must be carried out to make a resource operational
		Rules []OperationalRule `json:"rules" yaml:"rules"`
		// Configuration specifies how to act on any intrinsic values of a resource to make it operational
		Configuration []Configuration `json:"configuration" yaml:"configuration"`
		// SanitizationRules defines a set of rules that are used to ensure the resource's name is valid
		NameSanitization Sanitization `json:"sanitization" yaml:"sanitization"`
		// DeleteContext defines the context in which a resource can be deleted
		DeleteContext construct.DeleteContext `json:"delete_context" yaml:"delete_context"`
		// Views defines the views that the resource should be added to as a distinct node
		Views map[string]string `json:"views" yaml:"views"`
	}

	// OperationalRule defines a rule that must pass checks and actions which must be carried out to make a resource operational
	OperationalRule struct {
		// Enforcement defines how the rule should be enforced
		Enforcement OperationEnforcement `json:"enforcement" yaml:"enforcement"`
		// Direction defines the direction of the rule. The direction options are upstream or downstream
		Direction Direction `json:"direction" yaml:"direction"`
		// ResourceTypes defines the resource types that the rule should be enforced on. Resource types must be specified if classifications is not specified
		ResourceTypes []string `json:"resource_types" yaml:"resource_types"`
		// Classifications defines the classifications that the rule should be enforced on. Classifications must be specified if resource types is not specified
		Classifications []string `json:"classifications" yaml:"classifications"`
		// SetField defines the field on the resource that should be set to the resource that satisfies the rule
		SetField string `json:"set_field" yaml:"set_field"`
		// RemoveDirectDependency defines if the direct dependency between the resource and the rule's resource(s) that satisfies the rule should be removed.
		// We also use this flag to determine if we are retrieving direct dependencies or all dependencies in the specified direction, when looking for rule satisfaction.
		RemoveDirectDependency bool `json:"remove_direct_dependency" yaml:"remove_direct_dependency"`
		// NumNeeded defines the number of resources that must satisfy the rule
		NumNeeded int `json:"num_needed" yaml:"num_needed"`
		// Rules defines a set of sub rules that will be carried out based on the evaluation of the initial parent rule
		Rules []OperationalRule `json:"rules" yaml:"rules"`
		// UnsatisfiedAction defines what action should be taken if the rule is not satisfied
		UnsatisfiedAction UnsatisfiedAction `json:"unsatisfied_action" yaml:"unsatisfied_action"`

		// NoParentDependency is a flag to signal if a sub rule is not supposed to add a dependency on the resource that satisfies the parent rule
		NoParentDependency bool `json:"no_parent_dependency" yaml:"no_parent_dependency"`
	}

	// UnsatisfiedAction defines what action should be taken if the rule is not satisfied
	UnsatisfiedAction struct {
		// Operation defines what action should be taken if the rule is not satisfied
		Operation UnsatisfiedActionOperation `json:"operation" yaml:"operation"`
		// DefaultType defines the default type of resource that should be acted upon if the rule is not satisfied
		DefaultType string `json:"default_type" yaml:"default_type"`
		// Unique defines if the resource that is created should be unique
		Unique bool `json:"unique" yaml:"unique"`
	}

	// Configuration defines how to act on any intrinsic values of a resource to make it operational
	Configuration struct {
		// Fields defines a field that should be set on the resource
		Field string `json:"field" yaml:"field"`
		// Value defines the value that should be set on the resource
		Value any `json:"value" yaml:"value"`
		// ZeroValueAllowed defines if the value can be set to the zero value of the field
		ZeroValueAllowed bool `json:"zero_value_allowed" yaml:"zero_value_allowed"`
	}

	Sanitization struct {
		Rules     []SanitizationRule `json:"rules" yaml:"rules"`
		MaxLength int                `json:"max_length" yaml:"max_length"`
		MinLength int                `json:"min_length" yaml:"min_length"`
	}
	SanitizationRule struct {
		Pattern     string `json:"pattern" yaml:"pattern"`
		Replacement string `json:"replacement" yaml:"replacement"`
	}

	// OperationEnforcement defines how the rule should be enforced
	OperationEnforcement string
	// UnsatisfiedActionOperation defines what action should be taken if the rule is not satisfied
	UnsatisfiedActionOperation string
	// Direction defines the direction of the rule. The direction options are upstream or downstream
	Direction string
)

const (
	// ExactlyOne defines that the rule should be enforced on exactly one resource
	ExactlyOne OperationEnforcement = "exactly_one"
	// Conditional defines that the rule should be enforced on a resource if it exists
	Conditional OperationEnforcement = "conditional"
	// AnyAvailable defines that the rule should be enforced on any available resource
	AnyAvailable OperationEnforcement = "any_available"

	// CreateUnsatisfiedResource defines that a resource should be created if the rule is not satisfied
	CreateUnsatisfiedResource UnsatisfiedActionOperation = "create"
	// ErrorUnsatisfiedResource defines that an error should be returned if the rule is not satisfied
	ErrorUnsatisfiedResource UnsatisfiedActionOperation = "error"

	Upstream   Direction = "upstream"
	Downstream Direction = "downstream"
)

func (or *OperationalRule) String() string {
	if or.ResourceTypes != nil {
		return fmt.Sprintf("%s %s", or.Enforcement, or.ResourceTypes)
	} else if or.Classifications != nil {
		return fmt.Sprintf("%s %s", or.Enforcement, or.Classifications)
	}
	return string(or.Enforcement)
}
