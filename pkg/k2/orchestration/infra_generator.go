package orchestration

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine"
	"github.com/klothoplatform/klotho/pkg/engine/solution"
	"github.com/klothoplatform/klotho/pkg/infra/iac"
	kio "github.com/klothoplatform/klotho/pkg/io"
	"github.com/klothoplatform/klotho/pkg/knowledgebase/reader"
	"github.com/klothoplatform/klotho/pkg/logging"
	"github.com/klothoplatform/klotho/pkg/provider/aws"
	"github.com/klothoplatform/klotho/pkg/templates"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

type (
	InfraGenerator struct {
		Engine *engine.Engine
		FS     afero.Fs
	}

	InfraRequest struct {
		engine.SolveRequest
		OutputDir string
	}
)

func NewInfraGenerator(fs afero.Fs) (*InfraGenerator, error) {
	kb, err := reader.NewKBFromFs(templates.ResourceTemplates, templates.EdgeTemplates, templates.Models)
	if err != nil {
		return nil, err
	}
	return &InfraGenerator{
		Engine: engine.NewEngine(kb),
		FS:     fs,
	}, nil
}

func (g *InfraGenerator) writeYamlFile(outDir string, path string, v any) error {
	if !strings.HasPrefix(path, outDir) {
		path = filepath.Join(outDir, path)
	}
	err := g.FS.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return err
	}

	f, err := g.FS.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	return yaml.NewEncoder(f).Encode(v)
}

func (g *InfraGenerator) writeKFile(outDir string, kf kio.File) error {
	path := kf.Path()
	if !strings.HasPrefix(path, outDir) {
		path = filepath.Join(outDir, path)
	}

	err := g.FS.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return err
	}

	f, err := g.FS.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = kf.WriteTo(f)
	return err
}

func (g *InfraGenerator) Run(ctx context.Context, req engine.SolveRequest, outDir string) (solution.Solution, error) {
	if err := g.writeYamlFile(outDir, "engine_input.yaml", req); err != nil {
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

	var fileWriteErrs []error

	log.Info("Serializing constraints")
	vizFiles, err := g.Engine.VisualizeViews(sol)
	if err != nil {
		return nil, fmt.Errorf("failed to generate views %w", err)
	}
	for _, f := range vizFiles {
		fileWriteErrs = append(fileWriteErrs, g.writeKFile(request.OutputDir, f))
	}

	log.Info("Generating resources.yaml")
	fileWriteErrs = append(fileWriteErrs, g.writeYamlFile(
		request.OutputDir,
		"resources.yaml",
		construct.YamlGraph{Graph: sol.DataflowGraph(), Outputs: sol.Outputs()},
	))

	policyBytes, err := aws.DeploymentPermissionsPolicy(sol)
	if err != nil {
		return nil, fmt.Errorf("failed to generate deployment permissions policy: %w", err)
	}
	if policyBytes != nil {
		fileWriteErrs = append(fileWriteErrs,
			g.writeKFile(request.OutputDir, &kio.RawFile{
				FPath:   "aws_deployment_policy.json",
				Content: policyBytes,
			}),
		)
	}

	return sol, errors.Join(fileWriteErrs...)
}

type iacRequest struct {
	PulumiAppName string
	Solution      solution.Solution
	OutputDir     string
}

func (g *InfraGenerator) generateIac(request iacRequest) error {
	pulumiPlugin := iac.Plugin{
		Config: &iac.PulumiConfig{AppName: request.PulumiAppName},
		KB:     g.Engine.Kb,
	}
	iacFiles, err := pulumiPlugin.Translate(request.Solution)
	if err != nil {
		return err
	}
	var fileWriteErrs []error
	for _, f := range iacFiles {
		fileWriteErrs = append(fileWriteErrs, g.writeKFile(request.OutputDir, f))
	}

	return errors.Join(fileWriteErrs...)
}
