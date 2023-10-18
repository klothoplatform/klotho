package knowledgebase2

import (
	"reflect"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
)

type (
	// DependencyLayer represents how far away a resource to return for the [Upstream]/[Downstream] methods.
	// 1. ResourceLocalLayer (layer 1) represents any unique resources the target resource needs to be operational
	// 2. ResourceGlueLayer (layer 2) represents all upstream/downstream resources that represent glue.
	//  This will not include any other functional resources and will stop searching paths
	//  once a functional resource is reached.
	// 3. FirstFunctionalLayer (layer 3) represents all upstream/downstream resources that represent glue and
	//  the first functional resource in other paths from the target resource.
	DependencyLayer int
)

const (
	_ DependencyLayer = iota

	// ResourceLocalLayer (layer 1)
	ResourceLocalLayer
	// ResourceGlueLayer (layer 2)
	ResourceGlueLayer
	// FirstFunctionalLayer (layer 3)
	FirstFunctionalLayer
	// AllDepsLayer (layer 4)
	AllDepsLayer
)

func Downstream(dag construct.Graph, kb TemplateKB, rid construct.ResourceId, layer DependencyLayer) ([]construct.ResourceId, error) {
	ids, err := construct.AllDownstreamDependencies(dag, rid)
	if err != nil {
		return nil, err
	}
	result := make([]construct.ResourceId, 0, len(ids))
	switch layer {
	case ResourceLocalLayer:
		for _, id := range ids {
			_, err := dag.Edge(rid, id)
			if err != nil {
				continue
			}
			if IsOperationalResourceSideEffect(dag, kb, rid, id) {
				result = append(result, id)
			}
			return result, nil
		}
	case ResourceGlueLayer:
		for _, res := range ids {
			if kb.GetFunctionality(res) == Unknown && isDownstreamWithinFunctionalBoundary(dag, kb, rid, res) {
				result = append(result, res)
			}
		}
		return result, nil
	case FirstFunctionalLayer:
		for _, res := range ids {
			if isDownstreamWithinFunctionalBoundary(dag, kb, rid, res) {
				result = append(result, res)
			}
		}
		return result, nil
	case AllDepsLayer:
		return ids, nil
	}
	return nil, nil
}

func DownstreamFunctional(dag construct.Graph, kb TemplateKB, resource construct.ResourceId) ([]construct.ResourceId, error) {
	resources, err := Downstream(dag, kb, resource, FirstFunctionalLayer)
	if err != nil {
		return nil, err
	}
	result := make([]construct.ResourceId, 0, len(resources))
	for _, res := range resources {
		if kb.GetFunctionality(res) != Unknown {
			result = append(result, res)
		}
	}
	return result, nil
}

func isAllWithinFunctionalBoundary(dag construct.Graph, kb TemplateKB, ids []construct.ResourceId) bool {
	for _, id := range ids {
		if kb.GetFunctionality(id) != Unknown {
			return false
		}
	}
	return true
}

func isDownstreamWithinFunctionalBoundary(dag construct.Graph, kb TemplateKB, resource, downstream construct.ResourceId) bool {
	paths, err := graph.AllPathsBetween(dag, resource, downstream)
	if err != nil {
		return false
	}
	for _, path := range paths {
		if isAllWithinFunctionalBoundary(dag, kb, path[1:len(path)-1]) {
			return true
		}
	}
	return true
}

func isUpstreamWithinFunctionalBoundary(dag construct.Graph, kb TemplateKB, resource, downstream construct.ResourceId) bool {
	paths, err := graph.AllPathsBetween(dag, downstream, resource)
	if err != nil {
		return false
	}
	for _, path := range paths {
		if isAllWithinFunctionalBoundary(dag, kb, path[1:len(path)-1]) {
			return true
		}
	}
	return true
}

