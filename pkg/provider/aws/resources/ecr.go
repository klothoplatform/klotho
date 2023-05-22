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

type RepoCreateParams struct {
	AppName string
	Refs    []core.AnnotationKey
}

func (repo *EcrRepository) Create(dag *core.ResourceGraph, params RepoCreateParams) error {
	repo.Name = params.AppName
	repo.ConstructsRef = params.Refs

	existingRepo := dag.GetResource(repo.Id())
	if existingRepo != nil {
		graphRepo := existingRepo.(*EcrRepository)
		graphRepo.ConstructsRef = core.DedupeAnnotationKeys(append(graphRepo.KlothoConstructRef(), params.Refs...))
	} else {
		dag.AddResource(repo)
	}
	return nil
}

type EcrRepositoryConfigureParams struct {
	ForceDelete bool
}

func (repo *EcrRepository) Configure(params EcrRepositoryConfigureParams) error {
	repo.ForceDelete = true
	return nil
}

type ImageCreateParams struct {
	AppName string
	Refs    []core.AnnotationKey
	Name    string
}

func (image *EcrImage) Create(dag *core.ResourceGraph, params ImageCreateParams) error {
	name := fmt.Sprintf("%s-%s", params.AppName, params.Name)
	image.Name = name
	image.ConstructsRef = params.Refs

	existingImage := dag.GetResource(image.Id())
	if existingImage != nil {
		return fmt.Errorf("ecr image with name %s already exists", name)
	}

	err := dag.CreateDependencies(image, map[string]any{
		"Repo": RepoCreateParams{
			AppName: params.AppName,
			Refs:    params.Refs,
		},
	})
	return err
}

type EcrImageConfigureParams struct {
	ExtraOptions []string
}

// Configure sets the intristic characteristics of a vpc based on parameters passed in
func (image *EcrImage) Configure(params EcrImageConfigureParams) error {
	image.ExtraOptions = []string{"--platform", "linux/amd64", "--quiet"}
	return nil
}

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

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (repo *EcrRepository) KlothoConstructRef() []core.AnnotationKey {
	return repo.ConstructsRef
}

// Id returns the id of the cloud resource
func (repo *EcrRepository) Id() core.ResourceId {
	return GenerateRepoId(repo.Name)
}

func GenerateRepoId(name string) core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     ECR_REPO_TYPE,
		Name:     name,
	}
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

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (image *EcrImage) KlothoConstructRef() []core.AnnotationKey {
	return image.ConstructsRef
}

// Id returns the id of the cloud resource
func (image *EcrImage) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     ECR_IMAGE_TYPE,
		Name:     image.Name,
	}
}
