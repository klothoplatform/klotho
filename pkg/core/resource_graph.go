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

var (
	resourceType = reflect.TypeOf((*Resource)(nil)).Elem()
)

func NewResourceGraph() *ResourceGraph {
	return &ResourceGraph{
		underlying: graph.NewDirected[Resource](),
	}
}

func (cg *ResourceGraph) AddResource(resource Resource) {
	cg.underlying.AddVertex(resource)
	zap.S().Debugf("adding resource: %s", resource.Id())
}

// AddDependency2 deliberately renamed from AddDependency in the short term to catch when dependencies were
// being added in the inverse order. If either `source` or `dest` don't exist in the graph, they are added.
func (rg *ResourceGraph) AddDependency2(source Resource, dest Resource) {
	for _, res := range []Resource{source, dest} {
		if rg.GetResource(res.Id()) == nil {
			rg.AddResource(res)
		}
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
	rv := reflect.ValueOf(source)
	rt := rv.Type()
	if rt.Kind() == reflect.Pointer {
		rv = rv.Elem()
		rt = rt.Elem()
	}
	for i := 0; i < rt.NumField(); i++ {
		// TODO maybe add a tag for options for things like ignoring fields

		v := rv.Field(i)
		switch v.Kind() {
		case reflect.Interface, reflect.Pointer:
			r, ok := v.Interface().(Resource)
			if ok {
				rg.AddDependency2(source, r)
			}

		case reflect.Slice, reflect.Array:
			et := rt.Field(i).Type.Elem()
			if et.Implements(resourceType) {
				for ei := 0; ei < v.Len(); ei++ {
					ev := v.Index(ei)
					if r, ok := ev.Interface().(Resource); ok {
						rg.AddDependency2(source, r)
					}
				}
			}

		case reflect.Map:
			et := rt.Field(i).Type.Elem()
			if et.Implements(resourceType) {
				for iter := v.MapRange(); iter.Next(); {
					ev := iter.Value()
					if r, ok := ev.Interface().(Resource); ok {
						rg.AddDependency2(source, r)
					}
				}
			}
		}
	}
}
