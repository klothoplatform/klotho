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
	ServiceAccount struct {
		Name          string
		ConstructRefs construct.BaseConstructSet `yaml:"-"`
		Object        *corev1.ServiceAccount
		Values        map[string]construct.IaCValue
		FilePath      string
		Cluster       construct.ResourceId
	}

	ServiceAccountCreateParams struct {
		Name          string
		ConstructRefs construct.BaseConstructSet
	}
)

const (
	SERVICE_ACCOUNT_TYPE = "service_account"
)

// BaseConstructRefs returns a slice containing the ids of any Klotho constructs is correlated to
func (sa *ServiceAccount) BaseConstructRefs() construct.BaseConstructSet { return sa.ConstructRefs }

func (sa *ServiceAccount) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: provider.KUBERNETES,
		Type:     SERVICE_ACCOUNT_TYPE,
		Name:     sa.Name,
	}
}

func (sa *ServiceAccount) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream: true,
	}
}

func (sa *ServiceAccount) GetObject() v1.Object {
	return sa.Object
}

func (sa *ServiceAccount) Kind() string {
	return sa.Object.Kind
}

func (sa *ServiceAccount) Path() string {
	return sa.FilePath
}

func (sa *ServiceAccount) GetResourcesUsingServiceAccount(dag *construct.ResourceGraph) []construct.Resource {
	var pods []construct.Resource
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

func (sa *ServiceAccount) MakeOperational(dag *construct.ResourceGraph, appName string, classifier classification.Classifier) error {
	if sa.Cluster.IsZero() {
		return fmt.Errorf("%s has no cluster", sa.Id())
	}
	if sa.Object == nil {
		sa.Object = &corev1.ServiceAccount{}
	}

	SetDefaultObjectMeta(sa, sa.Object.GetObjectMeta())
	sa.FilePath = ManifestFilePath(sa)
	return nil
}

func (sa *ServiceAccount) GetValues() map[string]construct.IaCValue {
	return sa.Values
}

func (sa *ServiceAccount) Create(dag *construct.ResourceGraph, params ServiceAccountCreateParams) error {
	sa.Name = fmt.Sprintf("%s-%s", "service_account", params.Name)
	sa.ConstructRefs = params.ConstructRefs.Clone()
	sa.Object = &corev1.ServiceAccount{
		TypeMeta: v1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
	}
	return nil
}
