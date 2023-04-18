package kubernetes

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
)

const (
	KUBE_CONFIG_TYPE = "kubeconfig"
)

type (
	Kubeconfig struct {
		ConstructsRef  []core.AnnotationKey
		Name           string
		ApiVersion     string
		Kind           string
		CurrentContext string

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
		User    string
	}

	KubeconfigUsers struct {
		Name string
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

func (k Kubeconfig) Provider() string { return "kubernetes" }

func (k Kubeconfig) KlothoConstructRef() []core.AnnotationKey { return k.ConstructsRef }

func (k Kubeconfig) Id() string {
	return fmt.Sprintf("%s:%s:%s", k.Provider(), KUBE_CONFIG_TYPE, k.Name)
}