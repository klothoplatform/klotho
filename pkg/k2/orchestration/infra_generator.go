package orchestration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine"
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

func writeYamlFile(path string, v any) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	return yaml.NewEncoder(f).Encode(v)
}

func (g *InfraGenerator) Run(ctx context.Context, req engine.SolveRequest, outDir string) (solution.Solution, error) {
	if err := writeYamlFile(filepath.Join(outDir, "engine_input.yaml"), req); err != nil {
		return nil, fmt.Errorf("failed to write engine input: %w", err)
	}

	sol, errs := g.resolveResources(ctx, InfraRequest{
		SolveRequest: req,
		OutputDir:    outDir,
	})
	if errs != nil {
		return nil, fmt.Errorf("failed to resolve resources: %v", errs)
	}

	err := g.generateIac(iacRequest{
		PulumiAppName: "k2",
		Solution:      sol,
		OutputDir:     outDir,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate iac: %w", err)

	}
	return sol, nil
}

func (g *InfraGenerator) resolveResources(ctx context.Context, request InfraRequest) (solution.Solution, error) {
	log := logging.GetLogger(ctx)
	log.Info("Running engine")

	sol, engineErr := g.Engine.Run(ctx, &request.SolveRequest)
	if engineErr != nil {
		return nil, fmt.Errorf("Engine failed: %w", engineErr)
	}

	log.Info("Generating views")

	var files []kio.File

	log.Info("Serializing constraints")
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

	return sol, nil
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
