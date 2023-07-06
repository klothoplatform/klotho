package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/collectionutil"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/engine/classification"
	"github.com/klothoplatform/klotho/pkg/provider"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type (
	ServiceAccount struct {
		Name            string
		ConstructRefs   core.BaseConstructSet
		Object          *corev1.Service
		Transformations map[string]core.IaCValue
		FilePath        string
		Cluster         core.Resource
	}
)

const (
	SERVICE_ACCOUNT_TYPE = "service_account"
)

// BaseConstructsRef returns a slice containing the ids of any Klotho constructs is correlated to
func (sa *ServiceAccount) BaseConstructsRef() core.BaseConstructSet { return sa.ConstructRefs }

func (sa *ServiceAccount) Id() core.ResourceId {
	return core.ResourceId{
		Provider: provider.KUBERNETES,
		Type:     SERVICE_ACCOUNT_TYPE,
		Name:     sa.Name,
	}
}

func (sa *ServiceAccount) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}

func (sa *ServiceAccount) GetObject() runtime.Object {
	return sa.Object
}

func (sa *ServiceAccount) Kind() string {
	return sa.Object.Kind
}

func (sa *ServiceAccount) Path() string {
	return sa.FilePath
}

func (sa *ServiceAccount) MakeOperational(dag *core.ResourceGraph, appName string, classifier classification.Classifier) error {
	if sa.Cluster == nil {
		downstreamClustersFound := map[string]core.Resource{}
		for _, res := range dag.GetAllDownstreamResources(sa) {
			if classifier.GetFunctionality(res) == classification.Cluster {
				downstreamClustersFound[res.Id().String()] = res
			}
		}
		// See which cluster any pods or deployments using this service account use
		for _, res := range sa.GetResourcesUsingServiceAccount(dag) {
			for _, dres := range dag.GetAllDownstreamResources(res) {
				if classifier.GetFunctionality(dres) == classification.Cluster {
					downstreamClustersFound[dres.Id().String()] = dres
				}
			}
		}

		if len(downstreamClustersFound) == 1 {
			_, cluster := collectionutil.GetOneEntry(downstreamClustersFound)
			sa.Cluster = cluster
			dag.AddDependency(sa, cluster)
			return nil
		}
		if len(downstreamClustersFound) > 1 {
			return fmt.Errorf("target group binding %s has more than one cluster downstream", sa.Id())
		}

		return core.NewOperationalResourceError(sa, []string{string(classification.Cluster)}, fmt.Errorf("target group binding %s has no clusters to use", sa.Id()))
	}
	return nil
}

func (sa *ServiceAccount) GetResourcesUsingServiceAccount(dag *core.ResourceGraph) []core.Resource {
	var pods []core.Resource
	for _, res := range dag.GetAllUpstreamResources(sa) {
		if pod, ok := res.(*Pod); ok {
			if pod.Object.Spec.ServiceAccountName == sa.Name {
				pods = append(pods, pod)
			}
		} else if deployment, ok := res.(*Deployment); ok {
			if deployment.Object.Spec.Template.Spec.ServiceAccountName == sa.Name {
				pods = append(pods, deployment)
			}
		}
	}
	return pods
}
