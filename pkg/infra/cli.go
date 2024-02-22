package infra

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/pprof"

	construct "github.com/klothoplatform/klotho/pkg/construct"
	engine "github.com/klothoplatform/klotho/pkg/engine"
	"github.com/klothoplatform/klotho/pkg/infra/iac"
	"github.com/klothoplatform/klotho/pkg/infra/kubernetes"
	"github.com/klothoplatform/klotho/pkg/io"
	"github.com/klothoplatform/klotho/pkg/knowledgebase/reader"
	"github.com/klothoplatform/klotho/pkg/logging"
	"github.com/klothoplatform/klotho/pkg/templates"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

var commonCfg struct {
	verbose bool
	jsonLog bool
}

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
	flags := root.PersistentFlags()
	flags.BoolVarP(&commonCfg.verbose, "verbose", "v", false, "Verbose flag")
	flags.BoolVar(&commonCfg.jsonLog, "json-log", false, "Output logs in JSON format.")
	root.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		logOpts := logging.LogOpts{
			Verbose:         commonCfg.verbose,
			CategoryLogsDir: "", // IaC doesn't generate enough logs to warrant category-specific logs
		}
		if commonCfg.jsonLog {
			logOpts.Encoding = "json"
		}
		zap.ReplaceGlobals(logOpts.NewLogger())
	}
	root.PersistentPostRun = func(cmd *cobra.Command, args []string) {
		zap.L().Sync() //nolint:errcheck
	}

	generateCmd := &cobra.Command{
		Use:   "Generate",
		Short: "Generate IaC for a given graph",
		RunE:  GenerateIac,
	}
	flags = generateCmd.Flags()
	flags.StringVarP(&generateIacCfg.provider, "provider", "p", "pulumi", "Provider to use")
	flags.StringVarP(&generateIacCfg.inputGraph, "input-graph", "i", "", "Input graph to use")
	flags.StringVarP(&generateIacCfg.outputDir, "output-dir", "o", "", "Output directory to use")
	flags.StringVarP(&generateIacCfg.appName, "app-name", "a", "", "App name to use")
	flags.StringVar(&generateIacCfg.profileTo, "profiling", "", "Profile to file")
	root.AddCommand(generateCmd)
	return nil
}

func GenerateIac(cmd *cobra.Command, args []string) error {
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
		pulumiPlugin := iac.Plugin{
			Config: &iac.PulumiConfig{AppName: generateIacCfg.appName},
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
