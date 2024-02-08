package kubernetes

import (
	"fmt"
	"math/rand"
	"reflect"
	"strconv"

	construct "github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/set"
	"gopkg.in/yaml.v3"
)

type (
	ObjectOutput struct {
		Content []byte
		Values  map[string]construct.PropertyRef
	}
)

var excludedObjects = []construct.ResourceId{
	{Provider: "kubernetes", Type: "helm_chart"},
	{Provider: "kubernetes", Type: "kustomize_directory"},
	{Provider: "kubernetes", Type: "manifest"},
	{Provider: "kubernetes", Type: "kube_config"},
}

func includeObjectInChart(res construct.ResourceId) bool {
	for _, id := range excludedObjects {
		if id.Matches(res) {
			return false
		}
	}
	return true
}

func AddObject(res *construct.Resource) (*ObjectOutput, error) {
	object, err := res.GetProperty("Object")
	if err != nil {
		return nil, fmt.Errorf("unable to find object property on resource %s: %w", res.ID, err)
	}
	output := &ObjectOutput{
		Values: make(map[string]construct.PropertyRef),
	}
	converted, err := output.convertObject(object)
	if err != nil {
		return nil, fmt.Errorf("unable to convert object property on resource %s: %w", res.ID, err)
	}
	content, err := yaml.Marshal(converted)
	if err != nil {
		return output, fmt.Errorf("unable to marshal object property on resource %s: %w", res.ID, err)
	}
	output.Content = content
	return output, nil
}

func (h ObjectOutput) convertObject(arg any) (any, error) {
	switch arg := arg.(type) {
	case construct.ResourceId:
		if arg.Provider != "kubernetes" {
			return nil, fmt.Errorf("resource %s is not a kubernetes resource", arg)
		}
		return arg.Name, nil

	case construct.PropertyRef:
		valuesString := generateStringSuffix(5)
		h.Values[valuesString] = arg
		return fmt.Sprintf("{{ .Values.%s }}", valuesString), nil

	case string:
		// use templateString to quote the string value

		return templateString(arg), nil

	case bool, int, float64:
		// safe to use as-is
		return arg, nil

	case nil:
		// don't add to inputs
		return nil, nil

	default:
		switch val := reflect.ValueOf(arg); val.Kind() {
		case reflect.Slice, reflect.Array:
			yamlList := []any{}
			for i := 0; i < val.Len(); i++ {
				if !val.Index(i).IsValid() || val.Index(i).IsNil() {
					continue
				}
				output, err := h.convertObject(val.Index(i).Interface())
				if err != nil {
					return "", err
				}
				yamlList = append(yamlList, output)
			}
			return yamlList, nil
		case reflect.Map:
			yamlMap := make(map[string]any)
			for _, key := range val.MapKeys() {
				if !val.MapIndex(key).IsValid() || val.MapIndex(key).IsNil() {
					continue
				}
				keyStr, found := key.Interface().(string)
				if !found {
					return "", fmt.Errorf("map key is not a string")
				}
				output, err := h.convertObject(val.MapIndex(key).Interface())
				if err != nil {
					return "", err
				}
				yamlMap[keyStr] = output
			}
			return yamlMap, nil
		case reflect.Struct:
			if hashset, ok := val.Interface().(set.HashedSet[string, any]); ok {
				return h.convertObject(hashset.ToSlice())
			}
			fallthrough
		default:
			return nil, fmt.Errorf("unable to convert arg %v to yaml", arg)
		}
	}

}

type templateString string

func (s templateString) String() string {
	return strconv.Quote(string(s))
}

func generateStringSuffix(n int) string {
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)

}
