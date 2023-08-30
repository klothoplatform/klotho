package engine

import (
	"fmt"
	"os"
	"strings"

	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/engine/input"
	"github.com/klothoplatform/klotho/pkg/io"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider"
	"github.com/klothoplatform/klotho/pkg/provider/docker"
	"github.com/klothoplatform/klotho/pkg/provider/kubernetes"
	kubernetesKb "github.com/klothoplatform/klotho/pkg/provider/kubernetes/knowledgebase"
	"github.com/klothoplatform/klotho/pkg/provider/providers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type EngineMain struct {
	Engine *Engine
}

var engineCfg struct {
	provider   string
	guardrails string
}

var listResourceFieldsConfig struct {
	provider   string
	resource   string
	guardrails string
}

var architectureEngineCfg struct {
	provider    string
	guardrails  string
	inputFile   string
	constraints string
	outputDir   string
}

func (em *EngineMain) AddEngineCli(root *cobra.Command) error {
	engineGroup := &cobra.Group{
		ID:    "engine",
		Title: "engine",
	}
	listResourceTypesCmd := &cobra.Command{
		Use:     "ListResourceTypes",
		Short:   "List resource types available in the klotho engine",
		GroupID: engineGroup.ID,
		RunE:    em.ListResourceTypes,
	}

	flags := listResourceTypesCmd.Flags()
	flags.StringVarP(&engineCfg.provider, "provider", "p", "aws", "Provider to use")
	flags.StringVar(&engineCfg.guardrails, "guardrails", "", "Guardrails file")

	listAttributesCmd := &cobra.Command{
		Use:     "ListAttributes",
		Short:   "List attributes available in the klotho engine",
		GroupID: engineGroup.ID,
		RunE:    em.ListAttributes,
	}

	flags = listAttributesCmd.Flags()
	flags.StringVarP(&engineCfg.provider, "provider", "p", "aws", "Provider to use")
	flags.StringVar(&engineCfg.guardrails, "guardrails", "", "Guardrails file")

	listResourceFieldsCmd := &cobra.Command{
		Use:     "ListResourceTypesFields",
		Short:   "List a provider resource's fields",
		GroupID: engineGroup.ID,
		RunE:    em.ListResourceFields,
	}

	flags = listResourceFieldsCmd.Flags()
	flags.StringVarP(&listResourceFieldsConfig.provider, "provider", "p", "aws", "Provider to use")
	flags.StringVarP(&listResourceFieldsConfig.resource, "resource-type", "t", "", "resource type to use")
	flags.StringVar(&listResourceFieldsConfig.guardrails, "guardrails", "", "Guardrails file")

	runCmd := &cobra.Command{
		Use:     "Run",
		Short:   "Run the klotho engine",
		GroupID: engineGroup.ID,
		RunE:    em.RunEngine,
	}

	flags = runCmd.Flags()
	flags.StringVarP(&architectureEngineCfg.provider, "provider", "p", "aws", "Provider to use")
	flags.StringVar(&architectureEngineCfg.guardrails, "guardrails", "", "Guardrails file")
	flags.StringVarP(&architectureEngineCfg.inputFile, "input", "i", "", "Input file")
	flags.StringVarP(&architectureEngineCfg.outputDir, "output-dir", "o", "", "Output directory")

	root.AddGroup(engineGroup)
	root.AddCommand(listResourceTypesCmd)
	root.AddCommand(listAttributesCmd)
	root.AddCommand(listResourceFieldsCmd)
	root.AddCommand(runCmd)
	return nil
}

func (em *EngineMain) AddEngine(providerToAdd string, guardrails string) error {
	cfg := &config.Application{Provider: providerToAdd}
	cloudProvider, err := providers.GetProvider(cfg)
	if err != nil {
		return err
	}
	cloudkb, err := providers.GetKnowledgeBase(cfg)
	if err != nil {
		return err
	}
	kb, err := knowledgebase.MergeKBs([]knowledgebase.EdgeKB{cloudkb, kubernetesKb.KubernetesKB})
	if err != nil {
		return err
	}
	kubernetesProvider := &kubernetes.KubernetesProvider{}
	dockerProvider := &docker.DockerProvider{}
	em.Engine = NewEngine(map[string]provider.Provider{
		cloudProvider.Name():      cloudProvider,
		kubernetesProvider.Name(): kubernetesProvider,
		dockerProvider.Name():     dockerProvider,
	}, kb, types.ListAllConstructs())
	guardrailsBytes, err := readGuardrails(guardrails)
	if err != nil {
		return err
	}
	err = em.Engine.LoadGuardrails(guardrailsBytes)
	if err != nil {
		return err
	}
	return nil
}

func readGuardrails(guardrails string) ([]byte, error) {
	if guardrails != "" {
		f, err := os.ReadFile(guardrails)
		if err != nil {
			return nil, err
		}
		return f, nil
	}
	return nil, nil
}

func (em *EngineMain) ListResourceTypes(cmd *cobra.Command, args []string) error {
	err := em.AddEngine(engineCfg.provider, engineCfg.guardrails)
	if err != nil {
		return err
	}
	resourceTypes := em.Engine.ListResourcesByType()
	fmt.Println(strings.Join(resourceTypes, "\n"))
	return nil
}

func (em *EngineMain) ListAttributes(cmd *cobra.Command, args []string) error {
	err := em.AddEngine(engineCfg.provider, engineCfg.guardrails)
	if err != nil {
		return err
	}
	attributes := em.Engine.ListAttributes()
	fmt.Println(strings.Join(attributes, "\n"))
	return nil
}

func (em *EngineMain) ListResourceFields(cmd *cobra.Command, args []string) error {
	err := em.AddEngine(engineCfg.provider, engineCfg.guardrails)
	if err != nil {
		return err
	}
	fields := em.Engine.ListResourceFields(listResourceFieldsConfig.provider, listResourceFieldsConfig.resource)
	b, err := yaml.Marshal(fields)
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

func (em *EngineMain) createEngine() error {
	err := em.AddEngine(engineCfg.provider, engineCfg.guardrails)
	if err != nil {
		return err
	}

	f, err := os.Open(architectureEngineCfg.inputFile)
	if err != nil {
		return errors.Errorf("failed to open input file: %v", err)
	}
	defer f.Close() // nolint:errcheck

	var input input.Input
	err = yaml.NewDecoder(f).Decode(&input)
	if err != nil {
		return errors.Errorf("failed to decode input file: %v", err)
	}

	err = em.Engine.ContextFromInput(input)
	if err != nil {
		return errors.Errorf("failed to create engine context from input: %v", err)
	}
	return nil
}

func (em *EngineMain) RunEngine(cmd *cobra.Command, args []string) error {
	err := em.createEngine()
	if err != nil {
		return err
	}

	outputGraph, err := em.Engine.Run()
	if err != nil {
		return errors.Errorf("failed to run engine: %v", err)
	}
	err = outputGraph.OutputResourceGraph(architectureEngineCfg.outputDir)
	if err != nil {
		return errors.Errorf("failed to write output graph: %v", err)
	}
	files, err := em.Engine.VisualizeViews()
	if err != nil {
		return errors.Errorf("failed to visualize views: %v", err)
	}
	err = io.OutputTo(files, architectureEngineCfg.outputDir)
	if err != nil {
		return errors.Errorf("failed to write output files: %v", err)
	}
	return nil
}
