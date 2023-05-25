package core

import (
	"fmt"
	"reflect"

	"github.com/klothoplatform/klotho/pkg/graph"
	"github.com/klothoplatform/klotho/pkg/multierr"
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
	rg.AddDependencyWithData(deployedSecond, deployedFirst, nil)
}

// AddDependencyWithData Adds a dependency such that `deployedSecond` has to be deployed after `deployedFirst`. This makes the left-to-right
// association consistent with our visualizer, and with the Go struct graph.
// This method also allows any edge data to be attached to the dependency in the ResourceGraph
func (rg *ResourceGraph) AddDependencyWithData(deployedSecond Resource, deployedFirst Resource, data any) {
	rg.AddResource(deployedSecond)
	rg.AddResource(deployedFirst)
	if cycle, _ := rg.underlying.CreatesCycle(deployedSecond.Id().String(), deployedFirst.Id().String()); cycle {
		zap.S().Errorf("Not Adding Dependency, Cycle would be created from edge %s -> %s", deployedSecond.Id(), deployedFirst.Id())
	} else {
		rg.underlying.AddEdge(deployedSecond.Id().String(), deployedFirst.Id().String(), data)
		zap.S().Debugf("adding %s -> %s", deployedSecond.Id(), deployedFirst.Id())
	}
}

func (rg *ResourceGraph) GetResource(id ResourceId) Resource {
	return rg.underlying.GetVertex(id.String())
}

func GetResource[T Resource](g *ResourceGraph, id ResourceId) (resource T, ok bool) {
	rR := g.GetResource(id)
	resource, ok = rR.(T)
	return
}

func (rg *ResourceGraph) GetDependency(source ResourceId, target ResourceId) *graph.Edge[Resource] {
	return rg.underlying.GetEdge(source.String(), target.String())
}

