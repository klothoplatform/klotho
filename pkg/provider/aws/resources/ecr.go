package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
)

const (
	ECR_REPO_TYPE  = "ecr_repo"
	ECR_IMAGE_TYPE = "ecr_image"

	ECR_IMAGE_NAME_IAC_VALUE = "ecr_image_name"
)

type (
	EcrRepository struct {
		Name          string
		ConstructsRef []core.AnnotationKey
		ForceDelete   bool
	}

	EcrImage struct {
		Name          string
		ConstructsRef []core.AnnotationKey
		Repo          *EcrRepository
		Context       string
		Dockerfile    string
		ExtraOptions  []string
	}
)

func GenerateEcrRepoAndImage(appName string, unit *core.ExecutionUnit, dag *core.ResourceGraph) (*EcrImage, error) {
	// See if we have already created an ecr repository for the app and if not create one, otherwise add a ref to this exec unit
	var repo *EcrRepository
	existingRepo := dag.GetResource(GenerateRepoId(appName))
	if existingRepo == nil {
		repo = NewEcrRepository(appName, unit.Provenance())
		dag.AddResource(repo)
	} else {
		var ok bool
		repo, ok = existingRepo.(*EcrRepository)
		if !ok {
			return nil, fmt.Errorf("expected resource with id, %s, to be ecr repository", repo.Id())
		}
		repo.ConstructsRef = append(repo.ConstructsRef, unit.Provenance())
	}

	// Create image and make it dependent on the repository
	image := NewEcrImage(unit, appName, repo)
	dag.AddResource(image)
	dag.AddDependency(image, repo)
	return image, nil
}

func NewEcrRepository(appName string, ref core.AnnotationKey) *EcrRepository {
	return &EcrRepository{
		Name:          appName,
		ForceDelete:   true,
		ConstructsRef: []core.AnnotationKey{ref},
	}
}

// Provider returns name of the provider the resource is correlated to
func (repo *EcrRepository) Provider() string {
	return AWS_PROVIDER
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
	return fmt.Sprintf("%s:%s:%s", AWS_PROVIDER, ECR_REPO_TYPE, name)
}

func NewEcrImage(unit *core.ExecutionUnit, appName string, repo *EcrRepository) *EcrImage {
	return &EcrImage{
		Name:          fmt.Sprintf("%s-%s", appName, unit.ID),
		ConstructsRef: []core.AnnotationKey{unit.Provenance()},
		Repo:          repo,
		Context:       fmt.Sprintf("./%s", unit.ID),
		Dockerfile:    fmt.Sprintf("./%s/%s", unit.ID, unit.DockerfilePath),
		ExtraOptions:  []string{"--platform", "linux/amd64", "--quiet"},
	}
}

// Provider returns name of the provider the resource is correlated to
func (image *EcrImage) Provider() string {
	return AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (image *EcrImage) KlothoConstructRef() []core.AnnotationKey {
	return image.ConstructsRef
}

// ID returns the id of the cloud resource
func (image *EcrImage) Id() string {
	return fmt.Sprintf("%s:%s:%s", image.Provider(), ECR_IMAGE_TYPE, image.Name)
}
