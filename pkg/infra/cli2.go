package infra

import (
	"errors"
	"fmt"
	"os"

	"github.com/klothoplatform/klotho/pkg/config"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	"github.com/klothoplatform/klotho/pkg/infra/iac3"
	"github.com/klothoplatform/klotho/pkg/io"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/klothoplatform/klotho/pkg/templates"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func AddIacCli(root *cobra.Command) error {
	generateCmd := &cobra.Command{
		Use:   "Generate",
		Short: "Generate IaC for a given graph",
		RunE:  GenerateIac,
	}
	flags := generateCmd.Flags()
	flags.StringVarP(&generateIacCfg.provider, "provider", "p", "pulumi", "Provider to use")
	flags.StringVarP(&generateIacCfg.inputGraph, "input-graph", "i", "", "Input graph to use")
	flags.StringVarP(&generateIacCfg.outputDir, "output-dir", "o", "", "Output directory to use")
	flags.StringVarP(&generateIacCfg.appName, "app-name", "a", "", "App name to use")
	root.AddCommand(generateCmd)
	return nil
}

func GenerateIac(cmd *cobra.Command, args []string) error {
	var files []io.File
	if generateIacCfg.inputGraph == "" {
		return fmt.Errorf("input graph required")
	}
	inputF, err := os.Open(generateIacCfg.inputGraph)
	if err != nil {
		return err
	}
	defer inputF.Close()
	var input construct.YamlGraph
	err = yaml.NewDecoder(inputF).Decode(&input)
	if err != nil {
		return err
	}

	kb, err := knowledgebase.NewKBFromFs(templates.ResourceTemplates, templates.EdgeTemplates)
	if err != nil {
		return err
	}

	solCtx, err := inputToSolCtx(input.Graph, kb)
	if err != nil {
		return err
	}

	switch generateIacCfg.provider {
	case "pulumi":
		pulumiPlugin := iac3.Plugin{
			Config: &config.Application{AppName: generateIacCfg.appName},
			KB:     kb,
		}
		iacFiles, err := pulumiPlugin.Translate(solCtx)
		if err != nil {
			return err
		}
		files = append(files, iacFiles...)
	default:
		return fmt.Errorf("provider %s not supported", generateIacCfg.provider)
	}

	err = io.OutputTo(files, generateIacCfg.outputDir)
	if err != nil {
		return err
	}
	return nil
}

func inputToSolCtx(input construct.Graph, kb *knowledgebase.KnowledgeBase) (solution_context.SolutionContext, error) {
	solCtx := solution_context.NewSolutionContext(kb)
	cfgCtx := knowledgebase.ConfigTemplateContext{
		DAG: solCtx,
		KB:  kb,
	}

	resources, err := construct.ToplogicalSort(input)
	if err != nil {
		return solCtx, err
	}
	for _, r := range resources {
		res, err := input.Vertex(r)
		if err != nil {
			return solCtx, fmt.Errorf("error getting resource %s: %w", r, err)
		}
		tmpl, err := kb.GetResourceTemplate(r)
		if err != nil {
			return solCtx, fmt.Errorf("error getting resource template %s: %w", r, err)
		}

		data := knowledgebase.ConfigTemplateData{Resource: r}

		var errs error
		for _, prop := range tmpl.Properties {
			path, err := res.PropertyPath(prop.Name)
			if err != nil {
				errs = errors.Join(errs, err)
				continue
			}
			preXform := path.Get()
			if preXform == nil {
				continue
			}
			val, err := kb.TransformToPropertyValue(res, prop.Name, preXform, cfgCtx, data)
			if err != nil {
				errs = errors.Join(errs, err)
				continue
			}
			err = path.Set(val)
			if err != nil {
				errs = errors.Join(errs, err)
				continue
			}
		}
		if errs != nil {
			return solCtx, errs
		}

		err = errors.Join(
			solCtx.GetDataflowGraph().AddVertex(res),
			solCtx.GetDeploymentGraph().AddVertex(res),
		)
		if err != nil {
			return solCtx, fmt.Errorf("error adding resource %s: %w", r, err)
		}
	}

	adj, err := input.AdjacencyMap()
	if err != nil {
		return solCtx, err
	}
	for _, r := range adj {
		for _, edge := range r {
			tmpl := kb.GetEdgeTemplate(edge.Source, edge.Target)
			if tmpl == nil {
				return solCtx, fmt.Errorf("edge template not found for %s -> %s", edge.Source, edge.Target)
			}
			err = solCtx.GetDataflowGraph().AddEdge(edge.Source, edge.Target)
			if tmpl.DeploymentOrderReversed {
				err = errors.Join(err, solCtx.GetDeploymentGraph().AddEdge(edge.Target, edge.Source))
			} else {
				err = errors.Join(err, solCtx.GetDeploymentGraph().AddEdge(edge.Source, edge.Target))
			}
			if err != nil {
				return solCtx, err
			}
		}
	}
	return solCtx, nil
}
