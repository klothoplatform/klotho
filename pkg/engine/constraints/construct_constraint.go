package constraints

import (
	"errors"
	"fmt"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/classification"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
)

type (
	// ConstructConstraint is a struct that represents constraints that can be applied on a specific construct in the resource graph
	//
	// Example
	//
	// To specify a constraint detailing construct expansion and configuration in yaml
	//
	// - scope: construct
	// operator: equals
	// target: klotho:orm:my_orm
	// type: rds_instance
	//
	// The end result of this should be that the orm construct is expanded into an rds instance + necessary resources
	ConstructConstraint struct {
		Operator   ConstraintOperator   `yaml:"operator"`
		Target     construct.ResourceId `yaml:"target"`
		Type       string               `yaml:"type"`
		Attributes map[string]any       `yaml:"attributes"`
	}
)

func (constraint *ConstructConstraint) Scope() ConstraintScope {
	return ConstructConstraintScope
}

func (constraint *ConstructConstraint) IsSatisfied(dag *construct.ResourceGraph, kb knowledgebase.EdgeKB, mappedConstructResources map[construct.ResourceId][]construct.Resource, classifier *classification.ClassificationDocument) bool {
	switch constraint.Operator {
	case EqualsConstraintOperator:
		// Well look at all resources to see if there is a resource matching the type, that references the base construct passed in
		// Cuirrently attributes go unchecked
		for _, res := range dag.ListResources() {
			if constraint.Type != "" && res.Id().Type == constraint.Type && res.BaseConstructRefs().Has(constraint.Target) {
				return true
			} else if constraint.Type == "" && res.BaseConstructRefs().Has(constraint.Target) {
				return true
			}
		}

	}
	return false
}

func (constraint *ConstructConstraint) Validate() error {
	if constraint.Target.Provider != construct.AbstractConstructProvider {
		return errors.New("node constraint must be applied to an abstract construct")
	}
	return nil
}

func (constraint *ConstructConstraint) String() string {
	return fmt.Sprintf("Constraint: %s %s %s", constraint.Scope(), constraint.Operator, constraint.Target)
}