func (rg *ResourceGraph) RemoveDependency(source ResourceId, target ResourceId) error {
	return rg.underlying.RemoveEdge(source.String(), target.String())
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

func (rg *ResourceGraph) GetUpstreamDependencies(source Resource) []graph.Edge[Resource] {
	return rg.underlying.IncomingEdges(source)
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
	upstreamsSet := make(map[Resource]struct{})
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

// CreateDependencies takes in a resource and set of metadata and looks at any fields which point to specific dependencies which match the Resource Interface
// If a specific dependency is found, the .Create method will be called to create the parent resource.
// Each specific dependency value will be updated on the resources field itself
// This method will recurse down each one of the resources fields and do a DFS until a Resource is found
//
// Before adding a dependency and setting the value on the resource's field, the method will ensure there is no existing node in the graph for the named node created
// If there is an existing node, the method will ensure the resource's field is set to point to the already existent node
//
// IaCValues today are not labeled as a specific resource, so the value must already be present. This method will still add the dependency and ensure the resource is created based on the params passed in, but
// does not have the knowledge of which resource to create
//
//	T params are a map of the reflection field names, to their respective fields
//
// Example:
// for a struct:
//
//	 Test{
//	   Field1 Resource
//	   Field2 Resource
//	}
//
// The corresponding params would be:
//
//	params := map[string]any{
//	   "Field1": ResourceParams
//	   "Field2": ResourceParams
//	}
//
// Params for direct resources or IaCValues correlate to the field they are for
// For params on fields which correlate to the following types, the formats are:
//   - Map: map[string]ParamType    (the string key corresponds to the key in the map)
//   - Struct: map[string]ParamType (the string key corresponds to the field in the struct)
//   - Array or slice: []ParamType
func (rg *ResourceGraph) CreateDependencies(res Resource, params map[string]any) error {
	var merr multierr.Error
	source := reflect.ValueOf(res)

	for source.Kind() == reflect.Pointer {
		source = source.Elem()
	}
	for i := 0; i < source.NumField(); i++ {
		targetValue := source.Field(i)
		fieldsParams := params[source.Type().Field(i).Name]
		if fieldsParams != nil {
			merr.Append(rg.actOnValue(targetValue, res, fieldsParams, nil, reflect.Value{}))
		}
	}
	rg.AddDependenciesReflect(res)
	return merr.ErrOrNil()
}

// CreateResource is a wrapper around a Resources .Create method
//
// CreateResource provides safety in assuring that the up to date resource (which is inline with what exists in the ResourceGraph) is returned
func CreateResource[T Resource](rg *ResourceGraph, params any) (resource T, err error) {
	res := reflect.New(reflect.TypeOf(resource).Elem()).Interface()
	err = rg.callCreate(reflect.ValueOf(res), params)
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

func (rg *ResourceGraph) actOnValue(targetValue reflect.Value, res Resource, metadata any, parent *reflect.Value, index reflect.Value) error {
	switch value := targetValue.Interface().(type) {
	case Resource:
		if targetValue.IsNil() {
			value = reflect.New(targetValue.Type().Elem()).Interface().(Resource)
		}
		err := rg.callCreate(reflect.ValueOf(value), metadata)
		currValue := rg.GetResource(value.Id())
		if currValue != nil {
			value = currValue
		}
		if err == nil && value != nil {
			if parent != nil {
				parent.SetMapIndex(index, reflect.ValueOf(value))
			} else {
				targetValue.Set(reflect.ValueOf(value))
			}
		} else {
			return err
		}
	case *IaCValue:
		if value != nil && value.Resource != nil {
			err := rg.callCreate(reflect.ValueOf(value.Resource), metadata)
			currValue := rg.GetResource(value.Resource.Id())
			if currValue != nil {
				value.Resource = currValue
			}
			if err == nil && value.Resource != nil {
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
			err := rg.callCreate(reflect.ValueOf(value.Resource), metadata)
			currValue := rg.GetResource(value.Resource.Id())
			if currValue != nil {
				value.Resource = currValue
			}
			if err == nil && value.Resource != nil {
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

func (rg *ResourceGraph) callCreate(targetValue reflect.Value, metadata any) error {
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

func (rg *ResourceGraph) checkChild(child reflect.Value, res Resource, metadata any) error {
	var merr multierr.Error
	switch child.Kind() {
	case reflect.Struct:
		params := reflect.ValueOf(metadata)
		if params.Kind() != reflect.Map {
			return fmt.Errorf("field %s params does not conform to type for structs", child.Type().String())
		}
		for i := 0; i < child.NumField(); i++ {
			childVal := child.Field(i)
			fieldName := child.Type().Field(i).Name

			// Loop over the keys of the params map and see if anything correlates to the field in the struct. If so then we will act on that field of the struct
			for _, key := range params.MapKeys() {
				if key.String() == fieldName {
					merr.Append(rg.actOnValue(childVal, res, params.MapIndex(reflect.ValueOf(fieldName)).Interface(), nil, reflect.Value{}))
				}
			}
		}
	case reflect.Slice, reflect.Array:
		params := reflect.ValueOf(metadata)
		if params.Kind() != reflect.Slice && params.Kind() != reflect.Array {
			return fmt.Errorf("field %s does not match parent type %s", params.Type().String(), child.Type().String())
		} else if params.Len() != child.Len() {
			return fmt.Errorf("field %s does not have the same number of elements as parent", child.Type().String())
		}
		for elemIdx := 0; elemIdx < child.Len(); elemIdx++ {
			elemValue := child.Index(elemIdx)
			merr.Append(rg.actOnValue(elemValue, res, params.Index(elemIdx).Interface(), nil, reflect.Value{}))
		}
	case reflect.Map:
		params := reflect.ValueOf(metadata)
		if params.Kind() != reflect.Map {
			return fmt.Errorf("field %s params does not conform to type for maps", child.Type().String())
		}
		for _, key := range child.MapKeys() {
			elemValue := child.MapIndex(key)
			for _, paramKey := range params.MapKeys() {
				if key.String() == paramKey.String() {
					merr.Append(rg.actOnValue(elemValue, res, params.MapIndex(key).Interface(), &child, key))
				}
			}
		}
	}
	return merr.ErrOrNil()
}
