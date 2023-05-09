package core

import (
	"reflect"

	"github.com/klothoplatform/klotho/pkg/graph"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"go.uber.org/zap"
)

type (
	ResourceGraph struct {
		underlying *graph.Directed[Resource]
	}
)

func NewResourceGraph() *ResourceGraph {
	return &ResourceGraph{
		underlying: graph.NewDirected(func(r Resource) string {
			return r.Id().String()
		}),
	}
}

func (rg *ResourceGraph) AddResource(resource Resource) {
	if rg.GetResource(resource.Id()) == nil {
		rg.underlying.AddVertex(resource)
		zap.S().Debugf("adding resource: %s", resource.Id())
	}
}

// Adds a dependency such that `deployedSecond` has to be deployed after `deployedFirst`. This makes the left-to-right
// association consistent with our visualizer, and with the Go struct graph.
//
// For example, if you have a Lambda and its execution role, then:
//
//	╭────────────────╮   ╭────────────────╮
//	│ LambdaFunction ├──➤│    IamRole     │
//	│ deployedSecond │   │ deployedFirst  │
//	╰────────────────╯   ╰────────────────╯
//
// And you would use it as:
//
//	lambda := LambdaFunction {
//		Role: executionRole
//		...
//	}
//
//	rg.AddDependency(lambda, lambda.Role)
func (rg *ResourceGraph) AddDependency(deployedSecond Resource, deployedFirst Resource) {
	for _, res := range []Resource{deployedSecond, deployedFirst} {
		rg.AddResource(res)
	}
	if cycle, _ := rg.underlying.CreatesCycle(deployedSecond.Id().String(), deployedFirst.Id().String()); cycle {
		zap.S().Errorf("Not Adding Dependency, Cycle would be created from edge %s -> %s", deployedSecond.Id(), deployedFirst.Id())
	} else {
		rg.underlying.AddEdge(deployedSecond.Id().String(), deployedFirst.Id().String())
		zap.S().Debugf("adding %s -> %s", deployedSecond.Id(), deployedFirst.Id())
	}
}

func (rg *ResourceGraph) GetResource(id ResourceId) Resource {
	return rg.underlying.GetVertex(id.String())
}

func GetResourceOfType[T Resource](g *ResourceGraph, id string) *T {
	vertices := g.underlying.GetAllVertices()
	for _, v := range vertices {
		if vT, ok := v.(T); ok {
			if vT.Id().String() == id {
				return &vT
			}
		}
	}
	return nil
}

func (rg *ResourceGraph) GetResourceByVertexId(id string) Resource {
	return rg.underlying.GetVertex(id)
}

func (rg *ResourceGraph) GetDependency(source Resource, target Resource) *graph.Edge[Resource] {
	return rg.underlying.GetEdge(rg.underlying.IdForNode(source), rg.underlying.IdForNode(target))
}

