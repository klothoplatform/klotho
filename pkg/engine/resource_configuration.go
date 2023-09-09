package engine

import (
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
)

func (e *Engine) ConfigureResource(context *SolveContext, resource construct.Resource) error {
	template := e.ResourceTemplates[construct.ResourceId{Provider: resource.Id().Provider, Type: resource.Id().Type}]
	if template != nil {
		resourceConstraints := make(map[string]*constraints.ResourceConstraint)
		for _, c := range e.Context.Constraints[constraints.ResourceConstraintScope] {
			constraint := c.(*constraints.ResourceConstraint)
			if constraint.Target == resource.Id() {
				resourceConstraints[constraint.Property] = constraint
			}
		}

		for _, config := range template.Configuration {
			constraint := resourceConstraints[config.Field]
			e.handleDecision(context, Decision{
				Level: LevelInfo,
				Result: &DecisionResult{
					Resource: resource,
					Config:   config,
				},
				Action: ActionConfigure,
				Cause:  &Cause{Constraint: constraint},
			})
		}
	}

	// TODO remove this once the configurations have moved into the resource templates
	err := context.ResourceGraph.CallConfigure(resource, nil)
	if err != nil {
		return err
	}

	return nil
}
