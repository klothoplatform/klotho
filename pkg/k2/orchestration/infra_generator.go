package orchestration

import (
	"context"
	"fmt"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	"github.com/klothoplatform/klotho/pkg/engine/solution"
	"github.com/klothoplatform/klotho/pkg/infra/iac"
	kio "github.com/klothoplatform/klotho/pkg/io"
	"github.com/klothoplatform/klotho/pkg/knowledgebase"
	"github.com/klothoplatform/klotho/pkg/knowledgebase/reader"
	"github.com/klothoplatform/klotho/pkg/logging"
	"github.com/klothoplatform/klotho/pkg/provider/aws"
	"github.com/klothoplatform/klotho/pkg/templates"
	"gopkg.in/yaml.v3"
)

type (
	InfraGenerator struct {
		Engine *engine.Engine
	}

	InfraRequest struct {
		engine.SolveRequest
		OutputDir string
	}
)

func NewInfraGenerator() (*InfraGenerator, error) {
	kb, err := reader.NewKBFromFs(templates.ResourceTemplates, templates.EdgeTemplates, templates.Models)
	if err != nil {
		return nil, err
	}
	return &InfraGenerator{
		Engine: engine.NewEngine(kb),
	}, nil
}

func (g *InfraGenerator) Run(ctx context.Context, c constraints.Constraints, outDir string) error {
	// TODO the engine currently assumes only 1 run globally, so the debug graphs and other files
	// will get overwritten with each run. We should fix this.
	sol, errs := g.resolveResources(ctx, InfraRequest{
		SolveRequest: engine.SolveRequest{
			Constraints: c,
			GlobalTag:   "k2",
		},
		OutputDir: outDir,
	})
	if errs != nil {
		return fmt.Errorf("failed to resolve resources: %v", errs)
	}

	err := g.generateIac(iacRequest{
		PulumiAppName: "k2",
		Solution:      sol,
		OutputDir:     outDir,
	})
	if err != nil {
		return fmt.Errorf("failed to generate iac: %w", err)

	}
	return nil
}

func (g *InfraGenerator) resolveResources(ctx context.Context, request InfraRequest) (solution.Solution, error) {
	sol, engineErr := g.Engine.Run(ctx, &request.SolveRequest)
	log := logging.GetLogger(ctx)

	log.Info("Engine finished running... Generating views")

	var files []kio.File

	log.Info("Serializing constraints")
	constraintBytes, err := yaml.Marshal(request.Constraints)
	if err != nil {
		log.Error("Failed to marshal constraints")
	}
	constraintFile := &kio.RawFile{
		FPath:   "constraints.yaml",
		Content: constraintBytes,
	}
	files = append(files, constraintFile)
	vizFiles, err := g.Engine.VisualizeViews(sol)
	if err != nil {
		return nil, fmt.Errorf("failed to generate views %w", err)
	}
	files = append(files, vizFiles...)
	log.Info("Generating resources.yaml")
	b, err := yaml.Marshal(construct.YamlGraph{Graph: sol.DataflowGraph(), Outputs: sol.Outputs()})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal graph: %w", err)
	}
	files = append(files,
		&kio.RawFile{
			FPath:   "resources.yaml",
			Content: b,
		},
	)

	policyBytes, err := aws.DeploymentPermissionsPolicy(sol)
	if err != nil {
		return nil, fmt.Errorf("failed to generate deployment permissions policy: %w", err)
	}
	if policyBytes != nil {
		files = append(files,
			&kio.RawFile{
				FPath:   "aws_deployment_policy.json",
				Content: policyBytes,
			},
		)
	}

	err = kio.OutputTo(files, request.OutputDir)
	if err != nil {
		return nil, fmt.Errorf("failed to write output files: %w", err)
	}

	return sol, engineErr
}

type iacRequest struct {
	PulumiAppName string
	Solution      solution.Solution
	OutputDir     string
}

var cachedKb *knowledgebase.KnowledgeBase

func (g *InfraGenerator) generateIac(request iacRequest) error {
	var files []kio.File

	var kb *knowledgebase.KnowledgeBase
	var err error
	if cachedKb == nil {
		kb, err = reader.NewKBFromFs(templates.ResourceTemplates, templates.EdgeTemplates, templates.Models)
		if err != nil {
			return err
		}
		cachedKb = kb
	}
	kb = cachedKb

	pulumiPlugin := iac.Plugin{
		Config: &iac.PulumiConfig{AppName: request.PulumiAppName},
		KB:     kb,
	}
	iacFiles, err := pulumiPlugin.Translate(request.Solution)
	if err != nil {
		return err
	}
	files = append(files, iacFiles...)

	err = kio.OutputTo(files, request.OutputDir)
	if err != nil {
		return err
	}

	return nil
}
