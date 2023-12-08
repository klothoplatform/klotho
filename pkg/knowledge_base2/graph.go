package knowledgebase2

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/graph_addons"
)

type (
	// DependencyLayer represents how far away a resource to return for the [Upstream]/[Downstream] methods.
	// 1. ResourceLocalLayer (layer 1) represents any unique resources the target resource needs to be operational
	// 2. ResourceGlueLayer (layer 2) represents all upstream/downstream resources that represent glue.
	//  This will not include any other functional resources and will stop searching paths
	//  once a functional resource is reached.
	// 3. FirstFunctionalLayer (layer 3) represents all upstream/downstream resources that represent glue and
	//  the first functional resource in other paths from the target resource.
	DependencyLayer string
)

const (

	// ResourceLocalLayer (layer 1)
	ResourceLocalLayer DependencyLayer = "local"
	// ResourceDirectLayer (layer 2)
	ResourceDirectLayer DependencyLayer = "direct"
	// ResourceGlueLayer (layer 2)
	ResourceGlueLayer DependencyLayer = "glue"
	// FirstFunctionalLayer (layer 3)
	FirstFunctionalLayer DependencyLayer = "first"
	// AllDepsLayer (layer 4)
	AllDepsLayer DependencyLayer = "all"
)

func resourceLocal(
	dag construct.Graph,
	kb TemplateKB,
	rid construct.ResourceId,
	ids *[]construct.ResourceId,
) graph_addons.WalkGraphFunc[construct.ResourceId] {
	return func(id construct.ResourceId, nerr error) error {
		isSideEffect, err := IsOperationalResourceSideEffect(dag, kb, rid, id)
		if err != nil {
			return errors.Join(nerr, err)
		}
		if isSideEffect {
			if ids != nil {
				(*ids) = append(*ids, id)
			}
			return nil
		}
		return graph_addons.SkipPath
	}
}

func resourceDirect(
	dag construct.Graph,
	ids *[]construct.ResourceId,
) graph_addons.WalkGraphFunc[construct.ResourceId] {
	return func(id construct.ResourceId, nerr error) error {
		if ids != nil {
			(*ids) = append(*ids, id)
		}
		return graph_addons.SkipPath
	}
}

func resourceGlue(
	kb TemplateKB,
	ids *[]construct.ResourceId,
) graph_addons.WalkGraphFunc[construct.ResourceId] {
	return func(id construct.ResourceId, nerr error) error {
		if GetFunctionality(kb, id) == Unknown {
			if ids != nil {
				(*ids) = append(*ids, id)
			}
			return nil
		}
		return graph_addons.SkipPath
	}
}

func firstFunctional(
	kb TemplateKB,
	ids *[]construct.ResourceId,
) graph_addons.WalkGraphFunc[construct.ResourceId] {
	return func(id construct.ResourceId, nerr error) error {
		if ids != nil {
			(*ids) = append(*ids, id)
		}
		if GetFunctionality(kb, id) == Unknown {
			return nil
		}
		return graph_addons.SkipPath
	}
}

func allDeps(
	ids *[]construct.ResourceId,
) graph_addons.WalkGraphFunc[construct.ResourceId] {
	return func(id construct.ResourceId, nerr error) error {
		if ids != nil {
			(*ids) = append(*ids, id)
		}
		return nil
	}
}

// DependenciesSkipEdgeLayer returns a function which can be used in calls to
// [construct.DownstreamDependencies] and [construct.UpstreamDependencies].
func DependenciesSkipEdgeLayer(
	dag construct.Graph,
	kb TemplateKB,
	rid construct.ResourceId,
	layer DependencyLayer,
) func(construct.Edge) bool {
	switch layer {
	case ResourceLocalLayer:
		return func(e construct.Edge) bool {
			isSideEffect, err := IsOperationalResourceSideEffect(dag, kb, rid, e.Target)
			return err != nil || !isSideEffect
		}

	case ResourceGlueLayer:
		return func(e construct.Edge) bool {
			return GetFunctionality(kb, e.Target) != Unknown
		}

	case FirstFunctionalLayer:
		return func(e construct.Edge) bool {
			// Keep the source -> X edges, since source likely is != Unknown
			if e.Source == rid {
				return false
			}
			// Unknown -> X edges are not interesting, keep those
			if GetFunctionality(kb, e.Source) == Unknown {
				return false
			}
			// Since source is now != Unknown, only keep edges w/ target == Unknown
			return GetFunctionality(kb, e.Target) != Unknown
		}

	default:
		fallthrough
	case AllDepsLayer:
		return construct.DontSkipEdges
	}
}

func Downstream(dag construct.Graph, kb TemplateKB, rid construct.ResourceId, layer DependencyLayer) ([]construct.ResourceId, error) {
	var result []construct.ResourceId
	var f graph_addons.WalkGraphFunc[construct.ResourceId]
	switch layer {
	case ResourceLocalLayer:
		f = resourceLocal(dag, kb, rid, &result)
	case ResourceDirectLayer:
		f = resourceDirect(dag, &result)
	case ResourceGlueLayer:
		f = resourceGlue(kb, &result)
	case FirstFunctionalLayer:
		f = firstFunctional(kb, &result)
	case AllDepsLayer:
		f = allDeps(&result)
	default:
		return nil, fmt.Errorf("unknown layer %s", layer)
	}
	err := graph_addons.WalkDown(dag, rid, f)
	return result, err
}

