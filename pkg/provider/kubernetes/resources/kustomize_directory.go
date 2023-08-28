package resources

import (
	"github.com/klothoplatform/klotho/pkg/construct"
)

const (
	KUSTOMIZE_DIRECTORY_TYPE = "kustomize_directory"
)

type (
	KustomizeDirectory struct {
		Name             string
		ConstructRefs    construct.BaseConstructSet `yaml:"-"`
		Directory        string
		ClustersProvider construct.IaCValue
		Cluster          construct.ResourceId
	}
)

// BaseConstructRefs returns a slice containing the ids of any Klotho constructs is correlated to
func (dir *KustomizeDirectory) BaseConstructRefs() construct.BaseConstructSet {
	return dir.ConstructRefs
}

func (dir *KustomizeDirectory) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: "kubernetes",
		Type:     KUSTOMIZE_DIRECTORY_TYPE,
		Name:     dir.Name,
	}
}
func (k *KustomizeDirectory) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream: true,
	}
}
