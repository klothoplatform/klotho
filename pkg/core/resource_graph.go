package core

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/klothoplatform/klotho/pkg/graph"
	"github.com/pkg/errors"
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

func (rg *ResourceGraph) Clone() *ResourceGraph {
	newGraph := &ResourceGraph{
		underlying: graph.NewLike(rg.underlying),
	}
	for _, v := range rg.ListResources() {
		newGraph.AddResource(v)
	}
	for _, dep := range rg.ListDependencies() {
		newGraph.AddDependencyWithData(dep.Source, dep.Destination, dep.Properties.Data)
	}
	return newGraph
}

func (rg *ResourceGraph) String() string {
	buf := new(strings.Builder)

	nodes := rg.ListResources()
	node_ids := make([]string, len(nodes))
	for _, node := range nodes {
		node_ids = append(node_ids, node.Id().String())
	}
	sort.Strings(node_ids)

	deps := rg.ListDependencies()
	dep_ids := make([]string, len(deps))
	for _, dep := range deps {
		dep_ids = append(dep_ids, fmt.Sprintf("%s -> %s", dep.Source.Id(), dep.Destination.Id()))
	}
	sort.Strings(dep_ids)

	for _, v := range node_ids {
		buf.WriteString(v)
	}
	for _, dep := range dep_ids {
		buf.WriteString(dep)
	}
	return buf.String()
}

func (rg *ResourceGraph) ShortestPath(source ResourceId, dest ResourceId) ([]Resource, error) {
	ids, err := rg.underlying.ShortestPath(source.String(), dest.String())
	if err != nil {
		return nil, err
	}
	resources := make([]Resource, len(ids))
	for i, id := range ids {
		resources[i] = rg.underlying.GetVertex(id)
	}
	return resources, nil
}

func (rg *ResourceGraph) AllPaths(source ResourceId, dest ResourceId) ([][]Resource, error) {
	paths, err := rg.underlying.AllPaths(source.String(), dest.String())
	if err != nil {
		return nil, err
	}
	resources := make([][]Resource, len(paths))
	for i, path := range paths {
		resources[i] = make([]Resource, len(path))
		for j, id := range path {
			resources[i][j] = rg.underlying.GetVertex(id)
		}
	}
	return resources, nil
}

func (rg *ResourceGraph) AddResource(resource Resource) {
	if rg.GetResource(resource.Id()) == nil {
		rg.underlying.AddVertex(resource)
		zap.S().Debugf("adding resource: %s", resource.Id())
	}
}

func (rg *ResourceGraph) AddResourceWithProperties(resource Resource, properties map[string]string) {
	rg.underlying.AddVertexWithProperties(resource, graph.ToVertexAttributes(properties))
	zap.S().Debugf("adding resource: %s, with properties: %s", resource.Id(), properties)
}

func (rg *ResourceGraph) GetResource(id ResourceId) Resource {
	return rg.underlying.GetVertex(id.String())
}

func (rg *ResourceGraph) GetResourceFromString(id string) Resource {
	return rg.underlying.GetVertex(id)
}

func (rg *ResourceGraph) GetResourceWithProperties(id ResourceId) (Resource, map[string]string) {
	res, props := rg.underlying.GetVertexWithProperties(id.String())
	return res, graph.AttributesFromVertexProperties(props)
}

// Adds a dependency such that `source` has to be deployed after `destination`. This makes the left-to-right
// association consistent with our visualizer, and with the Go struct graph.
//
// For example, if you have a Lambda and its execution role, then:
//
//	╭────────────────╮   ╭────────────────╮
//	│ LambdaFunction ├──➤│    IamRole     │
//	│ source │   │ destination  │
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
func (rg *ResourceGraph) AddDependency(source Resource, destination Resource) {
	rg.AddDependencyWithData(source, destination, nil)
}

// AddDependencyWithData Adds a dependency such that `source` has to be deployed after `destination`. This makes the left-to-right
// association consistent with our visualizer, and with the Go struct graph.
// This method also allows any edge data to be attached to the dependency in the ResourceGraph
func (rg *ResourceGraph) AddDependencyWithData(source Resource, destination Resource, data any) {
	if source.Id() == destination.Id() {
		return
	}
	rg.AddResource(source)
	rg.AddResource(destination)
	rg.AddDependencyById(source.Id(), destination.Id(), data)

}

func (rg *ResourceGraph) AddDependencyById(source ResourceId, destination ResourceId, data any) {
	if cycle, _ := rg.underlying.CreatesCycle(source.String(), destination.String()); cycle {
		zap.S().Errorf("Not Adding Dependency, Cycle would be created from edge %s -> %s", source, destination)
	} else {
		rg.underlying.AddEdge(source.String(), destination.String(), data)
		zap.S().Debugf("adding %s -> %s", source, destination)
	}
}

