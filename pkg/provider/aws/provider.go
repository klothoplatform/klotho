package aws

import (
	"sort"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

type AWS struct {
	Config                 *config.Application
	constructIdToResources map[core.ResourceId][]core.Resource
	PolicyGenerator        *resources.PolicyGenerator
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
