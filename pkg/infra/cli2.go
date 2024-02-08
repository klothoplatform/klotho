package infra

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/pprof"

	"github.com/klothoplatform/klotho/pkg/closenicely"
	construct "github.com/klothoplatform/klotho/pkg/construct"
	engine "github.com/klothoplatform/klotho/pkg/engine"
	"github.com/klothoplatform/klotho/pkg/infra/iac3"
	"github.com/klothoplatform/klotho/pkg/infra/kubernetes"
	"github.com/klothoplatform/klotho/pkg/io"
	"github.com/klothoplatform/klotho/pkg/knowledgebase/reader"
	"github.com/klothoplatform/klotho/pkg/templates"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

var generateIacCfg struct {
	provider   string
	inputGraph string
	outputDir  string
	appName    string
	verbose    bool
	jsonLog    bool
	profileTo  string
}

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
	flags.BoolVarP(&generateIacCfg.verbose, "verbose", "v", false, "Verbose flag")
	flags.BoolVar(&generateIacCfg.jsonLog, "json-log", false, "Output logs in JSON format.")
	flags.StringVar(&generateIacCfg.profileTo, "profiling", "", "Profile to file")
	root.AddCommand(generateCmd)
	return nil
}

func setupLogger() (*zap.Logger, error) {
	var zapCfg zap.Config
	if generateIacCfg.verbose {
		zapCfg = zap.NewDevelopmentConfig()
	} else {
		zapCfg = zap.NewProductionConfig()
	}
	if generateIacCfg.jsonLog {
		zapCfg.Encoding = "json"
	} else {
		zapCfg.Encoding = "console"
	}

	return zapCfg.Build()
}

func GenerateIac(cmd *cobra.Command, args []string) error {
	z, err := setupLogger()
	if err != nil {
		return err
	}
	defer closenicely.FuncOrDebug(z.Sync)
	zap.ReplaceGlobals(z)

	if generateIacCfg.profileTo != "" {
		err := os.MkdirAll(filepath.Dir(generateIacCfg.profileTo), 0755)
		if err != nil {
			return fmt.Errorf("failed to create profile directory: %w", err)
		}
		profileF, err := os.OpenFile(generateIacCfg.profileTo, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open profile file: %w", err)
		}
		defer func() {
			pprof.StopCPUProfile()
			profileF.Close()
		}()
		err = pprof.StartCPUProfile(profileF)
		if err != nil {
			return fmt.Errorf("failed to start profile: %w", err)
		}
	}

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

	kb, err := reader.NewKBFromFs(templates.ResourceTemplates, templates.EdgeTemplates, templates.Models)
	if err != nil {
		return err
	}

	solCtx := engine.NewSolutionContext(kb)
	err = solCtx.LoadGraph(input.Graph)
	if err != nil {
		return err
	}
	kubernetesPlugin := kubernetes.Plugin{
		AppName: generateIacCfg.appName,
		KB:      kb,
	}
	k8sfiles, err := kubernetesPlugin.Translate(solCtx)
	if err != nil {
		return err
	}
	files = append(files, k8sfiles...)
	switch generateIacCfg.provider {
	case "pulumi":
		pulumiPlugin := iac3.Plugin{
			Config: &iac3.PulumiConfig{AppName: generateIacCfg.appName},
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
