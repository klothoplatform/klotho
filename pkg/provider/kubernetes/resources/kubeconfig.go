package resources

import (
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/provider"
)

const (
	KUBE_CONFIG_TYPE = "kubeconfig"
)

type (
	Kubeconfig struct {
		ConstructRefs  construct.BaseConstructSet `yaml:"-"`
		Name           string
		ApiVersion     string
		Kind           string
		CurrentContext construct.IaCValue

		Clusters []KubeconfigCluster
		Contexts []KubeconfigContexts
		Users    []KubeconfigUsers
	}

	KubeconfigCluster struct {
		Name    construct.IaCValue
		Cluster map[string]construct.IaCValue
	}

	KubeconfigContexts struct {
		Context KubeconfigContext
		Name    construct.IaCValue
	}
	KubeconfigContext struct {
		Cluster construct.IaCValue
		User    construct.IaCValue
	}

	KubeconfigUsers struct {
		Name construct.IaCValue
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

func (k Kubeconfig) BaseConstructRefs() construct.BaseConstructSet { return k.ConstructRefs }

func (k Kubeconfig) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: provider.KUBERNETES,
		Type:     KUBE_CONFIG_TYPE,
		Name:     k.Name,
	}
}

func (k Kubeconfig) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream: true,
	}
}
