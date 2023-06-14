package constraints

import (
	"errors"

	"github.com/klothoplatform/klotho/pkg/core"
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
		Operator   ConstraintOperator `yaml:"operator"`
		Target     core.ResourceId    `yaml:"target"`
		Type       string             `yaml:"type"`
		Attributes map[string]any     `yaml:"attributes"`
	}
)

func (constraint *ConstructConstraint) Scope() ConstraintScope {
	return ConstructConstraintScope
}

func (constraint *ConstructConstraint) IsSatisfied(dag *core.ResourceGraph, kb knowledgebase.EdgeKB, mappedConstructResources map[core.ResourceId][]core.Resource) bool {
	switch constraint.Operator {
	case EqualsConstraintOperator:
		// Well look at all resources to see if there is a resource matching the type, that references the base construct passed in
		// Cuirrently attributes go unchecked
		if constraint.Type != "" {
			for _, res := range dag.ListResources() {
				if res.Id().Type == constraint.Type && res.BaseConstructsRef().Has(constraint.Target) {
					return true
				}
			}
		}
	}
	return false
}

func (constraint *ConstructConstraint) Validate() error {
	if constraint.Target.Provider != core.AbstractConstructProvider {
		return errors.New("node constraint must be applied to an abstract construct")
	}
	return nil
}
