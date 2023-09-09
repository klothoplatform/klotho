package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/classification"
	"github.com/klothoplatform/klotho/pkg/provider"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type (
	Namespace struct {
		Name          string
		ConstructRefs construct.BaseConstructSet `yaml:"-"`
		Object        *corev1.Namespace
		Values        map[string]construct.IaCValue
		FilePath      string
		Cluster       construct.ResourceId
	}
)

const (
	NAMESPACE_TYPE = "namespace"
)

func (namespace *Namespace) BaseConstructRefs() construct.BaseConstructSet {
	return namespace.ConstructRefs
}

func (namespace *Namespace) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: provider.KUBERNETES,
		Type:     NAMESPACE_TYPE,
		Name:     namespace.Name,
	}
}

func (namespace *Namespace) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream: true,
	}
}

func (namespace *Namespace) GetObject() v1.Object {
	return namespace.Object
}

func (namespace *Namespace) Kind() string {
	return namespace.Object.Kind
}

func (namespace *Namespace) Path() string {
	return namespace.FilePath
}

func (namespace *Namespace) GetResourcesInNamespace(dag *construct.ResourceGraph) []construct.Resource {
	var resources []construct.Resource
	for _, res := range dag.GetAllUpstreamResources(namespace) {
		if manifest, ok := res.(ManifestFile); ok {
			if manifest.GetObject() != nil && manifest.GetObject().GetNamespace() == namespace.Name {
				resources = append(resources, manifest)
			}
		}
	}
	return resources
}

func (namespace *Namespace) MakeOperational(dag *construct.ResourceGraph, appName string, classifier *classification.ClassificationDocument) error {
	if namespace.Cluster.Name == "" {
		return fmt.Errorf("namespace %s has no cluster", namespace.Name)
	}

	SetDefaultObjectMeta(namespace, namespace.Object.GetObjectMeta())
	namespace.FilePath = ManifestFilePath(namespace)
	return nil
}

func (namespace *Namespace) GetValues() map[string]construct.IaCValue {
	return namespace.Values
}