func (rg *ResourceGraph) GetDependencyByVertexIds(source string, target string) *graph.Edge[Resource] {
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
// and add dependencies for each dependency nested within the object.
// Structs that are a type of a valid dependency, will not be recursed further as they will already have a
// direct dependency to their own fields.
//
// Supported field types (`*T` is a struct that implements Resource, *K is a struct that does not implement Resource)
// - `SingleDependency   Resource`
// - `SpecificDependency *T`
// - `DependencyArray  []Resource`
// - `SpecificDepArray []*T`
// - `DependencyMap  map[string]Resource`
// - `SpecificDepMap map[string]*T`
// - `NestedStructDependency K`
// - `NestedSpecificDependency *K`
// - `NestedDependencyArray  []K`
// - `NestedSpecificDepArray []*K`
// - `NestedDependencyMap  map[string]K`
// - `NestedSpecificDepMap map[string]*K`

func (rg *ResourceGraph) AddDependenciesReflect(source Resource) {
	rg.AddResource(source)

	sourceValue := reflect.ValueOf(source)
	sourceType := sourceValue.Type()
	if sourceType.Kind() == reflect.Pointer {
		sourceValue = sourceValue.Elem()
		sourceType = sourceType.Elem()
	}
	for i := 0; i < sourceType.NumField(); i++ {
		// TODO maybe add a tag for options for things like ignoring fields

		fieldValue := sourceValue.Field(i)
		switch fieldValue.Kind() {
		case reflect.Slice, reflect.Array:
			for elemIdx := 0; elemIdx < fieldValue.Len(); elemIdx++ {
				elemValue := fieldValue.Index(elemIdx)
				rg.addDependenciesReflect(source, elemValue)
			}

		case reflect.Map:
			for iter := fieldValue.MapRange(); iter.Next(); {
				elemValue := iter.Value()
				rg.addDependenciesReflect(source, elemValue)
			}

		default:
			rg.addDependenciesReflect(source, fieldValue)
		}
	}
}
func (rg *ResourceGraph) addDependenciesReflect(source Resource, targetValue reflect.Value) {
	if targetValue.Kind() == reflect.Pointer && targetValue.IsNil() {
		return
	}
	if !targetValue.CanInterface() {
		return
	}
	switch value := targetValue.Interface().(type) {
	case Resource:
		rg.AddDependency(source, value)
	case *IaCValue:
		if value.Resource != nil {
			rg.AddDependency(source, value.Resource)
		}
	case IaCValue:
		if value.Resource != nil {
			rg.AddDependency(source, value.Resource)
		}
	default:
		correspondingValue := targetValue
		for correspondingValue.Kind() == reflect.Pointer {
			correspondingValue = targetValue.Elem()
		}
		switch correspondingValue.Kind() {

		case reflect.Struct:
			for i := 0; i < correspondingValue.NumField(); i++ {
				childVal := correspondingValue.Field(i)
				rg.addDependenciesReflect(source, childVal)
			}
		case reflect.Slice, reflect.Array:
			for elemIdx := 0; elemIdx < correspondingValue.Len(); elemIdx++ {
				elemValue := correspondingValue.Index(elemIdx)
				rg.addDependenciesReflect(source, elemValue)
			}

		case reflect.Map:
			for iter := correspondingValue.MapRange(); iter.Next(); {
				elemValue := iter.Value()
				rg.addDependenciesReflect(source, elemValue)
			}

		}
	}
}

func (rg *ResourceGraph) GetAllUpstreamResources(source Resource) []Resource {
	var upstreams []Resource
	upstreamsSet := map[Resource]struct{}{}
	for r := range rg.getAllUpstreamResourcesSet(source, upstreamsSet) {
		upstreams = append(upstreams, r)
	}
	return upstreams
}

func (rg *ResourceGraph) getAllUpstreamResourcesSet(source Resource, upstreams map[Resource]struct{}) map[Resource]struct{} {
	for _, r := range rg.underlying.IncomingVertices(source) {
		upstreams[r] = struct{}{}
		rg.getAllUpstreamResourcesSet(r, upstreams)
	}
	return upstreams
}

func (rg *ResourceGraph) GetAllDownstreamResources(source Resource) []Resource {
	var upstreams []Resource
	upstreamsSet := map[Resource]struct{}{}
	for r := range rg.getAllDownstreamResourcesSet(source, upstreamsSet) {
		upstreams = append(upstreams, r)
	}
	return upstreams
}

func (rg *ResourceGraph) getAllDownstreamResourcesSet(source Resource, upstreams map[Resource]struct{}) map[Resource]struct{} {
	for _, r := range rg.underlying.OutgoingVertices(source) {
		upstreams[r] = struct{}{}
		rg.getAllDownstreamResourcesSet(r, upstreams)
	}
	return upstreams
}

// CreateRecursively takes in a resource and set of metadata and looks at any fields which point to specific dependencies which match the Resource Interface
// If a specific dependency is found, the .Create method will be called to create the parent resource.
// Each specific dependency value will be updated on the resources field itself
// This method will recurse down each one of the resources fields and do a DFS until a Resource is found
//
// IaCValues today are not labeled as a specific resource, so the value must already be present. This method will still add the dependency, but does not have the knowledge of which resource to create
func (rg *ResourceGraph) CreateRecursively(res Resource, metadata map[string]any) error {
	var merr multierr.Error
	rg.AddResource(res)

	source := reflect.ValueOf(res)
	for source.Kind() == reflect.Pointer {
		source = source.Elem()
	}
	for i := 0; i < source.NumField(); i++ {
		targetValue := source.Field(i)
		merr.Append(rg.actOnValue(targetValue, res, metadata, nil, reflect.Value{}))
	}
	return merr.ErrOrNil()
}

func (rg *ResourceGraph) actOnValue(targetValue reflect.Value, res Resource, metadata map[string]any, parent *reflect.Value, index reflect.Value) error {
	switch value := targetValue.Interface().(type) {
	case Resource:
		resource, err := value.Create(rg, metadata)
		if err == nil && resource != nil {
			rg.AddDependency(res, resource)
			if parent != nil {
				parent.SetMapIndex(index, reflect.ValueOf(resource))
			} else {
				targetValue.Set(reflect.ValueOf(resource))
			}
		} else {
			return err
		}
	case *IaCValue:
		if value != nil && value.Resource != nil {
			resource, err := value.Resource.Create(rg, metadata)
			if err == nil && resource != nil {
				rg.AddDependency(res, resource)
				value.Resource = resource
				if parent != nil {
					parent.SetMapIndex(index, reflect.ValueOf(value))
				} else {
					targetValue.Set(reflect.ValueOf(value))
				}
			} else {
				return err
			}
		}
	case IaCValue:
		if value.Resource != nil {
			resource, err := value.Resource.Create(rg, metadata)
			if err == nil && resource != nil {
				value.Resource = resource
				rg.AddDependency(res, resource)

				if parent != nil {
					parent.SetMapIndex(index, reflect.ValueOf(value))
				} else {
					targetValue.Set(reflect.ValueOf(value))
				}
			} else {
				return err
			}
		}
	default:
		correspondingValue := targetValue
		for correspondingValue.Kind() == reflect.Pointer {
			correspondingValue = targetValue.Elem()
		}

		err := rg.checkChild(correspondingValue, res, metadata)
		if err != nil {
			return err
		}
	}
	return nil
}

func (rg *ResourceGraph) checkChild(child reflect.Value, res Resource, metadata map[string]any) error {
	var merr multierr.Error
	switch child.Kind() {
	case reflect.Struct:
		for i := 0; i < child.NumField(); i++ {
			childVal := child.Field(i)
			merr.Append(rg.actOnValue(childVal, res, metadata, nil, reflect.Value{}))
		}
	case reflect.Slice, reflect.Array:
		child.Type().Elem()
		for elemIdx := 0; elemIdx < child.Len(); elemIdx++ {
			elemValue := child.Index(elemIdx)
			merr.Append(rg.actOnValue(elemValue, res, metadata, nil, reflect.Value{}))
		}

	case reflect.Map:
		for _, key := range child.MapKeys() {
			elemValue := child.MapIndex(key)
			merr.Append(rg.actOnValue(elemValue, res, metadata, &child, key))
		}
	}
	return merr.ErrOrNil()
}
