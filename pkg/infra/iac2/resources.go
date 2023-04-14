package iac2

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
)

type (
	KubernetesProvider struct {
		ConstructsRef         []core.AnnotationKey
		KubeConfig            core.Resource
		Name                  string
		EnableServerSideApply bool
	}
)

func (e *KubernetesProvider) Provider() string {
	return "pulumi"
}

func (e *KubernetesProvider) KlothoConstructRef() []core.AnnotationKey {
	return e.ConstructsRef
}

func (e *KubernetesProvider) Id() string {
	return fmt.Sprintf("%s:%s:%s", e.Provider(), "kubernetes_provider", e.Name)
}
