package iac3

import (
	construct "github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/solution_context"
	kio "github.com/klothoplatform/klotho/pkg/io"
)

// RenderDockerfiles is a temporary workaround for rendering trivial Dockerfiles for resources.
// Ideally this isn't explicit but instead handled by a template in some fashion.
func RenderDockerfiles(ctx solution_context.SolutionContext) ([]kio.File, error) {
	resources, err := construct.ReverseTopologicalSort(ctx.DeploymentGraph())
	if err != nil {
		return nil, err
	}
	var files []kio.File
	for _, rid := range resources {
		if rid.QualifiedTypeName() != "aws:ecr_image" {
			continue
		}

		res, err := ctx.DeploymentGraph().Vertex(rid)
		if err != nil {
			return nil, err
		}

		baseImage, err := res.GetProperty("BaseImage")
		if err != nil {
			return nil, err
		}
		if baseImage == nil {
			continue
		}

		dockerfile, err := res.GetProperty("Dockerfile")
		if err != nil {
			return nil, err
		}

		files = append(files, &kio.RawFile{
			FPath:   dockerfile.(string),
			Content: []byte("FROM " + baseImage.(string)),
		})
	}
	return files, nil
}
