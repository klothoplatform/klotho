package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/io"
	"github.com/klothoplatform/klotho/pkg/provider"
)

const DOCKER_IMAGE_TYPE = "image"

type (
	// DockerImage is a little tailored to the current IaC structure and the fact that Pulumi doesn't have
	// a native docker push resource for copying an image to a registry.
	DockerImage struct {
		Name           string
		BaseImage      string
		ConstructRefs  construct.BaseConstructSet `yaml:"-"`
		DockerfilePath string
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

func (image *DockerImage) GetOutputFiles() []io.File {
	return []io.File{image.Dockerfile()}
}

func (image *DockerImage) Dockerfile() io.File {
	return &io.RawFile{
		FPath:   image.DockerfilePath,
		Content: []byte(fmt.Sprintf("FROM %s", image.BaseImage)),
	}
}
