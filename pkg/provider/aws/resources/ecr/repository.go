package ecr

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

const ECR_REPO_TYPE = "ecr_repo"

type (
	EcrRepository struct {
		Name          string
		ConstructsRef []core.AnnotationKey
		ForceDelete   bool
	}
)

func NewEcrRepository(appName string, ref core.AnnotationKey) *EcrRepository {
	return &EcrRepository{
		Name:          appName,
		ForceDelete:   true,
		ConstructsRef: []core.AnnotationKey{ref},
	}
}

// Provider returns name of the provider the resource is correlated to
func (repo *EcrRepository) Provider() string {
	return resources.AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (repo *EcrRepository) KlothoConstructRef() []core.AnnotationKey {
	return repo.ConstructsRef
}

// ID returns the id of the cloud resource
func (repo *EcrRepository) Id() string {
	return GenerateRepoId(repo.Name)
}

func GenerateRepoId(name string) string {
	return fmt.Sprintf("%s_%s", ECR_REPO_TYPE, name)
}
