package infra

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	"path/filepath"
	"runtime/pprof"

	construct "github.com/klothoplatform/klotho/pkg/construct"
	engine "github.com/klothoplatform/klotho/pkg/engine"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	"github.com/klothoplatform/klotho/pkg/infra/iac"
	"github.com/klothoplatform/klotho/pkg/infra/kubernetes"
	statereader "github.com/klothoplatform/klotho/pkg/infra/state_reader"
	statetemplate "github.com/klothoplatform/klotho/pkg/infra/state_reader/state_template"
	kio "github.com/klothoplatform/klotho/pkg/io"
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

var getImportConstraintsCfg struct {
	provider   string
	inputGraph string
	stateFile  string
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

	getLiveStateCmd := &cobra.Command{
		Use:   "GetLiveState",
		Short: "Reads the state file from the provider specified and translates it to Klotho Engine state graph.",
		RunE:  GetLiveState,
	}
	flags = getLiveStateCmd.Flags()
	flags.StringVarP(&getImportConstraintsCfg.provider, "provider", "p", "pulumi", "Provider to use")
	flags.StringVarP(&getImportConstraintsCfg.inputGraph, "input-graph", "i", "", "Input graph to use to provide additional context to the state file.")
	flags.StringVarP(&getImportConstraintsCfg.stateFile, "state-file", "s", "", "State file to use")
	root.AddCommand(getLiveStateCmd)

	return nil
}

func GetLiveState(cmd *cobra.Command, args []string) error {
	log := zap.S().Named("LiveState")

	kb, err := reader.NewKBFromFs(templates.ResourceTemplates, templates.EdgeTemplates, templates.Models)
	if err != nil {
		return err
	}
	log.Info("Loaded knowledge base")
	templates, err := statetemplate.LoadStateTemplates(getImportConstraintsCfg.provider)
	if err != nil {
		return err
	}
	log.Info("Loaded state templates")
	// read in the state file
	if getImportConstraintsCfg.stateFile == "" {
		log.Error("State file path is empty")
		return errors.New("state file path is empty")
	}
	log.Info("Reading state file")
	stateBytes, err := os.ReadFile(getImportConstraintsCfg.stateFile)
	if err != nil {
		log.Error("Failed to read state file", zap.Error(err))
		return err
	}
	var input engine.FileFormat
	if getImportConstraintsCfg.inputGraph != "" {
		inputF, err := os.Open(getImportConstraintsCfg.inputGraph)
		if err != nil {
			return err
		}
		defer inputF.Close()
		err = yaml.NewDecoder(inputF).Decode(&input)
		if err != nil {
			log.Error("Failed to decode input graph", zap.Error(err))
			return err
		}
	}
	bytesReader := bytes.NewReader(stateBytes)
	reader := statereader.NewPulumiReader(input.Graph, templates, kb)
	result, err := reader.ReadState(bytesReader)
	if err != nil {
		return err
	}
	enc := yaml.NewEncoder(os.Stdout)
	return enc.Encode(construct.YamlGraph{Graph: result})
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

	var files []kio.File
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

	solCtx := engine.NewSolution(kb, "", &constraints.Constraints{})
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

	err = kio.OutputTo(files, generateIacCfg.outputDir)
	if err != nil {
		return err
	}
	return nil
}
