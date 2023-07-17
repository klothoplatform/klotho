package resources

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider"
)

const (
	KUBE_CONFIG_TYPE = "kubeconfig"
)

type (
	Kubeconfig struct {
		ConstructRefs  core.BaseConstructSet
		Name           string
		ApiVersion     string
		Kind           string
		CurrentContext core.IaCValue

		Clusters []KubeconfigCluster
		Contexts []KubeconfigContexts
		Users    []KubeconfigUsers
	}

	KubeconfigCluster struct {
		Name    core.IaCValue
		Cluster map[string]core.IaCValue
	}

	KubeconfigContexts struct {
		Context KubeconfigContext
		Name    core.IaCValue
	}
	KubeconfigContext struct {
		Cluster core.IaCValue
		User    core.IaCValue
	}

	KubeconfigUsers struct {
		Name core.IaCValue
		User KubeconfigUser
	}

	KubeconfigUser struct {
		Exec KubeconfigExec
	}

	KubeconfigExec struct {
		ApiVersion string
		Command    string
		Args       []any
	}
)

func (k Kubeconfig) BaseConstructRefs() core.BaseConstructSet { return k.ConstructRefs }

func (k Kubeconfig) Id() core.ResourceId {
	return core.ResourceId{
		Provider: provider.KUBERNETES,
		Type:     KUBE_CONFIG_TYPE,
		Name:     k.Name,
	}
}

func (k Kubeconfig) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}
