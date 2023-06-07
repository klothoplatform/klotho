package aws

import (
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
	resources, found := a.constructIdToResources[construct.Id()]
	return resources, found
}

func (a *AWS) LoadGraph(graph core.OutputGraph, dag *core.ConstructGraph) error {
	typeToResource := make(map[string]core.Resource)
	namespacedResources := make(map[string][]core.Resource)
	for _, res := range resources.ListAll() {
		typeToResource[res.Id().Type] = res
	}
	// Subnets are special because they have a type that is not the same as their resource type since it uses a characteristic of the subnet
	typeToResource["subnet_private"] = &resources.Subnet{}
	typeToResource["subnet_public"] = &resources.Subnet{}

	for _, node := range graph.Resources {
		res, ok := typeToResource[node.Type]
		if !ok {
			return fmt.Errorf("unable to find resource of type %s", node.Type)
		}
		reflect.ValueOf(res).Elem().FieldByName("Name").SetString(node.Name)
		if node.Namespace != "" {
			namespacedResources[node.Namespace] = append(namespacedResources[node.Namespace], res)
			continue
		}
		dag.AddConstruct(res)
	}

	// For anything namespaced, we will call the Load Method with the namespace and dag as the argument.
	// This will allow the resource to be loaded into the graph since its id relies on the namespaced object
	for namespace, resources := range namespacedResources {
		for _, res := range resources {
			method := reflect.ValueOf(res).MethodByName("Load")
			if method.IsValid() {
				var callArgs []reflect.Value
				callArgs = append(callArgs, reflect.ValueOf(namespace))
				callArgs = append(callArgs, reflect.ValueOf(dag))
				eval := method.Call(callArgs)
				if eval[0].IsNil() {
					return nil
				} else {
					err, ok := eval[0].Interface().(error)
					if !ok {
						return fmt.Errorf("return type should be an error")
					}
					return err
				}
			}
			dag.AddConstruct(res)
		}
	}

	for _, edge := range graph.Edges {
		dag.AddDependency(edge.Source, edge.Destination)
	}
	return nil
}
