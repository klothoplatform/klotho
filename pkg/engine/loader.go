package engine

import (
	j_errors "errors"
	"fmt"
	"os"
	"reflect"

	"github.com/klothoplatform/klotho/pkg/core"

	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// LoadConstructGraphFromFile takes in a path to a file and loads in all of the BaseConstructs and edges which exist in the file.
func (e *Engine) LoadConstructGraphFromFile(path string) error {
	resourcesMap := map[core.ResourceId]core.BaseConstruct{}
	input := core.InputGraph{}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close() // nolint:errcheck
	err = yaml.NewDecoder(f).Decode(&input)
	if err != nil {
		return err
	}
	err = e.loadConstructs(input, resourcesMap)
	if err != nil {
		return errors.Errorf("Error Loading graph for constructs %s", err.Error())
	}
	err = e.LoadResources(input, resourcesMap)
	if err != nil {
		return errors.Errorf("Error Loading graph for providers. %s", err.Error())
	}
	for _, metadata := range input.ResourceMetadata {
		resource := resourcesMap[metadata.Id]
		md, err := yaml.Marshal(metadata.Metadata)
		if err != nil {
			return err
		}
		err = yaml.Unmarshal(md, resource)
		if err != nil {
			return err
		}
		err = correctPointers(resource, resourcesMap)
		if err != nil {
			return err
		}
	}
	for _, res := range resourcesMap {
		e.Context.InitialState.AddConstruct(res)
	}

	for _, edge := range input.Edges {
		e.Context.InitialState.AddDependency(resourcesMap[edge.Source].Id(), resourcesMap[edge.Destination].Id())
	}

	return nil
}

func (e *Engine) LoadResources(graph core.InputGraph, resourcesMap map[core.ResourceId]core.BaseConstruct) error {
	var joinedErr error
	for _, node := range graph.Resources {
		if node.Provider == core.AbstractConstructProvider {
			continue
		}
		provider := e.Providers[node.Provider]
		typeToResource := make(map[string]core.Resource)
		for _, res := range provider.ListResources() {
			typeToResource[res.Id().Type] = res
		}
		res, ok := typeToResource[node.Type]
		if !ok {
			joinedErr = j_errors.Join(joinedErr, fmt.Errorf("unable to find resource of type %s", node.Type))
			continue
		}
		newResource := reflect.New(reflect.TypeOf(res).Elem()).Interface()
		resource, ok := newResource.(core.Resource)
		if !ok {
			joinedErr = j_errors.Join(joinedErr, fmt.Errorf("item %s of type %T is not of type core.Resource", node, newResource))
			continue
		}
		reflect.ValueOf(resource).Elem().FieldByName("Name").SetString(node.Name)
		resourcesMap[node] = resource
	}
	return joinedErr
}

func (e *Engine) loadConstructs(input core.InputGraph, resourceMap map[core.ResourceId]core.BaseConstruct) error {

	var joinedErr error
	for _, res := range input.Resources {
		if res.Provider != core.AbstractConstructProvider {
			continue
		}
		construct, err := e.getConstructFromInputId(res)
		if err != nil {
			joinedErr = j_errors.Join(joinedErr, err)
			continue
		}
		resourceMap[construct.Id()] = construct
	}

	return joinedErr
}

func (e *Engine) getConstructFromInputId(res core.ResourceId) (core.Construct, error) {
	typeToResource := make(map[string]core.Construct)
	for _, construct := range e.Constructs {
		typeToResource[construct.Id().Type] = construct
	}
	construct, ok := typeToResource[res.Type]
	if !ok {
		return nil, fmt.Errorf("unable to find resource of type %s", res.Type)
	}
	newConstruct := reflect.New(reflect.TypeOf(construct).Elem()).Interface()
	construct, ok = newConstruct.(core.Construct)
	if !ok {
		return nil, fmt.Errorf("item %s of type %T is not of type core.Resource", res, newConstruct)
	}
	reflect.ValueOf(construct).Elem().FieldByName("Name").SetString(res.Name)
	return construct, nil
}

func (e *Engine) LoadConstraintsFromFile(path string) (map[constraints.ConstraintScope][]constraints.Constraint, error) {

	type Input struct {
		Constraints []any             `yaml:"constraints"`
		Resources   []core.ResourceId `yaml:"resources"`
		Edges       []core.OutputEdge `yaml:"edges"`
	}

	input := Input{}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close() // nolint:errcheck

	err = yaml.NewDecoder(f).Decode(&input)
	if err != nil {
		return nil, err
	}

	bytesArr, err := yaml.Marshal(input.Constraints)
	if err != nil {
		return nil, err
	}
	return constraints.ParseConstraintsFromFile(bytesArr)
}

// correctPointers is used to ensure that the attributes of each baseconstruct points to the baseconstruct which exists in the graph by passing those in via a resource map.
func correctPointers(source core.BaseConstruct, resourceMap map[core.ResourceId]core.BaseConstruct) error {
	sourceValue := reflect.ValueOf(source)
	sourceType := sourceValue.Type()
	if sourceType.Kind() == reflect.Pointer {
		sourceValue = sourceValue.Elem()
		sourceType = sourceType.Elem()
	}
	for i := 0; i < sourceType.NumField(); i++ {
		fieldValue := sourceValue.Field(i)
		switch fieldValue.Kind() {
		case reflect.Slice, reflect.Array:
			for elemIdx := 0; elemIdx < fieldValue.Len(); elemIdx++ {
				elemValue := fieldValue.Index(elemIdx)
				setNestedResourceFromId(source, elemValue, resourceMap)
			}

		case reflect.Map:
			for iter := fieldValue.MapRange(); iter.Next(); {
				elemValue := iter.Value()
				setNestedResourceFromId(source, elemValue, resourceMap)
			}

		default:
			setNestedResourceFromId(source, fieldValue, resourceMap)
		}
	}
	return nil
}

// setNestedResourcesFromIds looks at attributes of a base construct which correspond to resources and sets the field to be the construct which exists in the resource map,
//
//	based on the id which exists in the field currently.
func setNestedResourceFromId(source core.BaseConstruct, targetValue reflect.Value, resourceMap map[core.ResourceId]core.BaseConstruct) {
	if targetValue.Kind() == reflect.Pointer && targetValue.IsNil() {
		return
	}
	if !targetValue.CanInterface() {
		return
	}
	switch value := targetValue.Interface().(type) {
	case core.Resource:
		targetValue.Set(reflect.ValueOf(resourceMap[value.Id()]))
	case core.IaCValue:
		// fields are already set and have no subfields to process
	default:
		correspondingValue := targetValue
		for correspondingValue.Kind() == reflect.Pointer {
			correspondingValue = targetValue.Elem()
		}
		switch correspondingValue.Kind() {

		case reflect.Struct:
			for i := 0; i < correspondingValue.NumField(); i++ {
				childVal := correspondingValue.Field(i)
				setNestedResourceFromId(source, childVal, resourceMap)
			}
		case reflect.Slice, reflect.Array:
			for elemIdx := 0; elemIdx < correspondingValue.Len(); elemIdx++ {
				elemValue := correspondingValue.Index(elemIdx)
				setNestedResourceFromId(source, elemValue, resourceMap)
			}

		case reflect.Map:
			for iter := correspondingValue.MapRange(); iter.Next(); {
				elemValue := iter.Value()
				setNestedResourceFromId(source, elemValue, resourceMap)
			}

		}
	}
}