func Upstream(dag construct.Graph, kb TemplateKB, resource construct.ResourceId, layer DependencyLayer) ([]construct.ResourceId, error) {
	ids, err := construct.AllUpstreamDependencies(dag, resource)
	if err != nil {
		return nil, err
	}
	result := make([]construct.ResourceId, 0, len(ids))
	switch layer {
	case ResourceLocalLayer:
		for _, res := range ids {
			_, err := dag.Edge(res, resource)
			if err != nil {
				continue
			}
			if IsOperationalResourceSideEffect(dag, kb, resource, res) {
				result = append(result, res)
			}
			return result, nil
		}
	case ResourceGlueLayer:
		for _, res := range ids {
			if kb.GetFunctionality(res) == Unknown && isUpstreamWithinFunctionalBoundary(dag, kb, resource, res) {
				result = append(result, res)
			}
		}
		return result, nil
	case FirstFunctionalLayer:
		for _, res := range ids {
			if isUpstreamWithinFunctionalBoundary(dag, kb, resource, res) {
				result = append(result, res)
			}
		}
		return result, nil
	case AllDepsLayer:
		return ids, nil
	}
	return nil, nil
}

func UpstreamFunctional(dag construct.Graph, kb TemplateKB, resource construct.ResourceId) ([]construct.ResourceId, error) {
	resources, err := Upstream(dag, kb, resource, FirstFunctionalLayer)
	if err != nil {
		return nil, err
	}
	result := make([]construct.ResourceId, 0, len(resources))
	for _, res := range resources {
		functionality := kb.GetFunctionality(res)
		if functionality != Unknown {
			result = append(result, res)
		}
	}
	return result, nil
}

func IsOperationalResourceSideEffect(dag construct.Graph, kb TemplateKB, rid, sideEffect construct.ResourceId) bool {
	template, err := kb.GetResourceTemplate(rid)
	if template == nil || err != nil {
		return false
	}
	sideEffectResource, err := dag.Vertex(sideEffect)
	if err != nil {
		return false
	}
	resource, err := dag.Vertex(rid)
	if err != nil {
		return false
	}

	dynCtx := DynamicValueContext{Graph: dag, KnowledgeBase: kb}
	for _, property := range template.Properties {
		ruleSatisfied := false
		if property.OperationalRule == nil {
			continue
		}
		rule := property.OperationalRule
		for _, step := range rule.Steps {
			// We only check if the resource selector is a match in terms of properties and classifications (not the actual id)
			// We do this because if we have explicit ids in the selector and someone changes the id of a side effect resource
			// we would no longer think it is a side effect since the id would no longer match.
			// To combat this we just check against type
			for _, resourceSelector := range step.Resources {
				if resourceSelector.IsMatch(dynCtx, DynamicValueData{Resource: rid}, sideEffectResource) {
					ruleSatisfied = true
					break
				}
			}

			// If the side effect resource fits the rule we then perform 2 more checks
			// 1. is there a path in the direction of the rule
			// 2. Is the property set with the resource that we are checking for
			if ruleSatisfied {
				if step.Direction == DirectionUpstream {
					resources, err := graph.ShortestPath(dag, sideEffect, rid)
					if len(resources) == 0 || err != nil {
						continue
					}
				} else {
					resources, err := graph.ShortestPath(dag, rid, sideEffect)
					if len(resources) == 0 || err != nil {
						continue
					}
				}

				propertyVal, err := resource.GetProperty(property.Path)
				if err != nil {
					continue
				}
				val := reflect.ValueOf(propertyVal)
				if val.Kind() == reflect.Array || val.Kind() == reflect.Slice {
					for i := 0; i < val.Len(); i++ {
						if arrId, ok := val.Index(i).Interface().(construct.ResourceId); ok && arrId == sideEffect {
							return true
						}
					}
				} else {
					if valId, ok := val.Interface().(construct.ResourceId); ok && valId == sideEffect {
						return true
					}
				}
			}
		}
	}
	return false
}
