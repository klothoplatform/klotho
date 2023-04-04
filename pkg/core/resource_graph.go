package core

import (
	"reflect"

	"github.com/klothoplatform/klotho/pkg/graph"
	"go.uber.org/zap"
)

type (
	ResourceGraph struct {
		underlying *graph.Directed[Resource]
	}
)

func NewResourceGraph() *ResourceGraph {
	return &ResourceGraph{
		underlying: graph.NewDirected[Resource](),
	}
}

func (rg *ResourceGraph) AddResource(resource Resource) {
	if rg.GetResource(resource.Id()) == nil {
		rg.underlying.AddVertex(resource)
		zap.S().Debugf("adding resource: %s", resource.Id())
	}
}

// AddDependency2 deliberately renamed from AddDependency in the short term to catch when dependencies were
// being added in the inverse order. If either `source` or `dest` don't exist in the graph, they are added.
func (rg *ResourceGraph) AddDependency2(source Resource, dest Resource) {
	for _, res := range []Resource{source, dest} {
		rg.AddResource(res)
	}
	rg.underlying.AddEdge(source.Id(), dest.Id())
	zap.S().Debugf("adding %s -> %s", source.Id(), dest.Id())
}

func (rg *ResourceGraph) GetResource(id string) Resource {
	return rg.underlying.GetVertex(id)
}

func (rg *ResourceGraph) GetDependency(source string, target string) *graph.Edge[Resource] {
	return rg.underlying.GetEdge(source, target)
}

func (rg *ResourceGraph) ListResources() []Resource {
	return rg.underlying.GetAllVertices()
}

func (rg *ResourceGraph) ListDependencies() []graph.Edge[Resource] {
	return rg.underlying.GetAllEdges()
}

func (rg *ResourceGraph) VertexIdsInTopologicalOrder() ([]string, error) {
	return rg.underlying.VertexIdsInTopologicalOrder()
}

func (rg *ResourceGraph) GetDownstreamDependencies(source Resource) []graph.Edge[Resource] {
	return rg.underlying.OutgoingEdges(source)
}

func (rg *ResourceGraph) GetDownstreamResources(source Resource) []Resource {
	return rg.underlying.OutgoingVertices(source)
}

func (rg *ResourceGraph) GetUpstreamDependencies(source Resource) []graph.Edge[Resource] {
	return rg.underlying.IncomingEdges(source)
}

func (rg *ResourceGraph) GetUpstreamResources(source Resource) []Resource {
	return rg.underlying.IncomingVertices(source)
}

func (rg *ResourceGraph) TopologicalSort() ([]string, error) {
	return rg.underlying.VertexIdsInTopologicalOrder()
}

// AddDependenciesReflect uses reflection to inspect the fields of the resource given
// and add dependencies for each direct (ie, first-level) dependency.
//
// Supported field types (`*T` is a struct that implements Resource)
// - `SingleDependency   Resource`
// - `SpecificDependency *T`
// - `DependencyArray  []Resource`
// - `SpecificDepArray []*T`
// - `DependencyMap  map[string]Resource`
// - `SpecificDepMap map[string]*T`
func (rg *ResourceGraph) AddDependenciesReflect(source Resource) {
	rg.AddResource(source)

	sourceValue := reflect.ValueOf(source)
	sourceType := sourceValue.Type()
	if sourceType.Kind() == reflect.Pointer {
		sourceValue = sourceValue.Elem()
		sourceType = sourceType.Elem()
	}
	add := func(targetValue reflect.Value) {
		if targetValue.Kind() == reflect.Pointer && targetValue.IsNil() {
			return
		}
		if !targetValue.CanInterface() {
			return
		}
		if target, ok := targetValue.Interface().(Resource); ok {
			rg.AddDependency2(source, target)
		} else if wrapper, ok := targetValue.Interface().(*IaCValue); ok {
			rg.AddDependency2(source, wrapper.Resource)
		} else if wrapper, ok := targetValue.Interface().(IaCValue); ok {
			rg.AddDependency2(source, wrapper.Resource)
		}
	}
	for i := 0; i < sourceType.NumField(); i++ {
		// TODO maybe add a tag for options for things like ignoring fields

		fieldValue := sourceValue.Field(i)
		switch fieldValue.Kind() {
		case reflect.Slice, reflect.Array:
			for elemIdx := 0; elemIdx < fieldValue.Len(); elemIdx++ {
				elemValue := fieldValue.Index(elemIdx)
				add(elemValue)
			}

		case reflect.Map:
			for iter := fieldValue.MapRange(); iter.Next(); {
				elemValue := iter.Value()
				add(elemValue)
			}

		default:
			add(fieldValue)
		}
	}
}
