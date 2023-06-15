package aws

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

type AWS struct {
	AppName string
}

func (a *AWS) Name() string { return provider.AWS }

func (a *AWS) ExpandConstruct(construct core.Construct, dag *core.ResourceGraph, constructType string) (directlyMappedResources []core.Resource, err error) {
	switch construct := construct.(type) {
	case *core.ExecutionUnit:
		return a.expandExecutionUnit(dag, construct, constructType, map[string]any{})
	case *core.Gateway:
		return a.expandExpose(dag, construct, constructType)
	case *core.Orm:
		return a.expandOrm(dag, construct, constructType)
	case *core.Fs:
		return a.expandFs(dag, construct)
	case *core.InternalResource:
		return a.expandFs(dag, construct)
	case *core.Kv:
		return a.expandKv(dag, construct)
	case *core.RedisNode:
		return a.expandRedisNode(dag, construct)
	case *core.StaticUnit:
		return a.expandStaticUnit(dag, construct)
	case *core.Secrets:
		return a.expandSecrets(dag, construct)
	case *core.Config:
		return a.expandConfig(dag, construct)
	default:
		err = fmt.Errorf("unknown construct type %T", construct)
	}
	return
}

func (a *AWS) LoadGraph(graph core.OutputGraph, dag *core.ConstructGraph) error {
	typeToResource := make(map[string]core.Resource)
	namespacedResources := make(map[string][]core.Resource)
	createdResources := make(map[core.ResourceId]core.Resource)
	for _, res := range resources.ListAll() {
		typeToResource[res.Id().Type] = res
	}
	// Subnets are special because they have a type that is not the same as their resource type since it uses a characteristic of the subnet
	typeToResource["subnet_private"] = &resources.Subnet{}
	typeToResource["subnet_public"] = &resources.Subnet{}
	var joinedErr error
	for _, node := range graph.Resources {
		if node.Provider != "aws" {
			continue
		}
		res, ok := typeToResource[node.Type]
		if !ok {
			joinedErr = errors.Join(joinedErr, fmt.Errorf("unable to find resource of type %s", node.Type))
			continue
		}
		newResource := reflect.New(reflect.TypeOf(res).Elem()).Interface()
		resource, ok := newResource.(core.Resource)
		if !ok {
			joinedErr = errors.Join(joinedErr, fmt.Errorf("item %s of type %T is not of type core.Resource", node, newResource))
			continue
		}
		reflect.ValueOf(resource).Elem().FieldByName("Name").SetString(node.Name)
		if subnet, ok := resource.(*resources.Subnet); ok {
			if node.Type == "subnet_public" {
				subnet.Type = resources.PublicSubnet
			} else if node.Type == "subnet_private" {
				subnet.Type = resources.PrivateSubnet
			}
		}
		if node.Namespace != "" {
			namespacedResources[node.Namespace] = append(namespacedResources[node.Namespace], resource)
			createdResources[node] = resource
			continue
		}

		dag.AddConstruct(resource)
		createdResources[node] = resource
	}

	// For anything namespaced, we will call the Load Method with the namespace and dag as the argument.
	// This will allow the resource to be loaded into the graph since its id relies on the namespaced object
	for namespace, namespaceResources := range namespacedResources {
		for _, res := range namespaceResources {
			method := reflect.ValueOf(res).MethodByName("Load")
			if method.IsValid() {
				var callArgs []reflect.Value
				callArgs = append(callArgs, reflect.ValueOf(namespace))
				callArgs = append(callArgs, reflect.ValueOf(dag))
				eval := method.Call(callArgs)
				if !eval[0].IsNil() {
					err, ok := eval[0].Interface().(error)
					if !ok {
						joinedErr = errors.Join(joinedErr, fmt.Errorf("return type should be an error"))
						continue
					}
					joinedErr = errors.Join(joinedErr, err)
					continue
				}
			}
			dag.AddConstruct(res)
		}
	}

	for _, edge := range graph.Edges {
		if edge.Source.Provider != "aws" || edge.Destination.Provider != "aws" {
			continue
		}
		src, found := createdResources[edge.Source]
		if !found {
			joinedErr = errors.Join(joinedErr, fmt.Errorf("could not find created resource for %s", edge.Source))
			continue
		}
		dst, found := createdResources[edge.Destination]
		if !found {
			joinedErr = errors.Join(joinedErr, fmt.Errorf("could not find created resource for %s", edge.Destination))
			continue
		}
		dag.AddDependency(src.Id(), dst.Id())
	}
	return joinedErr
}

// CreateResourceFromId creates a resource from an id, but does not mutate the graph in any manner
// The graph is passed in to be able to understand what namespaces reference in resource ids
func (a *AWS) CreateResourceFromId(id core.ResourceId, dag *core.ResourceGraph) (core.Resource, error) {
	typeToResource := make(map[string]core.Resource)
	for _, res := range resources.ListAll() {
		typeToResource[res.Id().Type] = res
	}
	// Subnets are special because they have a type that is not the same as their resource type since it uses a characteristic of the subnet
	typeToResource["subnet_private"] = &resources.Subnet{}
	typeToResource["subnet_public"] = &resources.Subnet{}
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
	if subnet, ok := resource.(*resources.Subnet); ok {
		if id.Type == "subnet_public" {
			subnet.Type = resources.PublicSubnet
		} else if id.Type == "subnet_private" {
			subnet.Type = resources.PrivateSubnet
		}
	}

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
