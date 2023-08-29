package graph_loader

import (
	j_errors "errors"
	"fmt"
	"os"
	"reflect"

	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/provider"
	"github.com/klothoplatform/klotho/pkg/provider/aws"
	"github.com/klothoplatform/klotho/pkg/provider/docker"
	"github.com/klothoplatform/klotho/pkg/provider/kubernetes"
	"github.com/klothoplatform/klotho/pkg/yaml_util"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type (
	inputMetadata struct {
		Id       construct.ResourceId `yaml:"id"`
		Metadata *yaml_util.RawNode   `yaml:"metadata"`
	}
	inputGraph struct {
		Resources        []construct.ResourceId `yaml:"resources"`
		ResourceMetadata []inputMetadata        `yaml:"resourceMetadata"`
		Edges            []construct.OutputEdge `yaml:"edges"`
	}
)

func loadProviders() map[string]provider.Provider {
	providerMap := map[string]provider.Provider{
		"aws":        &aws.AWS{},
		"kubernetes": &kubernetes.KubernetesProvider{},
		"docker":     &docker.DockerProvider{},
	}
	return providerMap
}

// LoadConstructGraphFromFile takes in a path to a file and loads in all of the BaseConstructs and edges which exist in the file.
func LoadConstructGraphFromFile(path string) (*construct.ConstructGraph, error) {
	graph := construct.NewConstructGraph()

	resourcesMap := map[construct.ResourceId]construct.BaseConstruct{}
	var input inputGraph
	f, err := os.Open(path)
	if err != nil {
		return graph, err
	}
	defer f.Close() // nolint:errcheck
	err = yaml.NewDecoder(f).Decode(&input)
	if err != nil {
		return graph, err
	}
	err = loadConstructs(input.Resources, resourcesMap)
	if err != nil {
		return graph, errors.Errorf("Error Loading graph for constructs %s", err.Error())
	}
	err = loadResources(input.Resources, resourcesMap, loadProviders())
	if err != nil {
		return graph, errors.Errorf("Error Loading graph for providers. %s", err.Error())
	}
	for _, metadata := range input.ResourceMetadata {
		resource := resourcesMap[metadata.Id]
		err = metadata.Metadata.Decode(resource)
		if err != nil {
			return graph, err
		}
		err = correctPointers(resource, resourcesMap)
		if err != nil {
			return graph, err
		}
	}
	for _, res := range resourcesMap {
		graph.AddConstruct(res)
	}

	for _, edge := range input.Edges {
		graph.AddDependency(resourcesMap[edge.Source].Id(), resourcesMap[edge.Destination].Id())
	}

	return graph, nil
}

// LoadResourceGraphFromFile takes in a path to a file and loads in all of the Resources and edges which exist in the file.
func LoadResourceGraphFromFile(path string) (*construct.ResourceGraph, error) {
	joinedErr := error(nil)
	graph := construct.NewResourceGraph()

	resourcesMap := map[construct.ResourceId]construct.BaseConstruct{}
	var input inputGraph
	f, err := os.Open(path)
	if err != nil {
		return graph, err
	}
	defer f.Close() // nolint:errcheck
	err = yaml.NewDecoder(f).Decode(&input)
	if err != nil {
		return graph, err
	}
	err = loadConstructs(input.Resources, resourcesMap)
	if err != nil {
		return graph, errors.Errorf("Error Loading graph for constructs %s", err.Error())
	}
	err = loadResources(input.Resources, resourcesMap, loadProviders())
	if err != nil {
		return graph, errors.Errorf("Error Loading graph for providers. %s", err.Error())
	}
	for _, metadata := range input.ResourceMetadata {
		resource := resourcesMap[metadata.Id]
		err = metadata.Metadata.Decode(resource)
		if err != nil {
			return graph, err
		}
		err = correctPointers(resource, resourcesMap)
		if err != nil {
			return graph, err
		}
	}
	for _, res := range resourcesMap {
		resource, ok := res.(construct.Resource)
		if !ok {
			joinedErr = j_errors.Join(joinedErr, fmt.Errorf("%s is not a resource", res.Id()))
			continue
		}
		graph.AddResource(resource)
	}

	for _, edge := range input.Edges {
		src, ok := resourcesMap[edge.Source].(construct.Resource)
		if !ok {
			joinedErr = j_errors.Join(joinedErr, fmt.Errorf("%s is not a resource", src.Id()))
			continue
		}
		dst, ok := resourcesMap[edge.Destination].(construct.Resource)
		if !ok {
			joinedErr = j_errors.Join(joinedErr, fmt.Errorf("%s is not a resource", dst.Id()))
			continue
		}
		graph.AddDependency(src, dst)
	}

	return graph, joinedErr
}

func loadResources(resources []construct.ResourceId, resourcesMap map[construct.ResourceId]construct.BaseConstruct, providers map[string]provider.Provider) error {
	var joinedErr error
	for _, node := range resources {
		if node.Provider == construct.AbstractConstructProvider {
			continue
		}
		provider := providers[node.Provider]
		typeToResource := make(map[string]construct.Resource)
		for _, res := range provider.ListResources() {
			typeToResource[res.Id().Type] = res
		}
		res, ok := typeToResource[node.Type]
		if !ok {
			joinedErr = j_errors.Join(joinedErr, fmt.Errorf("unable to find resource of type %s", node.Type))
			continue
		}
		newResource := reflect.New(reflect.TypeOf(res).Elem()).Interface()
		resource, ok := newResource.(construct.Resource)
		if !ok {
			joinedErr = j_errors.Join(joinedErr, fmt.Errorf("item %s of type %T is not of type construct.Resource", node, newResource))
			continue
		}
		reflect.ValueOf(resource).Elem().FieldByName("Name").SetString(node.Name)
		resourcesMap[node] = resource
	}
	return joinedErr
}

func loadConstructs(resources []construct.ResourceId, resourceMap map[construct.ResourceId]construct.BaseConstruct) error {

	var joinedErr error
	for _, res := range resources {
		if res.Provider != construct.AbstractConstructProvider {
			continue
		}
		construct, err := GetConstructFromInputId(res)
		if err != nil {
			joinedErr = j_errors.Join(joinedErr, err)
			continue
		}
		resourceMap[construct.Id()] = construct
	}

	return joinedErr
}

func GetConstructFromInputId(res construct.ResourceId) (construct.Construct, error) {
	typeToResource := make(map[string]construct.Construct)
	for _, construct := range types.ListAllConstructs() {
		typeToResource[construct.Id().Type] = construct
	}
	c, ok := typeToResource[res.Type]
	if !ok {
		return nil, fmt.Errorf("unable to find resource of type %s", res.Type)
	}
	newConstruct := reflect.New(reflect.TypeOf(c).Elem()).Interface()
	c, ok = newConstruct.(construct.Construct)
	if !ok {
		return nil, fmt.Errorf("item %s of type %T is not of type construct.Resource", res, newConstruct)
	}
	reflect.ValueOf(c).Elem().FieldByName("Name").SetString(res.Name)
	return c, nil
}

// correctPointers is used to ensure that the attributes of each baseconstruct points to the baseconstruct which exists in the graph by passing those in via a resource map.
func correctPointers(source construct.BaseConstruct, resourceMap map[construct.ResourceId]construct.BaseConstruct) error {
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
func setNestedResourceFromId(source construct.BaseConstruct, targetField reflect.Value, resourceMap map[construct.ResourceId]construct.BaseConstruct) {
	if targetField.Kind() == reflect.Pointer && targetField.IsNil() {
		return
	}
	if !targetField.CanInterface() {
		return
	}
	switch value := targetField.Interface().(type) {
	case construct.Resource:
		targetValue := reflect.ValueOf(resourceMap[value.Id()])
		if targetField.IsValid() && targetField.CanSet() && targetValue.IsValid() {
			targetField.Set(targetValue)
		}
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
