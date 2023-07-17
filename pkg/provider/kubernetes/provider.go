package kubernetes

import (
	"fmt"
	"reflect"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/kubernetes/resources"
)

type (
	KubernetesProvider struct {
		AppName string
	}
)

func (k *KubernetesProvider) Name() string {
	return "kubernetes"
}
func (k *KubernetesProvider) ListResources() []core.Resource {
	return resources.ListAll()
}
func (k *KubernetesProvider) CreateResourceFromId(id core.ResourceId, dag *core.ConstructGraph) (core.Resource, error) {
	typeToResource := make(map[string]core.Resource)
	for _, res := range resources.ListAll() {
		typeToResource[res.Id().Type] = res
	}
	res, ok := typeToResource[id.Type]
	if !ok {
		return nil, fmt.Errorf("unable to find resource of type %s", id.Type)
	}
	newResource := reflect.New(reflect.TypeOf(res).Elem()).Interface()
	resource, ok := newResource.(core.Resource)
	if !ok {
		return nil, fmt.Errorf("item %s of type %T is not of type core.Resource", id, newResource)
	}
	reflect.ValueOf(resource).Elem().FieldByName("Name").SetString(id.Name)

	if id.Namespace != "" {
		method := reflect.ValueOf(resource).MethodByName("Load")
		if method.IsValid() {
			var callArgs []reflect.Value
			callArgs = append(callArgs, reflect.ValueOf(id.Namespace))
			callArgs = append(callArgs, reflect.ValueOf(dag))
			eval := method.Call(callArgs)
			if !eval[0].IsNil() {
				err, ok := eval[0].Interface().(error)
				if !ok {
					return nil, fmt.Errorf("return type should be an error")
				}
				if err != nil {
					return nil, err
				}
			}
		}
	}
	return resource, nil
}
