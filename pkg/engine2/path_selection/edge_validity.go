package path_selection

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/dominikbraun/graph"
	"github.com/klothoplatform/klotho/pkg/collectionutil"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

// checkUniquenessValidity checks if the candidate is valid based on if it is intended to be created as unique resource
// for another resource. If the resource was created by an operational rule with the unique flag, we wont consider it as valid
func checkUniquenessValidity(
	ctx solution_context.SolutionContext,
	src, trgt construct.ResourceId,
) (bool, error) {
	// check if the  node is a phantom node
	source, err := ctx.RawView().Vertex(src)
	switch {
	case errors.Is(err, graph.ErrVertexNotFound):
		source = &construct.Resource{ID: src}
	case err != nil:
		return false, err
	}

	target, err := ctx.RawView().Vertex(trgt)
	switch {
	case errors.Is(err, graph.ErrVertexNotFound):
		target = &construct.Resource{ID: trgt}
	case err != nil:
		return false, err
	}

	// check if the upstream resource has a unique rule for the matched resource type
	valid, err := checkProperties(ctx, target, source, knowledgebase.DirectionUpstream)
	if err != nil {
		return false, err
	}
	if !valid {
		return false, nil
	}

	// check if the downstream resource has a unique rule for the matched resource type
	valid, err = checkProperties(ctx, source, target, knowledgebase.DirectionDownstream)
	if err != nil {
		return false, err
	}
	if !valid {
		return false, nil
	}

	return true, nil
}

// check properties checks the resource's properties to make sure its not supposed to have a unique toCheck type
// if it is, it makes sure that the toCheck is not used elsewhere or already being used as its unique type
func checkProperties(ctx solution_context.SolutionContext, resource, toCheck *construct.Resource, direction knowledgebase.Direction) (bool, error) {
	//check if the upstream resource has a unique rule for the matched resource type
	template, err := ctx.KnowledgeBase().GetResourceTemplate(resource.ID)
	if err != nil || template == nil {
		return false, fmt.Errorf("error getting resource template for resource %s: %w", resource.ID, err)
	}

	if strings.Contains(resource.ID.Name, PHANTOM_PREFIX) {
		return true, nil
	}

	explicitlyNotValid := false
	explicitlyValid := false
	err = template.LoopProperties(resource, func(prop knowledgebase.Property) error {
		details := prop.Details()
		rule := details.OperationalRule
		if rule == nil || rule.Step == nil {
			return nil
		}
		step := rule.Step
		if !step.Unique || step.Direction != direction {
			return nil
		}
		//check if the upstream resource is the same type as the matched resource type
		for _, selector := range step.Resources {
			match, err := selector.CanUse(solution_context.DynamicCtx(ctx), knowledgebase.DynamicValueData{Resource: resource.ID},
				toCheck)
			if err != nil {
				return fmt.Errorf("error checking if resource %s matches selector %s: %w", toCheck, selector, err)
			}
			// if its a match for the selectors, lets ensure that it has a dependency and exists in the properties of the rul
			if !match {
				continue
			}
			property, err := resource.GetProperty(details.Path)
			if err != nil {
				return fmt.Errorf("error getting property %s for resource %s: %w", details.Path, toCheck, err)
			}
			if property != nil {
				if checkIfPropertyContainsResource(property, toCheck.ID) {
					explicitlyValid = true
					return knowledgebase.ErrStopWalk
				}
			} else {
				loneDep, err := checkIfLoneDependency(ctx, resource.ID, toCheck.ID, direction, selector)
				if err != nil {
					return err
				}
				if loneDep {
					explicitlyValid = true
					return knowledgebase.ErrStopWalk
				}
			}
			explicitlyNotValid = true
			return knowledgebase.ErrStopWalk
		}

		return nil
	})
	if err != nil {
		return false, err
	}
	if explicitlyValid {
		return true, nil
	} else if explicitlyNotValid {
		return false, nil
	}

	// if we cant validate uniqueness off of properties we then need to see if the resource was created to be unique
	// check if the upstream resource was created as a unique resource by any of its direct dependents
	valid, err := checkIfCreatedAsUniqueValidity(ctx, resource, toCheck, direction)
	if err != nil {
		return false, err
	}
	if !valid {
		return false, nil
	}
	return true, nil
}

