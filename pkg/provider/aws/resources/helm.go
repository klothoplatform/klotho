package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
)

// TODO look into a better way to represent the k8s provider since it's more of a pulumi construct
type (
	AwsKubernetesProvider struct {
		ConstructRefs []core.AnnotationKey
		KubeConfig    string
		Name          string
	}
)

func (e *AwsKubernetesProvider) Provider() string {
	return "aws"
}

func (e *AwsKubernetesProvider) KlothoConstructRef() []core.AnnotationKey {
	return e.ConstructRefs
}

func (e *AwsKubernetesProvider) Id() string {
	return fmt.Sprintf("%s:%s:%s", e.Provider(), "eks_provider", e.Name)
}
