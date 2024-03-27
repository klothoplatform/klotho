package docker

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"path/filepath"

	"github.com/klothoplatform/klotho/pkg/code/docker/queries"
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/dockerfile"
)

var lang = dockerfile.GetLanguage()

func FindDockerConstraints(ctx context.Context, files fs.FS) (constraints.Constraints, error) {
	var c constraints.Constraints
	err := fs.WalkDir(files, ".", func(path string, d fs.DirEntry, nerr error) error {
		if d.IsDir() {
			return nerr
		}
		if filepath.Ext(path) != ".Dockerfile" && filepath.Base(path) != "Dockerfile" {
			return nerr
		}
		file, err := files.Open(path)
		if err != nil {
			return errors.Join(nerr, err)
		}
		defer file.Close()

		p := sitter.NewParser()
		p.SetLanguage(lang)

		content, err := io.ReadAll(file)
		if err != nil {
			return errors.Join(nerr, err)
		}

		tree, err := p.ParseCtx(ctx, nil, content)
		if err != nil {
			return errors.Join(nerr, err)
		}

		name := "service"
		if filepath.Ext(path) == ".Dockerfile" {
			name = filepath.Base(path)
			name = name[:len(name)-len(filepath.Ext(name))]
		}

		for m := range sitter.QueryIterator(tree.RootNode(), queries.From) {
			serviceId := construct.ResourceId{Provider: "aws", Type: "ecs_service", Name: name}
			taskId := construct.ResourceId{Provider: "aws", Type: "ecs_task_definition", Name: name}
			imageId := construct.ResourceId{Provider: "aws", Type: "ecr_image", Name: name}
			c.Application = append(c.Application,
				constraints.ApplicationConstraint{
					Operator: constraints.MustExistConstraintOperator,
					Node:     serviceId,
				},
				constraints.ApplicationConstraint{
					Operator: constraints.MustExistConstraintOperator,
					Node:     taskId,
				},
				constraints.ApplicationConstraint{
					Operator: constraints.MustExistConstraintOperator,
					Node:     imageId,
				},
			)
			c.Edges = append(c.Edges,
				constraints.EdgeConstraint{
					Operator: constraints.MustExistConstraintOperator,
					Target: constraints.Edge{
						Source: serviceId,
						Target: taskId,
					},
				},
				constraints.EdgeConstraint{
					Operator: constraints.MustExistConstraintOperator,
					Target: constraints.Edge{
						Source: taskId,
						Target: imageId,
					},
				},
			)
			c.Resources = append(c.Resources, constraints.ResourceConstraint{
				Operator: constraints.EqualsConstraintOperator,
				Target:   imageId,
				Property: "BaseImage",
				Value:    m["image"].Content(),
			})
		}
		return nil
	})
	return c, err
}
