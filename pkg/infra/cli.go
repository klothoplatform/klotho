package infra

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/graph_loader"
	"github.com/klothoplatform/klotho/pkg/infra/iac2"
	"github.com/klothoplatform/klotho/pkg/io"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type IacCli struct {
	Graph *construct.ResourceGraph
}

var generateIacCfg struct {
	provider   string
	inputGraph string
	outputDir  string
}

func (i *IacCli) AddIacCli(root *cobra.Command) error {
	generateCmd := &cobra.Command{
		Use:   "Generate",
		Short: "Generate IaC for a given graph",
		RunE:  i.GenerateIac,
	}
	flags := generateCmd.Flags()
	flags.StringVarP(&generateIacCfg.provider, "provider", "p", "aws", "Provider to use")
	flags.StringVarP(&generateIacCfg.inputGraph, "inputGraph", "i", "", "Input graph to use")
	flags.StringVarP(&generateIacCfg.outputDir, "outputDir", "o", "", "Output directory to use")
	root.AddCommand(generateCmd)
	return nil
}

func (i *IacCli) GenerateIac(cmd *cobra.Command, args []string) error {
	var files []io.File

	if generateIacCfg.inputGraph != "" {
		rg, err := graph_loader.LoadResourceGraphFromFile(generateIacCfg.inputGraph)
		if err != nil {
			return errors.Errorf("failed to load construct graph: %s", err.Error())
		}
		i.Graph = rg
	} else {
		return fmt.Errorf("input graph required")
	}

	files = append(files, i.Graph.OutputResourceFiles()...)

	switch generateIacCfg.provider {
	case "pulumi":
		pulumiPlugin := iac2.Plugin{}
		iacFiles, err := pulumiPlugin.Translate(i.Graph)
		if err != nil {
			return err
		}
		files = append(files, iacFiles...)
	default:
		return fmt.Errorf("provider %s not supported", generateIacCfg.provider)
	}

	err := io.OutputTo(files, generateIacCfg.outputDir)
	if err != nil {
		return err
	}
	return nil
}
