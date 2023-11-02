package kubernetes

// import (
// 	"fmt"

// 	construct "github.com/klothoplatform/klotho/pkg/construct2"
// 	kio "github.com/klothoplatform/klotho/pkg/io"
// )

// type (
// 	HelmChartOutput struct {
// 		Name           string
// 		Files          []kio.File
// 		ProviderValues map[string]construct.PropertyRef
// 	}
// )

// var includedChartObjects = []construct.ResourceId{
// 	{Provider: "kubernetes", Type: "pod"},
// 	{Provider: "kubernetes", Type: "service"},
// 	{Provider: "kubernetes", Type: "deployment"},
// 	{Provider: "kubernetes", Type: "target_group_binding"},
// 	{Provider: "kubernetes", Type: "cluster_set"},
// 	{Provider: "kubernetes", Type: "config_map"},
// 	{Provider: "kubernetes", Type: "horizontal_pod_autoscaler"},
// 	{Provider: "kubernetes", Type: "storage_class"},
// 	{Provider: "kubernetes", Type: "persistent_volume_claim"},
// 	{Provider: "kubernetes", Type: "persistent_volume"},
// 	{Provider: "kubernetes", Type: "service_account"},
// 	{Provider: "kubernetes", Type: "namespace"},
// 	{Provider: "kubernetes", Type: "service_export"},
// }

// func (h HelmChartOutput) AddObject(res construct.Resource) error {
// 	shouldInclude := false
// 	for _, id := range includedChartObjects {
// 		if id.Matches(res.ID) {
// 			shouldInclude = true
// 			break
// 		}
// 	}
// 	if !shouldInclude {
// 		return nil
// 	}
// 	object, err := res.GetProperty("Object")
// 	if err != nil {
// 		return fmt.Errorf("unable to find object property on resource %s: %w", res.ID, err)
// 	}

// 	return nil
// }

// func convertObject(object map[string]any) (map[string]any, error) {
// 	result := make(map[string]any)

// 	switch arg := arg.(type) {
// 	case construct.ResourceId:
// 		return tc.vars[arg], nil

// 	case construct.PropertyRef:
// 		return tc.PropertyRefValue(arg)

// 	case string:
// 		// use templateString to quote the string value
// 		return templateString(arg), nil

// 	case bool, int, float64:
// 		// safe to use as-is
// 		return arg, nil

// 	case nil:
// 		// don't add to inputs
// 		return nil, nil

// 	default:
// 		switch val := reflect.ValueOf(arg); val.Kind() {
// 		case reflect.Slice, reflect.Array:
// 			list := &TsList{l: make([]any, 0, val.Len())}
// 			for i := 0; i < val.Len(); i++ {
// 				if !val.Index(i).IsValid() || val.Index(i).IsNil() {
// 					continue
// 				}
// 				output, err := tc.convertArg(val.Index(i).Interface(), templateArg)
// 				if err != nil {
// 					return "", err
// 				}
// 				list.Append(output)
// 			}
// 			return list, nil
// 		case reflect.Map:
// 			TsMap := &TsMap{m: make(map[string]any)}
// 			for _, key := range val.MapKeys() {
// 				if !val.MapIndex(key).IsValid() || val.MapIndex(key).IsNil() {
// 					continue
// 				}
// 				keyStr, found := key.Interface().(string)
// 				if !found {
// 					return "", fmt.Errorf("map key is not a string")
// 				}
// 				keyResult := strcase.ToLowerCamel(keyStr)
// 				if templateArg != nil && templateArg.Wrapper == string(CamelCaseWrapper) {
// 					keyResult = strcase.ToCamel(keyStr)
// 				} else if templateArg != nil && templateArg.Wrapper == string(ModelCaseWrapper) {
// 					keyResult = keyStr
// 				}

// 				output, err := tc.convertArg(val.MapIndex(key).Interface(), templateArg)
// 				if err != nil {
// 					return "", err
// 				}
// 				TsMap.SetKey(keyResult, output)
// 			}
// 			return TsMap, nil
// 		case reflect.Struct:
// 			if hashset, ok := val.Interface().(set.HashedSet[string, any]); ok {
// 				return tc.convertArg(hashset.ToSlice(), templateArg)
// 			}
// 			fallthrough
// 		default:
// 			return jsonValue{Raw: arg}, nil
// 		}
// 	}

// }
