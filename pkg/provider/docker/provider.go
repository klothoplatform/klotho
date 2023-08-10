package docker

import (
	"fmt"
	"reflect"

	"github.com/klothoplatform/klotho/pkg/core"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider"
	"github.com/klothoplatform/klotho/pkg/provider/docker/resources"
)

type DockerProvider struct {
}

func (a *DockerProvider) GetOperationalTempaltes() map[string]*core.ResourceTemplate {
	// Not implemented
	return map[string]*core.ResourceTemplate{}
}

func (a *DockerProvider) GetEdgeTempaltes() map[string]*knowledgebase.EdgeTemplate {
	// Not implemented
	return map[string]*knowledgebase.EdgeTemplate{}
}

func (a *DockerProvider) Name() string { return provider.DOCKER }

func (a *DockerProvider) ListResources() []core.Resource {
	return resources.ListAll()
}

// CreateResourceFromId creates a resource from an id, but does not mutate the graph in any manner
// The graph is passed in to be able to understand what namespaces reference in resource ids
func (a *DockerProvider) CreateResourceFromId(id core.ResourceId, dag *core.ConstructGraph) (core.Resource, error) {
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
