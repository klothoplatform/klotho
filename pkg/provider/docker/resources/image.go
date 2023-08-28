package resources

import (
	"fmt"
	"regexp"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/io"
	"github.com/klothoplatform/klotho/pkg/provider"
)

const DOCKER_IMAGE_TYPE = "image"

type (
	DockerImage struct {
		Name              string
		ConstructRefs     construct.BaseConstructSet `yaml:"-"`
		CreatesDockerfile bool
	}

	DockerImageCreateParams struct {
		Name string
		Refs construct.BaseConstructSet
	}
)

func (image *DockerImage) BaseConstructRefs() construct.BaseConstructSet {
	return image.ConstructRefs
}

func (image *DockerImage) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream: true,
	}
}

func (image *DockerImage) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: provider.DOCKER,
		Type:     DOCKER_IMAGE_TYPE,
		Name:     image.Name,
	}
}

func (image *DockerImage) Create(dag *construct.ResourceGraph, params DockerImageCreateParams) error {
	image.Name = params.Name
	image.ConstructRefs = params.Refs.Clone()

	existingImage := dag.GetResource(image.Id())
	if existingImage != nil {
		return fmt.Errorf("docker image with name %s already exists", image.Name)
	}
	dag.AddResource(image)
	return nil
}

func (image *DockerImage) GetOutputFiles() []io.File {
	var files []io.File
	if image.CreatesDockerfile {
		files = append(files, image.Dockerfile())
	}
	return files
}

func (image *DockerImage) Dockerfile() io.File {
	return &io.RawFile{
		FPath:   fmt.Sprintf("dockerfiles/%s.dockerfile", regexp.MustCompile(`[\\/]+`).ReplaceAllString(image.Name, "_")),
		Content: []byte(fmt.Sprintf("FROM %s", image.Name)),
	}
}
