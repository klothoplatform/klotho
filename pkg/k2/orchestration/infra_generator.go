package orchestration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	engine_errs "github.com/klothoplatform/klotho/pkg/engine/errors"
	"github.com/klothoplatform/klotho/pkg/engine/solution"
	"github.com/klothoplatform/klotho/pkg/infra/iac"
	kio "github.com/klothoplatform/klotho/pkg/io"
	"github.com/klothoplatform/klotho/pkg/knowledgebase"
	"github.com/klothoplatform/klotho/pkg/knowledgebase/reader"
	"github.com/klothoplatform/klotho/pkg/logging"
	"github.com/klothoplatform/klotho/pkg/provider/aws"
	"github.com/klothoplatform/klotho/pkg/templates"
	"go.uber.org/zap"
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
	sol, err := g.Engine.Run(ctx, &request.SolveRequest)
	if err != nil {
		return nil, err
	}
	log := logging.GetLogger(ctx)

	log.Info("Engine finished running... Generating views")

	var files []kio.File

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

func writeEngineErrsJson(errs []engine_errs.EngineError, out io.Writer) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	// NOTE: since this isn't used in a web context (it's a CLI), we can disable escaping.
	enc.SetEscapeHTML(false)

	outErrs := make([]map[string]any, len(errs))
	for i, e := range errs {
		outErrs[i] = e.ToJSONMap()
		outErrs[i]["error_code"] = e.ErrorCode()
		wrapped := errors.Unwrap(e)
		if wrapped != nil {
			outErrs[i]["error"] = engine_errs.ErrorsToTree(wrapped)
		}
	}
	return enc.Encode(outErrs)
}

func extractEngineErrors(err error) []engine_errs.EngineError {
	if err == nil {
		return nil
	}
	var errs []engine_errs.EngineError
	queue := []error{err}
	for len(queue) > 0 {
		err := queue[0]
		queue = queue[1:]
		switch err := err.(type) {
		case engine_errs.EngineError:
			errs = append(errs, err)
		case interface{ Unwrap() []error }:
			queue = append(queue, err.Unwrap()...)
		case interface{ Unwrap() error }:
			queue = append(queue, err.Unwrap())
		}
	}
	if len(errs) == 0 {
		errs = append(errs, engine_errs.InternalError{Err: err})
	}
	return errs
}

func writeDebugGraphs(sol solution.Solution) {
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		err := engine.GraphToSVG(sol.KnowledgeBase(), sol.DataflowGraph(), "dataflow")
		if err != nil {
			zap.S().Named("engine").Errorf("failed to write dataflow graph: %w", err)
		}
	}()
	go func() {
		defer wg.Done()
		err := engine.GraphToSVG(sol.KnowledgeBase(), sol.DeploymentGraph(), "iac")
		if err != nil {
			zap.S().Named("engine").Errorf("failed to write iac graph: %w", err)
		}
	}()
	wg.Wait()
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