func DownstreamFunctional(dag construct.Graph, kb TemplateKB, resource construct.ResourceId) ([]construct.ResourceId, error) {
	var result []construct.ResourceId
	err := graph_addons.WalkDown(dag, resource, func(id construct.ResourceId, nerr error) error {
		if GetFunctionality(kb, id) != Unknown {
			result = append(result, id)
			return graph_addons.SkipPath
		}
		return nil
	})
	return result, err
}

func Upstream(dag construct.Graph, kb TemplateKB, rid construct.ResourceId, layer DependencyLayer) ([]construct.ResourceId, error) {
	var result []construct.ResourceId
	var f graph_addons.WalkGraphFunc[construct.ResourceId]
	switch layer {
	case ResourceLocalLayer:
		f = resourceLocal(dag, kb, rid, &result)
	case ResourceDirectLayer:
		f = resourceDirect(dag, &result)
	case ResourceGlueLayer:
		f = resourceGlue(kb, &result)
	case FirstFunctionalLayer:
		f = firstFunctional(kb, &result)
	case AllDepsLayer:
		f = allDeps(&result)
	default:
		return nil, fmt.Errorf("unknown layer %s", layer)
	}
	err := graph_addons.WalkUp(dag, rid, f)
	return result, err
}

func layerWalkFunc(
	dag construct.Graph,
	kb TemplateKB,
	rid construct.ResourceId,
	layer DependencyLayer,
	result []construct.ResourceId,
) (graph_addons.WalkGraphFunc[construct.ResourceId], error) {
	switch layer {
	case ResourceLocalLayer:
		return resourceLocal(dag, kb, rid, &result), nil
	case ResourceDirectLayer:
		return resourceDirect(dag, &result), nil
	case ResourceGlueLayer:
		return resourceGlue(kb, &result), nil
	case FirstFunctionalLayer:
		return firstFunctional(kb, &result), nil
	case AllDepsLayer:
		return allDeps(&result), nil
	default:
		return nil, fmt.Errorf("unknown layer %s", layer)
	}
}

func UpstreamFunctional(dag construct.Graph, kb TemplateKB, resource construct.ResourceId) ([]construct.ResourceId, error) {
	var result []construct.ResourceId
	err := graph_addons.WalkUp(dag, resource, func(id construct.ResourceId, nerr error) error {
		if GetFunctionality(kb, id) != Unknown {
			result = append(result, id)
			return graph_addons.SkipPath
		}
		return nil
	})
	return result, err
}

func IsOperationalResourceSideEffect(dag construct.Graph, kb TemplateKB, rid, sideEffect construct.ResourceId) (bool, error) {
	template, err := kb.GetResourceTemplate(rid)
	if err != nil {
		return false, fmt.Errorf("error cheecking %s is side effect of %s: %w", sideEffect, rid, err)
	}
	sideEffectResource, err := dag.Vertex(sideEffect)
	if err != nil {
		return false, fmt.Errorf("could not find side effect resource %s: %w", sideEffect, err)
	}
	resource, err := dag.Vertex(rid)
	if err != nil {
		return false, fmt.Errorf("could not find resource %s: %w", rid, err)
	}

	dynCtx := DynamicValueContext{Graph: dag, KnowledgeBase: kb}
	for _, property := range template.Properties {
		ruleSatisfied := false
		rule := property.Details().OperationalRule
		if rule == nil || len(rule.Step.Resources) == 0 {
			continue
		}
		step := rule.Step
		// We only check if the resource selector is a match in terms of properties and classifications (not the actual id)
		// We do this because if we have explicit ids in the selector and someone changes the id of a side effect resource
		// we would no longer think it is a side effect since the id would no longer match.
		// To combat this we just check against type
		for j, resourceSelector := range step.Resources {
			if match, err := resourceSelector.IsMatch(dynCtx, DynamicValueData{Resource: rid}, sideEffectResource); match {
				ruleSatisfied = true
				break
			} else if err != nil {
				return false, fmt.Errorf(
					"error checking if %s is side effect of %s in property %s, resource %d: %w",
					sideEffect, rid, property.Details().Name, j, err,
				)
			}
		}

		// If the side effect resource fits the rule we then perform 2 more checks
		// 1. is there a path in the direction of the rule
		// 2. Is the property set with the resource that we are checking for
		if ruleSatisfied {
			if step.Direction == DirectionUpstream {
				resources, err := graph.ShortestPathStable(dag, sideEffect, rid, construct.ResourceIdLess)
				if len(resources) == 0 || err != nil {
					continue
				}
			} else {
				resources, err := graph.ShortestPathStable(dag, rid, sideEffect, construct.ResourceIdLess)
				if len(resources) == 0 || err != nil {
					continue
				}
			}

			propertyVal, err := resource.GetProperty(property.Details().Path)
			if err != nil || propertyVal == nil {
				continue
			}
			val := reflect.ValueOf(propertyVal)
			if val.Kind() == reflect.Array || val.Kind() == reflect.Slice {
				for i := 0; i < val.Len(); i++ {
					if arrId, ok := val.Index(i).Interface().(construct.ResourceId); ok && arrId == sideEffect {
						return true, nil
					}
				}
			} else {
				if val.IsZero() {
					continue
				}
				if valId, ok := val.Interface().(construct.ResourceId); ok && valId == sideEffect {
					return true, nil
				}
			}
		}

	}
	return false, nil
}
