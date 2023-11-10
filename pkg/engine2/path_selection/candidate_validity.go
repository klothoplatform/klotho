package path_selection

import (
	"errors"
	"fmt"
	"strings"

	"github.com/klothoplatform/klotho/pkg/collectionutil"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

type (
	validityChecker interface {
		isValid(resourceToCheck, targetResource construct.ResourceId) (bool, error)
		makeValid(resource, operationResource construct.ResourceId) error
	}

	downstreamChecker struct {
		ctx solution_context.SolutionContext
	}
)

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
	nonBoundaryResources := path[1 : len(path)-1]
	matchIdx := matchesNonBoundary(resource.ID, nonBoundaryResources)
	if matchIdx < 0 {
		return false, nil
	}
	rt, err := ctx.KnowledgeBase().GetResourceTemplate(resource.ID)
	if err != nil {
		return false, err
	}
	if rt == nil {
		return true, nil
	}
	var errs error
	if matchIdx >= 1 {
		valid, err := checkAsTargetValidity(ctx, resource, path[:matchIdx+1], classification)
		if err != nil {
			errs = errors.Join(errs, err)
		}
		if !valid {
			return false, errs
		}
	}
	if matchIdx <= len(path)-3 {
		valid, err := checkAsSourceValidity(ctx, resource, path[matchIdx:], classification)
		if err != nil {
			errs = errors.Join(errs, err)
		}
		if !valid {
			return false, errs
		}
	}
	return true, errs
}

func checkAsTargetValidity(
	ctx solution_context.SolutionContext,
	resource *construct.Resource,
	path []construct.ResourceId,
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
					errs = errors.Join(errs, err)
					continue
				}
			}
			for _, res := range resources {
				valid, err := checkValidityOperation(ctx, path[0], res, ps)
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

func checkAsSourceValidity(
	ctx solution_context.SolutionContext,
	resource *construct.Resource,
	path []construct.ResourceId,
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
					errs = errors.Join(errs, err)
					continue
				}
				if len(resources) == 0 {
					err = assignForValidity(ctx, resource, path[len(path)-1], ps)
					errs = errors.Join(errs, err)
				}
			}
			for _, res := range resources {
				valid, err := checkValidityOperation(ctx, res, path[len(path)-1], ps)
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

func (d downstreamChecker) makeValid(resource, operationResource *construct.Resource, propertyRef string) error {
	downstreams, err := solution_context.Downstream(d.ctx, operationResource.ID, knowledgebase.FirstFunctionalLayer)
	if err != nil {
		return err
	}
	cfgCtx := solution_context.DynamicCtx(d.ctx)

	assign := func(resource construct.ResourceId, property string) (bool, error) {
		var errs error
		rt, err := d.ctx.KnowledgeBase().GetResourceTemplate(resource)
		if err != nil || rt == nil {
			return false, fmt.Errorf("error getting resource template for resource %s: %w", resource, err)
		}
		p := rt.Properties[property]
		for _, downstream := range downstreams {
			val, err := knowledgebase.TransformToPropertyValue(resource, property, downstream, cfgCtx,
				knowledgebase.DynamicValueData{Resource: resource})
			if err != nil || val == nil {
				continue // Becuase this error may just mean that its not the right type of resource
			}
			res, err := d.ctx.RawView().Vertex(resource)
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf("error getting resource %s: %w", resource, err))
				continue
			}
			if p.IsPropertyTypeScalar() {
				res.SetProperty(property, downstream)
				return true, errs
			} else {
				res.AppendProperty(property, downstream)
				return true, errs
			}
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
	return nil
}

func (d downstreamChecker) isValid(resourceToCheck, targetResource construct.ResourceId) (bool, error) {
	downstreams, err := solution_context.Downstream(d.ctx, resourceToCheck, knowledgebase.FirstFunctionalLayer)
	if err != nil {
		return false, err
	}
	return collectionutil.Contains(downstreams, targetResource), nil
}
