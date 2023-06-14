package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"go.uber.org/zap"
)

const (
	ECR_REPO_TYPE  = "ecr_repo"
	ECR_IMAGE_TYPE = "ecr_image"

	ECR_IMAGE_NAME_IAC_VALUE = "ecr_image_name"
)

type (
	EcrRepository struct {
		Name          string
		ConstructsRef core.BaseConstructSet
		ForceDelete   bool
	}

	EcrImage struct {
		Name          string
		ConstructsRef core.BaseConstructSet
		Repo          *EcrRepository
		Context       string
		Dockerfile    string
		ExtraOptions  []string
	}
)

type RepoCreateParams struct {
	AppName string
	Refs    core.BaseConstructSet
}

func (repo *EcrRepository) Create(dag *core.ResourceGraph, params RepoCreateParams) error {
	repo.Name = params.AppName
	repo.ConstructsRef = params.Refs.Clone()

	existingRepo := dag.GetResource(repo.Id())
	if existingRepo != nil {
		graphRepo := existingRepo.(*EcrRepository)
		graphRepo.ConstructsRef.AddAll(params.Refs)
	} else {
		dag.AddResource(repo)
	}
	return nil
}

type EcrRepositoryConfigureParams struct {
}

func (repo *EcrRepository) Configure(params EcrRepositoryConfigureParams) error {
	repo.ForceDelete = true
	return nil
}

type ImageCreateParams struct {
	AppName string
	Refs    core.BaseConstructSet
	Name    string
}

func (image *EcrImage) Create(dag *core.ResourceGraph, params ImageCreateParams) error {
	name := fmt.Sprintf("%s-%s", params.AppName, params.Name)
	image.Name = name
	image.ConstructsRef = params.Refs.Clone()

	existingImage := dag.GetResource(image.Id())
	if existingImage != nil {
		return fmt.Errorf("ecr image with name %s already exists", name)
	}

	err := dag.CreateDependencies(image, map[string]any{
		"Repo": RepoCreateParams{
			AppName: params.AppName,
			Refs:    params.Refs.Clone(),
		},
	})
	return err
}

type EcrImageConfigureParams struct {
	Context    string
	Dockerfile string
}

// Configure sets the intristic characteristics of a vpc based on parameters passed in
func (image *EcrImage) Configure(params EcrImageConfigureParams) error {
	image.ExtraOptions = []string{"--platform", "linux/amd64", "--quiet"}
	if params.Dockerfile == "" {
		zap.S().Warnf("image %s does not have dockerfile set, leaving empty", image.Name)
		// return fmt.Errorf("image %s must have dockerfile set", image.Name)
	}
	image.Dockerfile = params.Dockerfile
	if params.Context == "" {
		zap.S().Warnf("image %s does not have context set, leaving empty", image.Name)
		// return fmt.Errorf("image %s must have context set", image.Name)
	}
	image.Context = params.Context
	return nil
}

// BaseConstructsRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (repo *EcrRepository) BaseConstructsRef() core.BaseConstructSet {
	return repo.ConstructsRef
}

// Id returns the id of the cloud resource
func (repo *EcrRepository) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     ECR_REPO_TYPE,
		Name:     repo.Name,
	}
}

// BaseConstructsRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (image *EcrImage) BaseConstructsRef() core.BaseConstructSet {
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
