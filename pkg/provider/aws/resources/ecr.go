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

func (repo *EcrRepository) Create(dag *core.ResourceGraph, metadata map[string]any) (core.Resource, error) {
	type repoMetadata struct {
		AppName string
		Refs    []core.AnnotationKey
	}

	if repo == nil {
		repoMetadata := &repoMetadata{}
		decoder := getMapDecoder(repoMetadata)
		err := decoder.Decode(metadata)
		if err != nil {
			return repo, err
		}
		repo = &EcrRepository{
			Name:          repoMetadata.AppName,
			ForceDelete:   true,
			ConstructsRef: repoMetadata.Refs,
		}
	}

	err := dag.CreateRecursively(repo, metadata)
	return repo, err
}

func (image *EcrImage) Create(dag *core.ResourceGraph, metadata map[string]any) (core.Resource, error) {
	type imageMetadata struct {
		AppName        string
		Refs           []core.AnnotationKey
		Unit           string
		DockerfilePath string
	}
	fmt.Println(image)

	if image == nil {
		fmt.Println("image is nil")
		imageMetadata := &imageMetadata{}
		decoder := getMapDecoder(imageMetadata)
		err := decoder.Decode(metadata)
		if err != nil {
			return image, err
		}
		fmt.Println("setting image")
		image = &EcrImage{
			Name:          fmt.Sprintf("%s-%s", imageMetadata.AppName, imageMetadata.Unit),
			ConstructsRef: imageMetadata.Refs,
			Context:       fmt.Sprintf("./%s", imageMetadata.Unit),
			Dockerfile:    fmt.Sprintf("./%s/%s", imageMetadata.Unit, imageMetadata.DockerfilePath),
			ExtraOptions:  []string{"--platform", "linux/amd64", "--quiet"},
		}
	}

	fmt.Println(image)

	err := dag.CreateRecursively(image, metadata)
	return image, err
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
