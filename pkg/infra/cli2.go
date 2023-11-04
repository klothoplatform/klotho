package infra

import (
	"fmt"
	"os"

	"github.com/klothoplatform/klotho/pkg/config"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	engine "github.com/klothoplatform/klotho/pkg/engine2"
	"github.com/klothoplatform/klotho/pkg/infra/iac3"
	"github.com/klothoplatform/klotho/pkg/infra/kubernetes"
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
		return fmt.Errorf("failed to open input graph: %w", err)
	}
	defer inputF.Close()
	var input construct.YamlGraph
	err = yaml.NewDecoder(inputF).Decode(&input)
	if err != nil {
		return err
	}

	kb, err := knowledgebase.NewKBFromFs(templates.ResourceTemplates, templates.EdgeTemplates, templates.Models)
	if err != nil {
		return err
	}

	solCtx := engine.NewSolutionContext(kb)
	err = solCtx.LoadGraph(input.Graph)
	if err != nil {
		return err
	}
	kubernetesPlugin := kubernetes.Plugin{
		Config: &config.Application{AppName: generateIacCfg.appName},
		KB:     kb,
	}
	k8sfiles, err := kubernetesPlugin.Translate(solCtx)
	if err != nil {
		return err
	}
	files = append(files, k8sfiles...)
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
