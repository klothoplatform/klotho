package path_selection

import (
	"errors"
	"fmt"
	"strings"

	"github.com/klothoplatform/klotho/pkg/collectionutil"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine/solution_context"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"go.uber.org/zap"
)

type (
	// downstreamChecker is a validityChecker that checks if a candidate is valid based on what is downstream of the specified
	// resources
	downstreamChecker struct {
		ctx solution_context.SolutionContext
	}
)

// checkCandidatesValidity checks if the candidate is valid based on the validity of its own path satisfaction rules and namespace
func checkCandidatesValidity(
	ctx solution_context.SolutionContext,
	resource *construct.Resource,
	path []construct.ResourceId,
	classification string,
) (bool, error) {
	// We only care if the validity is true if its not a direct edge since we know direct edges are valid
	if len(path) <= 3 {
		return true, nil
	}
	rt, err := ctx.KnowledgeBase().GetResourceTemplate(resource.ID)
	if err != nil || rt == nil {
		return false, err
	}

	var errs error
	// check validity of candidate being a target if not direct edge to source
	valid, err := checkAsTargetValidity(ctx, resource, path[0], classification)
	if err != nil {
		errs = errors.Join(errs, err)
	}
	if !valid {
		zap.S().Debugf("candidate %s is not valid as target", resource.ID)
		return false, errs
	}

	// check validity of candidate being a source if not direct edge to target
	valid, err = checkAsSourceValidity(ctx, resource, path[len(path)-1], classification)
	if err != nil {
		errs = errors.Join(errs, err)
	}
	if !valid {
		zap.S().Debugf("candidate %s is not valid as source", resource.ID)
		return false, errs
	}
	return true, errs
}

// checkNamespaceValidity checks if the candidate is valid based on the namespace it is a part of.
// If the candidate is namespaced and the target is not in the same namespace,
//
//	then the candidate is not valid if those namespace resources are the same type
func checkNamespaceValidity(
	ctx solution_context.SolutionContext,
	resource *construct.Resource,
	target construct.ResourceId,
) (bool, error) {
	// Check if its a valid namespaced resource
	ids, err := ctx.KnowledgeBase().GetAllowedNamespacedResourceIds(solution_context.DynamicCtx(ctx), resource.ID)
	if err != nil {
		return false, err
	}
	for _, i := range ids {
		if i.Matches(target) {
			ns, err := ctx.KnowledgeBase().GetResourcesNamespaceResource(resource)
			if err != nil {
				return false, err
			}
			if !ns.Matches(target) {
				return false, nil
			}
		}
	}
	return true, nil
}

// checkAsTargetValidity checks if the candidate is valid based on the validity of its own path satisfaction rules
// for the specified classification. If the candidate uses property references to check validity then the candidate
// can be considered valid if those properties are not set
func checkAsTargetValidity(
	ctx solution_context.SolutionContext,
	resource *construct.Resource,
	source construct.ResourceId,
	classification string,
) (bool, error) {
	rt, err := ctx.KnowledgeBase().GetResourceTemplate(resource.ID)
	if err != nil {
		return false, err
	}
	if rt == nil {
		return true, nil
	}
	var errs error
	for _, ps := range rt.PathSatisfaction.AsTarget {
		if ps.Classification == classification && ps.Validity != "" {
			resources := []construct.ResourceId{resource.ID}
			if ps.PropertyReference != "" {
				resources, err = solution_context.GetResourcesFromPropertyReference(ctx,
					resource.ID, ps.PropertyReference)
				if err != nil {
					// dont return error because it just means that the property isnt set and we can make the
					// resource valid
					zap.S().Debugf(
						"no resource available from resource %s from property ref %s: %v",
						resource.ID, ps.PropertyReference, err,
					)
				}
				if len(resources) == 0 {
					err = assignForValidity(ctx, resource, source, ps)
					errs = errors.Join(errs, err)
				}
			}
			for _, res := range resources {
				valid, err := checkValidityOperation(ctx, source, res, ps)
				if err != nil {
					errs = errors.Join(errs, err)
				}
				if !valid {
					return false, errs
				}
			}
		}
	}
	return true, errs
}

// checkAsSourceValidity checks if the candidate is valid based on the validity of its own path satisfaction rules
// for the specified classification. If the candidate uses property references to check validity then the candidate
// can be considered valid if those properties are not set
func checkAsSourceValidity(
	ctx solution_context.SolutionContext,
	resource *construct.Resource,
	target construct.ResourceId,
	classification string,
) (bool, error) {
	rt, err := ctx.KnowledgeBase().GetResourceTemplate(resource.ID)
	if err != nil {
		return false, err
	}
	if rt == nil {
		return true, nil
	}
	var errs error
	for _, ps := range rt.PathSatisfaction.AsSource {
		if ps.Classification == classification && ps.Validity != "" {
			resources := []construct.ResourceId{resource.ID}
			if ps.PropertyReference != "" {
				resources, err = solution_context.GetResourcesFromPropertyReference(ctx,
					resource.ID, ps.PropertyReference)
				if err != nil {
					// dont return error because it just means that the property isnt set and we can make the
					// resource valid
					zap.S().Debugf(
						"no resource available from resource %s from property ref %s: %v",
						resource.ID, ps.PropertyReference, err,
					)
				}
				if len(resources) == 0 {
					err = assignForValidity(ctx, resource, target, ps)
					errs = errors.Join(errs, err)
				}
			}
			for _, res := range resources {
				valid, err := checkValidityOperation(ctx, res, target, ps)
				if err != nil {
					errs = errors.Join(errs, err)
				}
				if !valid {
					return false, errs
				}
			}
		}
	}
	return true, errs
}