func (rg *ResourceGraph) AddDependencyByString(source string, destination string, data any) {
	if cycle, _ := rg.underlying.CreatesCycle(source, destination); cycle {
		zap.S().Errorf("Not Adding Dependency, Cycle would be created from edge %s -> %s", source, destination)
	} else {
		rg.underlying.AddEdge(source, destination, data)
		zap.S().Debugf("adding %s -> %s", source, destination)
	}
}

func GetResource[T Resource](g *ResourceGraph, id ResourceId) (resource T, ok bool) {
	rR := g.GetResource(id)
	resource, ok = rR.(T)
	return
}

func GetResources[T Resource](g *ResourceGraph) (resources []T) {
	for _, res := range g.ListResources() {
		if r, ok := res.(T); ok {
			resources = append(resources, r)
		}
	}
	return
}

func (rg *ResourceGraph) FindResourcesWithRef(id ResourceId) []Resource {
	var result []Resource
	for _, resource := range rg.ListResources() {
		if resource.BaseConstructRefs().Has(id) {
			result = append(result, resource)
		}
	}
	return result
}

func (rg *ResourceGraph) GetDependency(source ResourceId, target ResourceId) *graph.Edge[Resource] {
	return rg.underlying.GetEdge(source.String(), target.String())
}

func (rg *ResourceGraph) RemoveDependency(source ResourceId, target ResourceId) error {
	return rg.underlying.RemoveEdge(source.String(), target.String())
}

func (rg *ResourceGraph) RemoveResourceAndEdges(source Resource) error {
	for _, edge := range rg.GetDownstreamDependencies(source) {
		err := rg.RemoveDependency(edge.Source.Id(), edge.Destination.Id())
		if err != nil {
			return err
		}
	}
	for _, edge := range rg.GetUpstreamDependencies(source) {
		err := rg.RemoveDependency(edge.Source.Id(), edge.Destination.Id())
		if err != nil {
			return err
		}
	}
	return rg.RemoveResource(source)
}

func (rg *ResourceGraph) RemoveResource(resource Resource) error {
	zap.S().Debugf("Removing resource %s", resource.Id())
	return rg.underlying.RemoveVertex(resource.Id().String())
}

func (rg *ResourceGraph) ListResources() []Resource {
	return rg.underlying.GetAllVertices()
}

func (rg *ResourceGraph) ListDependencies() []graph.Edge[Resource] {
	return rg.underlying.GetAllEdges()
}

func (rg *ResourceGraph) GetDownstreamDependencies(source Resource) []graph.Edge[Resource] {
	return rg.underlying.OutgoingEdges(source)
}

func (rg *ResourceGraph) GetDownstreamResources(source Resource) []Resource {
	return rg.underlying.OutgoingVertices(source)
}

func GetDownstreamResourcesOfType[T Resource](rg *ResourceGraph, source Resource) (resources []T) {
	for _, res := range rg.underlying.OutgoingVertices(source) {
		if r, ok := res.(T); ok {
			resources = append(resources, r)
		}
	}
	return
}

func (rg *ResourceGraph) GetUpstreamDependencies(source Resource) []graph.Edge[Resource] {
	return rg.underlying.IncomingEdges(source)
}

func GetUpstreamResourcesOfType[T Resource](rg *ResourceGraph, source Resource) (resources []T) {
	for _, res := range rg.underlying.IncomingVertices(source) {
		if r, ok := res.(T); ok {
			resources = append(resources, r)
		}
	}
	return
}

func (rg *ResourceGraph) GetUpstreamResources(source Resource) []Resource {
	return rg.underlying.IncomingVertices(source)
}

func (rg *ResourceGraph) TopologicalSort() ([]Resource, error) {
	ids, err := rg.underlying.VertexIdsInTopologicalOrder()
	if err != nil {
		return nil, err
	}
	resources := make([]Resource, len(ids))
	for i, id := range ids {
		resources[i] = rg.underlying.GetVertex(id)
	}
	return resources, nil
}

