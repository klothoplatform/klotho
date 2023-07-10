package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/engine/classification"
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
		ConstructRefs core.BaseConstructSet `yaml:"-"`
		ForceDelete   bool
	}

	EcrImage struct {
		Name          string
		ConstructRefs core.BaseConstructSet `yaml:"-"`
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
	image.ConstructRefs = params.Refs.Clone()

	existingImage := dag.GetResource(image.Id())
	if existingImage != nil {
		return fmt.Errorf("ecr image with name %s already exists", name)
	}
	dag.AddResource(image)
	return nil
}

func (image *EcrImage) MakeOperational(dag *core.ResourceGraph, appName string, classifier classification.Classifier) error {
	if image.Repo == nil {
		repos := core.GetDownstreamResourcesOfType[*EcrRepository](dag, image)
		if len(repos) == 0 {
			err := dag.CreateDependencies(image, map[string]any{
				"Repo": RepoCreateParams{
					AppName: appName,
					Refs:    core.BaseConstructSetOf(image),
				},
			})
			if err != nil {
				return err
			}
		} else if len(repos) == 1 {
			image.Repo = repos[0]
			dag.AddDependency(image, image.Repo)
		} else {
			return fmt.Errorf("ecr image %s has more than one repo downstream", image.Id())
		}
	}
	return nil
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

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (repo *EcrRepository) BaseConstructRefs() core.BaseConstructSet {
	return repo.ConstructRefs
}

// Id returns the id of the cloud resource
func (repo *EcrRepository) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     ECR_REPO_TYPE,
		Name:     repo.Name,
	}
}

func (repo *EcrRepository) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (image *EcrImage) BaseConstructRefs() core.BaseConstructSet {
	return image.ConstructRefs
}

// Id returns the id of the cloud resource
func (image *EcrImage) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     ECR_IMAGE_TYPE,
		Name:     image.Name,
	}
}

func (image *EcrImage) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}