// checkIfPropertyContainsResource checks if the property contains the resource id passed in
func checkIfPropertyContainsResource(property interface{}, resource construct.ResourceId) bool {
	switch reflect.ValueOf(property).Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < reflect.ValueOf(property).Len(); i++ {
			val := reflect.ValueOf(property).Index(i).Interface()
			if id, ok := val.(construct.ResourceId); ok && id.Matches(resource) {
				return true
			}
			if pref, ok := val.(construct.PropertyRef); ok && pref.Resource.Matches(resource) {
				return true
			}
		}
	case reflect.Struct:
		if id, ok := property.(construct.ResourceId); ok && id.Matches(resource) {
			return true
		}
		if pref, ok := property.(construct.PropertyRef); ok && pref.Resource.Matches(resource) {
			return true
		}
	}
	return false
}

func checkIfLoneDependency(ctx solution_context.SolutionContext,
	resource, toCheck construct.ResourceId, direction knowledgebase.Direction,
	selector knowledgebase.ResourceSelector) (bool, error) {
	var resources []construct.ResourceId
	var err error
	// we are going to check if the resource was created as a unique resource by any of its direct dependents. if it was and that
	// dependent is not the other id, its not a valid candidate for this edge

	// here the direction matches because we are checking the resource for being used by another resource similar to other
	if direction == knowledgebase.DirectionDownstream {
		resources, err = solution_context.Upstream(ctx, resource, knowledgebase.ResourceDirectLayer)
		if err != nil {
			return false, err
		}
	} else {
		resources, err = solution_context.Downstream(ctx, resource, knowledgebase.ResourceDirectLayer)
		if err != nil {
			return false, err
		}
	}
	if len(resources) == 0 {
		return true, nil
	} else if len(resources) == 1 && resources[0].Matches(toCheck) {
		return true, nil
	} else {
		for _, res := range resources {
			depRes, err := ctx.RawView().Vertex(res)
			if err != nil {
				return false, err
			}
			data := knowledgebase.DynamicValueData{Resource: resource}
			dynCtx := solution_context.DynamicCtx(ctx)
			canUse, err := selector.CanUse(dynCtx, data, depRes)
			if err != nil {
				return false, err
			}
			if canUse {
				return false, nil
			}
		}
		return true, nil
	}
}

// checkIfCreatedAsUnique checks if the resource was created as a unique resource by any of its direct dependents. if it was and that
// dependent is not the other id, its not a valid candidate for this edge
func checkIfCreatedAsUniqueValidity(ctx solution_context.SolutionContext, resource, other *construct.Resource, direction knowledgebase.Direction) (bool, error) {
	var resources []construct.ResourceId
	var foundMatch bool
	var err error
	// we are going to check if the resource was created as a unique resource by any of its direct dependents. if it was and that
	// dependent is not the other id, its not a valid candidate for this edge

	// here the direction matches because we are checking the resource for being used by another resource similar to other
	if direction == knowledgebase.DirectionUpstream {
		resources, err = solution_context.Upstream(ctx, resource.ID, knowledgebase.ResourceDirectLayer)
		if err != nil {
			return false, err
		}
	} else {
		resources, err = solution_context.Downstream(ctx, resource.ID, knowledgebase.ResourceDirectLayer)
		if err != nil {
			return false, err
		}
	}
	// if the dependencies contains the other resource, dont run any checks as we assume its valid
	if collectionutil.Contains(resources, other.ID) {
		return true, nil
	}

	for _, res := range resources {

		// check if the upstream resource has a unique rule for the matched resource type
		template, err := ctx.KnowledgeBase().GetResourceTemplate(res)
		if err != nil || template == nil {
			return false, fmt.Errorf("error getting resource template for resource %s: %w", res, err)
		}
		currRes, err := ctx.RawView().Vertex(res)
		if err != nil {
			return false, err
		}
		err = template.LoopProperties(currRes, func(prop knowledgebase.Property) error {
			details := prop.Details()
			rule := details.OperationalRule
			if rule == nil || rule.Step == nil {
				return nil
			}
			step := rule.Step
			// we want the step to be the opposite of the direction passed in so we know its creating the resource in the direction of the resource
			// since we are looking at the resources dependencies
			if !step.Unique || step.Direction == direction {
				return nil
			}
			//check if the upstream resource is the same type as the matched resource type
			for _, selector := range step.Resources {
				match, err := selector.CanUse(solution_context.DynamicCtx(ctx), knowledgebase.DynamicValueData{Resource: currRes.ID},
					resource)
				if err != nil {
					return fmt.Errorf("error checking if resource %s matches selector %s: %w", other, selector, err)
				}
				// if its a match for the selectors, lets ensure that it has a dependency and exists in the properties of the rul
				if !match {
					continue
				}

				foundMatch = true
				return knowledgebase.ErrStopWalk
			}

			return nil
		})
		if err != nil {
			return false, err
		}
		if foundMatch {
			return false, nil
		}
	}
	return true, nil
}
