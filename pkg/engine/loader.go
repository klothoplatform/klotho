package engine

import (
	"os"
	"reflect"

	"github.com/klothoplatform/klotho/pkg/core"

	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

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

	err = core.LoadConstructs(input, resourcesMap)
	if err != nil {
		return errors.Errorf("Error Loading graph for constructs %s", err.Error())
	}

	err = e.Provider.LoadResources(input, resourcesMap)
	if err != nil {
		return errors.Errorf("Error Loading graph for provider %s. %s", e.Provider.Name(), err.Error())
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
		if value.Resource() != nil {
			value.SetResource(resourceMap[value.Resource().Id()].(core.Resource))
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
