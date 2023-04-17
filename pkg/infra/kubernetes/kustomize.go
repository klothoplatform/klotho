package kubernetes

import (
	"fmt"

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

// Provider returns name of the provider the resource is correlated to
func (dir *KustomizeDirectory) Provider() string { return "kubernetes" }

// KlothoConstructRef returns a slice containing the ids of any Klotho constructs is correlated to
func (dir *KustomizeDirectory) KlothoConstructRef() []core.AnnotationKey { return dir.ConstructRefs }

func (dir *KustomizeDirectory) Id() string {
	return fmt.Sprintf("%s:%s:%s", dir.Provider(), KUSTOMIZE_DIRECTORY_TYPE, dir.Name)
}