func (rg *ResourceGraph) ReverseTopologicalSort() ([]Resource, error) {
	ids, err := rg.underlying.VertexIdsInTopologicalOrder()
	if err != nil {
		return nil, err
	}
	total := len(ids)
	resources := make([]Resource, total)
	for i := total; i > 0; i-- {
		resources[total-i] = rg.underlying.GetVertex(ids[i-1])
	}
	return resources, nil
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

		if sourceValue.Field(i).Type().Name() == "BaseConstructSet" {
			continue
		}
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
	case IaCValue:
		if !value.ResourceId.IsZero() {
			rg.AddDependencyById(source.Id(), value.ResourceId, nil)
		}
	case ResourceId:
		if !value.IsZero() {
			rg.AddDependencyById(source.Id(), value, nil)
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
	upstreamsSet := make(map[Resource]struct{})
	for r := range rg.getAllUpstreamResourcesSet(source, upstreamsSet) {
		upstreams = append(upstreams, r)
	}
	return upstreams
}

func GetAllUpstreamResourcesOfType[T Resource](rg *ResourceGraph, source Resource) (resources []T) {
	upstreamsSet := map[Resource]struct{}{}
	for r := range rg.getAllUpstreamResourcesSet(source, upstreamsSet) {
		if rT, ok := r.(T); ok {
			resources = append(resources, rT)
		}
	}
	return
}

func GetSingleUpstreamResourceOfType[T Resource](rg *ResourceGraph, source Resource) (resource T, err error) {
	resources := GetAllUpstreamResourcesOfType[T](rg, source)
	if len(resources) == 0 {
		return resource, errors.Errorf("no upstream resource of type %T found for resource %s", source, source.Id())
	} else if len(resources) > 1 {
		return resource, errors.Errorf("multiple upstream resources of type %T found for resource %s", source, source.Id())
	}
	return resources[0], nil

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

func GetAllDownstreamResourcesOfType[T Resource](rg *ResourceGraph, source Resource) (resources []T) {
	upstreamsSet := map[Resource]struct{}{}
	for r := range rg.getAllDownstreamResourcesSet(source, upstreamsSet) {
		if rT, ok := r.(T); ok {
			resources = append(resources, rT)
		}
	}
	return
}

func GetSingleDownstreamResourceOfType[T Resource](rg *ResourceGraph, source Resource) (resource T, err error) {
	resources := GetAllDownstreamResourcesOfType[T](rg, source)
	if len(resources) == 0 {
		return resource, errors.Errorf("no downstream resource of type %T found for resource %s", source, source.Id())
	} else if len(resources) > 1 {
		return resource, errors.Errorf("multiple downstream resources of type %T found for resource %s", resources[0], source.Id())
	}
	return resources[0], nil

}

func (rg *ResourceGraph) ReplaceConstruct(resource Resource, new Resource) error {
	rg.AddResource(new)
	// Since its a construct we just assume every single edge can be removed
	for _, edge := range rg.GetDownstreamDependencies(resource) {
		rg.AddDependency(new, edge.Destination)
		err := rg.RemoveDependency(edge.Source.Id(), edge.Destination.Id())
		if err != nil {
			return err
		}
	}
	for _, edge := range rg.GetUpstreamDependencies(resource) {
		rg.AddDependency(edge.Source, new)
		err := rg.RemoveDependency(edge.Source.Id(), edge.Destination.Id())
		if err != nil {
			return err
		}
	}
	return rg.RemoveResource(resource)
}

// CreateResource is a wrapper around a Resources .Create method
//
// CreateResource provides safety in assuring that the up to date resource (which is inline with what exists in the ResourceGraph) is returned
func CreateResource[T Resource](rg *ResourceGraph, params any) (resource T, err error) {
	res := reflect.New(reflect.TypeOf(resource).Elem()).Interface()
	err = rg.CallCreate(reflect.ValueOf(res), params)
	if err != nil {
		return
	}
	castedRes, ok := res.(Resource)
	if !ok {
		err = fmt.Errorf("unable to cast to type Resource")
		return
	}

	currValue := rg.GetResource(castedRes.Id())
	if currValue != nil {
		return currValue.(T), nil
	}
	return castedRes.(T), nil
}

func (rg *ResourceGraph) CallCreate(targetValue reflect.Value, metadata any) error {
	method := targetValue.MethodByName("Create")
	if method.IsValid() {
		var callArgs []reflect.Value
		callArgs = append(callArgs, reflect.ValueOf(rg))
		params := reflect.New(method.Type().In(1)).Interface()
		decoder := GetMapDecoder(params)
		err := decoder.Decode(metadata)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("error decoding the following type %s", reflect.New(method.Type().In(1)).Type().String()))
		}
		callArgs = append(callArgs, reflect.ValueOf(params).Elem())
		eval := method.Call(callArgs)
		if eval[0].IsNil() {
			return nil
		} else {
			err, ok := eval[0].Interface().(error)
			if !ok {
				return fmt.Errorf("return type should be an error")
			}
			return err
		}
	}
	return nil
}

// CallConfigure uses the resource graph to ensure the node passed in exists, then uses reflection to call the resources Configure method
func (rg *ResourceGraph) CallConfigure(resource Resource, metadata any) error {
	if rg.GetResource(resource.Id()) == nil {
		return fmt.Errorf("resource with id %s cannot be configured since it does not exist in the ResourceGraph", resource.Id())
	}

	method := reflect.ValueOf(resource).MethodByName("Configure")
	if method.IsValid() {
		var callArgs []reflect.Value
		params := reflect.New(method.Type().In(0)).Interface()
		decoder := GetMapDecoder(params)
		err := decoder.Decode(metadata)
		if err != nil {
			return errors.Wrapf(err, fmt.Sprintf("error decoding the following type %s", reflect.New(method.Type().In(0)).Type()))
		}
		callArgs = append(callArgs, reflect.ValueOf(params).Elem())
		eval := method.Call(callArgs)
		if eval[0].IsNil() {
			return nil
		} else {
			err, ok := eval[0].Interface().(error)
			if !ok {
				return fmt.Errorf("return type should be an error")
			}
			return err
		}
	}
	return nil
}
