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
		ConstructRefs    core.BaseConstructSet
		Directory        string
		ClustersProvider core.IaCValue
	}
)

// BaseConstructsRef returns a slice containing the ids of any Klotho constructs is correlated to
func (dir *KustomizeDirectory) BaseConstructsRef() core.BaseConstructSet { return dir.ConstructRefs }

func (dir *KustomizeDirectory) Id() core.ResourceId {
	return core.ResourceId{
		Provider: "kubernetes",
		Type:     KUSTOMIZE_DIRECTORY_TYPE,
		Name:     dir.Name,
	}
}
func (k *KustomizeDirectory) DeleteCriteria() core.DeleteCriteria {
	return core.DeleteCriteria{
		RequiresNoUpstream: true,
	}
}
