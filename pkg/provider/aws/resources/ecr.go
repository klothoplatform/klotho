package resources

import (
	"fmt"
	"strings"

	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
	"github.com/klothoplatform/klotho/pkg/sanitization/docker"

	"github.com/klothoplatform/klotho/pkg/construct"
)

const (
	ECR_REPO_TYPE  = "ecr_repo"
	ECR_IMAGE_TYPE = "ecr_image"

	ECR_IMAGE_NAME_IAC_VALUE = "ecr_image_name"
)

type (
	EcrRepository struct {
		Name          string
		ConstructRefs construct.BaseConstructSet `yaml:"-"`
		ForceDelete   bool
	}

	EcrImage struct {
		Name          string
		ImageName     string
		ConstructRefs construct.BaseConstructSet `yaml:"-"`
		Repo          *EcrRepository
		Context       string
		Dockerfile    string
		ExtraOptions  []string
		// BaseImage is used to denote the base image for the dockerfile that will be pulled before building in the generated IaC
		BaseImage string
	}
)

type RepoCreateParams struct {
	AppName string
	Refs    construct.BaseConstructSet
}

func (repo *EcrRepository) Create(dag *construct.ResourceGraph, params RepoCreateParams) error {
	repo.Name = params.AppName
	repo.ConstructRefs = params.Refs.Clone()

	existingRepo := dag.GetResource(repo.Id())
	if existingRepo != nil {
		graphRepo := existingRepo.(*EcrRepository)
		graphRepo.ConstructRefs.AddAll(params.Refs)
	} else {
		dag.AddResource(repo)
	}
	return nil
}

func (repo *EcrRepository) SanitizedName() string {
	return aws.EcrRepositorySanitizer.Apply(repo.Name)
}

type EcrRepositoryConfigureParams struct {
}

func (repo *EcrRepository) Configure(params EcrRepositoryConfigureParams) error {
	repo.ForceDelete = true
	return nil
}

type ImageCreateParams struct {
	AppName string
	Refs    construct.BaseConstructSet
	Name    string
}

func (image *EcrImage) Create(dag *construct.ResourceGraph, params ImageCreateParams) error {
	name := params.Name
	i := strings.Index(name, "/")
	if i != -1 {
		name = params.Name[i+1:]
	}
	if imageName, tag, ok := strings.Cut(name, ":"); ok {
		image.ImageName = fmt.Sprintf("%s_%s", imageName, tag)
	}

	image.Name = name
	image.ConstructRefs = params.Refs.Clone()

	existingImage := dag.GetResource(image.Id())
	if existingImage != nil {
		return fmt.Errorf("ecr image with name %s already exists", name)
	}
	dag.AddResource(image)
	return nil
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (repo *EcrRepository) BaseConstructRefs() construct.BaseConstructSet {
	return repo.ConstructRefs
}

// Id returns the id of the cloud resource
func (repo *EcrRepository) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     ECR_REPO_TYPE,
		Name:     repo.Name,
	}
}

func (repo *EcrRepository) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream: true,
	}
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (image *EcrImage) BaseConstructRefs() construct.BaseConstructSet {
	return image.ConstructRefs
}

// Id returns the id of the cloud resource
func (image *EcrImage) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     ECR_IMAGE_TYPE,
		Name:     image.Name,
	}
}

func (image *EcrImage) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream: true,
	}
}

func (image *EcrImage) TagBase() string {
	return docker.TagSanitizer.Apply(image.Name)
}
