package kubernetes

import (
	"github.com/klothoplatform/klotho/pkg/core"
)

const (
	KUSTOMIZE_DIRECTORY_TYPE = "kustomize_directory"
)

type (
	KustomizeDirectory struct {
		Name             string
		ConstructRefs    []core.AnnotationKey
		Directory        string
		ClustersProvider core.IaCValue
	}
)

// KlothoConstructRef returns a slice containing the ids of any Klotho constructs is correlated to
func (dir *KustomizeDirectory) KlothoConstructRef() []core.AnnotationKey { return dir.ConstructRefs }

func (dir *KustomizeDirectory) Id() core.ResourceId {
	return core.ResourceId{
		Provider: "kubernetes",
		Type:     KUSTOMIZE_DIRECTORY_TYPE,
		Name:     dir.Name,
	}
}
