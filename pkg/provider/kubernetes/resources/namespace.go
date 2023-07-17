package resources

import (
	"fmt"
	"reflect"

	"github.com/klothoplatform/klotho/pkg/collectionutil"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/engine/classification"
	"github.com/klothoplatform/klotho/pkg/provider"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type (
	Namespace struct {
		Name            string
		ConstructRefs   core.BaseConstructSet
		Object          *corev1.Namespace
		Transformations map[string]core.IaCValue
		FilePath        string
		Cluster         core.ResourceId
	}
)

const (
	NAMESPACE_TYPE = "namespace"
)

func (namespace *Namespace) BaseConstructRefs() core.BaseConstructSet {
	return namespace.ConstructRefs
}

func (namespace *Namespace) Id() core.ResourceId {
	return core.ResourceId{
		Provider: provider.KUBERNETES,
		Type:     NAMESPACE_TYPE,
		Name:     namespace.Name,
	}
}

func (namespace *Namespace) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}

func (namespace *Namespace) GetObject() runtime.Object {
	return namespace.Object
}

func (namespace *Namespace) Kind() string {
	return namespace.Object.Kind
}

func (namespace *Namespace) Path() string {
	return namespace.FilePath
}

func (namespace *Namespace) MakeOperational(dag *core.ResourceGraph, appName string, classifier classification.Classifier) error {
	if namespace.Cluster.IsZero() {
		downstreamClustersFound := map[string]core.Resource{}
		for _, res := range dag.GetAllDownstreamResources(namespace) {
			if classifier.GetFunctionality(res) == core.Cluster {
				downstreamClustersFound[res.Id().String()] = res
			}
		}
		// See which cluster any pods or deployments using this service account use
		for _, res := range namespace.GetResourcesInNamespace(dag) {
			for _, dres := range dag.GetAllDownstreamResources(res) {
				if classifier.GetFunctionality(dres) == core.Cluster {
					downstreamClustersFound[dres.Id().String()] = dres
				}
			}
		}

		if len(downstreamClustersFound) == 1 {
			_, cluster := collectionutil.GetOneEntry(downstreamClustersFound)
			namespace.Cluster = cluster.Id()
			return nil
		}
		if len(downstreamClustersFound) > 1 {
			return fmt.Errorf("target group binding %s has more than one cluster downstream", namespace.Id())
		}

		return core.NewOperationalResourceError(namespace, []string{string(core.Cluster)}, fmt.Errorf("target group binding %s has no clusters to use", namespace.Id()))
	}
	return nil
}

func (namespace *Namespace) GetResourcesInNamespace(dag *core.ResourceGraph) []core.Resource {
	var resources []core.Resource
	for _, res := range dag.GetAllUpstreamResources(namespace) {
		if manifest, ok := res.(ManifestFile); ok {
			if manifest.GetObject() != nil {
				if reflect.ValueOf(manifest.GetObject()).Elem().FieldByName("Namespace").Interface() == namespace.Name {
					resources = append(resources, manifest)
				}
			}
		}
	}
	return resources
}