// checkValidityOperation checks if the candidate is valid based on the operation the validity check specifies
func checkValidityOperation(
	ctx solution_context.SolutionContext,
	src, target construct.ResourceId,
	ps knowledgebase.PathSatisfactionRoute,
) (bool, error) {
	var errs error
	switch ps.Validity {
	case knowledgebase.DownstreamOperation:
		valid, err := downstreamChecker{ctx}.isValid(src, target)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("error checking downstream validity: %w", err))
		}
		if !valid {
			return false, errs
		}
	}

	return true, errs
}

// assignForValidity assigns the candidate to be valid based on the operation the validity check specified
// This is allowed to be run if the property reference used in the validity check is not set on the candidate
func assignForValidity(
	ctx solution_context.SolutionContext,
	resource *construct.Resource,
	operationResourceId construct.ResourceId,
	ps knowledgebase.PathSatisfactionRoute,
) error {
	operationResource, err := ctx.RawView().Vertex(operationResourceId)
	if err != nil {
		return err
	}
	var errs error
	switch ps.Validity {
	case knowledgebase.DownstreamOperation:
		err := downstreamChecker{ctx}.makeValid(resource, operationResource, ps.PropertyReference)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("error making resource downstream validity: %w", err))
		}
	}
	return errs
}

// makeValid makes the candidate valid based on the operation the validity check specified
// It will find a resource to assign to the propertyRef specified based on what is downstream of the operationResource.
func (d downstreamChecker) makeValid(resource, operationResource *construct.Resource, propertyRef string) error {
	downstreams, err := solution_context.Downstream(d.ctx, operationResource.ID, knowledgebase.FirstFunctionalLayer)
	if err != nil {
		return err
	}
	// include the operation resource in downstreams in case it also can be assigned to the target property
	downstreams = append(downstreams, operationResource.ID)
	cfgCtx := solution_context.DynamicCtx(d.ctx)

	assign := func(r construct.ResourceId, property string) (bool, error) {
		var errs error
		rt, err := d.ctx.KnowledgeBase().GetResourceTemplate(r)
		if err != nil || rt == nil {
			return false, fmt.Errorf("error getting resource template for resource %s: %w", resource.ID, err)
		}
		p := rt.Properties[property]
		for _, downstream := range downstreams {
			val, err := knowledgebase.TransformToPropertyValue(r, property, downstream, cfgCtx,
				knowledgebase.DynamicValueData{Resource: r})
			if err != nil || val == nil {
				continue // Because this error may just mean that its not the right type of resource
			}
			// We need to check if the current resource is what we are operating on and if so not search our raw view
			// this is because it could be a phantom resource
			var currRes *construct.Resource
			if resource.ID == r {
				currRes = resource
			} else {
				currRes, err = d.ctx.RawView().Vertex(r)
				if err != nil {
					errs = errors.Join(errs, fmt.Errorf("error getting resource %s: %w", resource.ID, err))
					continue
				}
			}
			return true, errors.Join(errs, p.AppendProperty(currRes, downstream))
		}
		return false, errs
	}

	var errs error
	parts := strings.Split(propertyRef, "#")
	currResources := []construct.ResourceId{resource.ID}
	for _, part := range parts {
		var nextResources []construct.ResourceId
		for _, currResource := range currResources {
			val, err := cfgCtx.FieldValue(part, currResource)
			if err != nil {
				_, err = assign(currResource, part)
				if err != nil {
					errs = errors.Join(errs, err)
				}
				continue
			}
			if id, ok := val.(construct.ResourceId); ok {
				nextResources = append(nextResources, id)
			} else if ids, ok := val.([]construct.ResourceId); ok {
				nextResources = append(nextResources, ids...)
			}
		}
		currResources = nextResources
	}
	return errs
}

// isValid checks if the candidate is valid based on what is downstream of the resourceToCheck
func (d downstreamChecker) isValid(resourceToCheck, targetResource construct.ResourceId) (bool, error) {
	downstreams, err := solution_context.Downstream(d.ctx, resourceToCheck, knowledgebase.FirstFunctionalLayer)
	if err != nil {
		return false, err
	}
	return collectionutil.Contains(downstreams, targetResource), nil
}
