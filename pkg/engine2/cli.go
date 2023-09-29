package engine2

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/klothoplatform/klotho/pkg/analytics"
	"github.com/klothoplatform/klotho/pkg/closenicely"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/constraints"
	"github.com/klothoplatform/klotho/pkg/io"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/klothoplatform/klotho/pkg/logging"
	"github.com/klothoplatform/klotho/pkg/templates"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"
)

type EngineMain struct {
	Engine *Engine
}

var engineCfg struct {
	provider   string
	guardrails string
	jsonLog    bool
}

var architectureEngineCfg struct {
	provider    string
	guardrails  string
	inputGraph  string
	constraints string
	outputDir   string
	verbose     bool
}

var hadWarnings = atomic.NewBool(false)
var hadErrors = atomic.NewBool(false)

const consoleEncoderName = "engine-cli"

func init() {
	err := zap.RegisterEncoder(consoleEncoderName, func(zcfg zapcore.EncoderConfig) (zapcore.Encoder, error) {
		return logging.NewConsoleEncoder(architectureEngineCfg.verbose, hadWarnings, hadErrors), nil
	})

	if err != nil {
		panic(err)
	}
}

func setupLogger(analyticsClient *analytics.Client) (*zap.Logger, error) {
	var zapCfg zap.Config
	if architectureEngineCfg.verbose {
		zapCfg = zap.NewDevelopmentConfig()
	} else {
		zapCfg = zap.NewProductionConfig()
	}
	if engineCfg.jsonLog {
		zapCfg.Encoding = "json"
	} else {
		zapCfg.Encoding = consoleEncoderName
	}

	return zapCfg.Build(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		trackingCore := analyticsClient.NewFieldListener(zapcore.WarnLevel)
		return zapcore.NewTee(core, trackingCore)
	}))
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

	runCmd := &cobra.Command{
		Use:     "Run",
		Short:   "Run the klotho engine",
		GroupID: engineGroup.ID,
		RunE:    em.RunEngine,
	}

	flags = runCmd.Flags()
	flags.StringVarP(&architectureEngineCfg.provider, "provider", "p", "aws", "Provider to use")
	flags.StringVar(&architectureEngineCfg.guardrails, "guardrails", "", "Guardrails file")
	flags.StringVarP(&architectureEngineCfg.inputGraph, "input-graph", "i", "", "Input graph file")
	flags.StringVarP(&architectureEngineCfg.constraints, "constraints", "c", "", "Constraints file")
	flags.StringVarP(&architectureEngineCfg.outputDir, "output-dir", "o", "", "Output directory")
	flags.BoolVarP(&architectureEngineCfg.verbose, "verbose", "v", false, "Verbose flag")
	flags.BoolVar(&engineCfg.jsonLog, "json-log", false, "Output logs in JSON format.")

	root.AddGroup(engineGroup)
	root.AddCommand(listResourceTypesCmd)
	root.AddCommand(listAttributesCmd)
	root.AddCommand(runCmd)
	return nil
}

func (em *EngineMain) AddEngine() error {
	kb := knowledgebase.NewKB()
	resourceTemplates, err := knowledgebase.TemplatesFromFs(templates.ResourceTemplates)
	if err != nil {
		return fmt.Errorf("failed to load resource templates: %s", err.Error())
	}
	for _, template := range resourceTemplates {
		err := kb.AddResourceTemplate(template)
		if err != nil {
			return fmt.Errorf("failed to add resource template %s: %s", template.QualifiedTypeName, err.Error())
		}
	}

	edgeTemplates, err := knowledgebase.EdgeTemplatesFromFs(templates.EdgeTemplates)
	if err != nil {
		return fmt.Errorf("failed to load edge templates: %s", err.Error())
	}
	for _, template := range edgeTemplates {
		fmt.Println(template.Source.QualifiedTypeName(), template.Target.QualifiedTypeName())
		err := kb.AddEdgeTemplate(template)
		if err != nil {
			return fmt.Errorf("failed to add edge template %s -> %s: %s",
				template.Source.QualifiedTypeName(), template.Target.QualifiedTypeName(), err.Error())
		}
	}

	em.Engine = NewEngine(kb)
	return nil
}

func (em *EngineMain) ListResourceTypes(cmd *cobra.Command, args []string) error {
	err := em.AddEngine()
	if err != nil {
		return err
	}
	resourceTypes := em.Engine.Kb.ListResources()
	typeAndClassifications := map[string][]string{}
	for _, resourceType := range resourceTypes {
		typeAndClassifications[resourceType.QualifiedTypeName] = resourceType.Classification.Is
	}
	b, err := json.Marshal(typeAndClassifications)
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

func (em *EngineMain) ListAttributes(cmd *cobra.Command, args []string) error {
	err := em.AddEngine()
	if err != nil {
		return err
	}
	attributes := em.Engine.ListAttributes()
	fmt.Println(strings.Join(attributes, "\n"))
	return nil
}

func (em *EngineMain) RunEngine(cmd *cobra.Command, args []string) error {

	// Set up analytics, and hook them up to the logs
	analyticsClient := analytics.NewClient()
	analyticsClient.AppendProperties(map[string]any{})
	z, err := setupLogger(analyticsClient)
	if err != nil {
		return err
	}
	defer closenicely.FuncOrDebug(z.Sync)
	zap.ReplaceGlobals(z)

	err = em.AddEngine()
	if err != nil {
		return err
	}
	// if architectureEngineCfg.inputGraph != "" {
	// 	yamlGraph := construct2.YamlGraph{}

	// 	yamlGraph.UnmarshalYAML()
	// 	if err != nil {
	// 		return errors.Errorf("failed to load construct graph: %s", err.Error())
	// 	}
	// }

	runConstraints, err := constraints.LoadConstraintsFromFile(architectureEngineCfg.constraints)
	if err != nil {
		return errors.Errorf("failed to load constraints: %s", err.Error())
	}
	context := &EngineContext{
		InitialState: construct.NewGraph(),
		Constraints:  runConstraints,
	}

	err = em.Engine.Run(context)
	if err != nil {
		fmt.Println(err)
		return errors.Errorf("failed to run engine: %s", err.Error())
	}
	var files []io.File
	// files, err := em.Engine.VisualizeViews(context.Solutions[0])
	// if err != nil {
	// 	return errors.Errorf("failed to generate views %s", err.Error())
	// }
	b, err := yaml.Marshal(construct.YamlGraph{Graph: context.Solutions[0].GetDataflowGraph()})
	if err != nil {
		return errors.Errorf("failed to marshal graph: %s", err.Error())
	}
	files = append(files, &io.RawFile{
		FPath:   "state.yaml",
		Content: b,
	},
	)

	err = io.OutputTo(files, architectureEngineCfg.outputDir)
	if err != nil {
		return errors.Errorf("failed to write output files: %s", err.Error())
	}
	return nil
}
