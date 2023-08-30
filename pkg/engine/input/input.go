package input

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	"github.com/klothoplatform/klotho/pkg/provider"
	"github.com/klothoplatform/klotho/pkg/yaml_util"
)

type (
	Input struct {
		AppName   string                                    `yaml:"app"`
		Resources map[construct.ResourceId]ResourceMetadata `yaml:"resources"`
		Edges     []construct.OutputEdge                    `yaml:"edges"`

		// Operations are an imperitive sequence of commands to make changes to the input graph.
		Operations []Operation `yaml:"operations"`

		// Constraints currently the old constraints system is used
		// NEXT VERSION: influence graph expansion by:
		// 1. Restricting the types of resources that a generic construct can be expanded into.
		// 2. Restricting the paths available during edge expansion.
		Constraints constraints.ConstraintList `yaml:"constraints"`

		// Configs are used to configure the resources after graph expansion is complete.
		Configs []Config `yaml:"config"`
	}

	ResourceMetadata struct {
		Id       construct.ResourceId `yaml:"id"`
		Metadata *yaml_util.RawNode   `yaml:"metadata"`
	}
)

func (i *Input) Load(providers map[string]provider.Provider) (*construct.ConstructGraph, error) {
	var joinedErr error
	dag := construct.NewConstructGraph()
	for node := range i.Resources {
		provider := providers[node.Provider]
		construct, err := provider.CreateConstructFromId(node, dag)
		if err != nil {
			joinedErr = errors.Join(joinedErr, fmt.Errorf("unable to find resource of type %s: %w", node.Type, err))
			continue
		}
		dag.AddConstruct(construct)
	}
	for _, metadata := range i.Resources {
		resource := dag.GetConstruct(metadata.Id)
		err := metadata.Metadata.Decode(resource)
		if err != nil {
			return nil, fmt.Errorf("could not decode metadata for resource %s: %w", metadata.Id, err)
		}
		err = correctPointers(resource, dag)
		if err != nil {
			return nil, err
		}
	}
	for _, edge := range i.Edges {
		dag.AddDependency(edge.Source, edge.Destination)
	}
	return dag, joinedErr
}

// correctPointers is used to ensure that the attributes of each baseconstruct points to the baseconstruct
// which exists in the graph by passing those in via a resource map.
func correctPointers(source construct.BaseConstruct, dag *construct.ConstructGraph) error {
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
				setNestedResourceFromId(source, elemValue, dag)
			}

		case reflect.Map:
			for iter := fieldValue.MapRange(); iter.Next(); {
				elemValue := iter.Value()
				setNestedResourceFromId(source, elemValue, dag)
			}

		default:
			setNestedResourceFromId(source, fieldValue, dag)
		}
	}
	return nil
}

var baseConstructInterface = reflect.TypeOf((*construct.BaseConstruct)(nil)).Elem()

// setNestedResourcesFromIds looks at attributes of a base construct which correspond to resources and sets the field
// to be the construct which exists in the resource map, based on the id which exists in the field currently.
func setNestedResourceFromId(source construct.BaseConstruct, targetField reflect.Value, dag *construct.ConstructGraph) {
	if targetField.Kind() == reflect.Pointer && targetField.IsNil() {
		return
	}
	if !targetField.CanInterface() {
		return
	}

	if targetField.Type().Implements(baseConstructInterface) {
		baseConstruct := targetField.Interface().(construct.BaseConstruct)
		dagRef := dag.GetConstruct(baseConstruct.Id())
		if dagRef != nil && targetField.IsValid() && targetField.CanSet() {
			targetField.Set(reflect.ValueOf(dagRef))
		}
		return
	}

	switch targetField.Interface().(type) {
	case construct.IaCValue:
		// fields are already set and have no subfields to process

	default:
		correspondingValue := targetField
		for correspondingValue.Kind() == reflect.Pointer {
			correspondingValue = targetField.Elem()
		}
		switch correspondingValue.Kind() {
		case reflect.Struct:
			for i := 0; i < correspondingValue.NumField(); i++ {
				childVal := correspondingValue.Field(i)
				setNestedResourceFromId(source, childVal, dag)
			}
		case reflect.Slice, reflect.Array:
			for elemIdx := 0; elemIdx < correspondingValue.Len(); elemIdx++ {
				elemValue := correspondingValue.Index(elemIdx)
				setNestedResourceFromId(source, elemValue, dag)
			}

		case reflect.Map:
			for iter := correspondingValue.MapRange(); iter.Next(); {
				elemValue := iter.Value()
				setNestedResourceFromId(source, elemValue, dag)
			}
		}
	}
}
