package resources

import (
	"fmt"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider"
	"regexp"
)

const DOCKER_IMAGE_TYPE = "image"

type (
	DockerImage struct {
		Name              string
		ConstructRefs     core.BaseConstructSet `yaml:"-"`
		CreatesDockerfile bool
	}

	DockerImageCreateParams struct {
		Name string
		Refs core.BaseConstructSet
	}
)

func (image *DockerImage) BaseConstructRefs() core.BaseConstructSet {
	return image.ConstructRefs
}

func (image *DockerImage) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}

func (image *DockerImage) Id() core.ResourceId {
	return core.ResourceId{
		Provider: provider.DOCKER,
		Type:     DOCKER_IMAGE_TYPE,
		Name:     image.Name,
	}
}

func (image *DockerImage) Create(dag *core.ResourceGraph, params DockerImageCreateParams) error {
	image.Name = params.Name
	image.ConstructRefs = params.Refs.Clone()

	existingImage := dag.GetResource(image.Id())
	if existingImage != nil {
		return fmt.Errorf("docker image with name %s already exists", image.Name)
	}
	dag.AddResource(image)
	return nil
}

func (image *DockerImage) GetOutputFiles() []core.File {
	var files []core.File
	if image.CreatesDockerfile {
		files = append(files, image.Dockerfile())
	}
	return files
}

func (image *DockerImage) Dockerfile() core.File {
	return &core.RawFile{
		FPath:   fmt.Sprintf("dockerfiles/%s.dockerfile", regexp.MustCompile(`[\\/]+`).ReplaceAllString(image.Name, "_")),
		Content: []byte(fmt.Sprintf("FROM %s", image.Name)),
	}
}
