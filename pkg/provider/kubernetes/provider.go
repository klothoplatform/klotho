package kubernetes

import (
	"embed"
	"fmt"
	"io/fs"
	"reflect"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/kubernetes/resources"
	"gopkg.in/yaml.v3"
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

//go:embed resources/templates/*
var kubernetesTemplates embed.FS

func (k *KubernetesProvider) GetOperationalTempaltes() map[string]*core.ResourceTemplate {
	templates := map[string]*core.ResourceTemplate{}
	if err := fs.WalkDir(kubernetesTemplates, ".", func(path string, d fs.DirEntry, nerr error) error {
		if d.IsDir() {
			return nil
		}
		content, err := kubernetesTemplates.ReadFile(fmt.Sprintf("resources/templates/%s", d.Name()))
		if err != nil {
			panic(err)
		}
		resTemplate := &core.ResourceTemplate{}
		err = yaml.Unmarshal(content, resTemplate)
		if err != nil {
			panic(err)
		}
		if templates[resTemplate.Type] != nil {
			panic(fmt.Errorf("duplicate template for type %s", resTemplate.Type))
		}
		templates[resTemplate.Type] = resTemplate
		return nil
	}); err != nil {
		return templates
	}
	return templates
}
