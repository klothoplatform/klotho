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
		Contexts []KubeconfigContext
		Users    []KubeconfigUser
	}

	KubeconfigCluster struct {
		Name                     core.IaCValue
		CertificateAuthorityData core.IaCValue
		Server                   core.IaCValue
	}

	KubeconfigContext struct {
		Cluster core.IaCValue
		User    string
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
