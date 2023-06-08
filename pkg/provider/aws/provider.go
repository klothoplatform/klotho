package aws

import (
	"errors"
	"fmt"
	"reflect"
	"sort"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
)

type AWS struct {
	Config                 *config.Application
	constructIdToResources map[core.ResourceId][]core.Resource
	KnowledgeBase          knowledgebase.EdgeKB
}

// MapResourceDirectlyToConstruct tells this AWS instance that the given resource was generated directly because of
// the given construct. Directly is key, or else this could add dependencies to the provider graph that shouldn't be
// there, and in particular could add cycles to what should be a DAG.
//
// Essentially, think about why you're creating a resource in the smallest granularity you can. If it's because a
// construct told you to, then you should use this method. If it's because you need it for another resource, then don't
// use this method — even if that other resource is directly tied to a construct. (In that case, you would use this
// method for that other resource.)
//
// Visually:
//
//	Construct{A}
//	  │
//	  ├─ Resource{B}  // do call MapResourceDirectlyToConstruct for this
//	  │
//	  └─ Resource{C}  // also call it for this
//	       │
//	       └─ Resource{D}  // do NOT call it for this
func (a *AWS) MapResourceDirectlyToConstruct(resource core.Resource, construct core.BaseConstruct) {
	if a.constructIdToResources == nil {
		a.constructIdToResources = make(map[core.ResourceId][]core.Resource)
	}
	newList := append(a.constructIdToResources[construct.Id()], resource)
	sort.Slice(newList, func(i, j int) bool {
		return newList[i].Id().String() < newList[j].Id().String()
	})
	a.constructIdToResources[construct.Id()] = newList
}

func (a *AWS) GetResourcesDirectlyTiedToConstruct(construct core.BaseConstruct) ([]core.Resource, bool) {
	awsResources, found := a.constructIdToResources[construct.Id()]
	return awsResources, found
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
	for namespace, awsResources := range namespacedResources {
		for _, res := range awsResources {
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
