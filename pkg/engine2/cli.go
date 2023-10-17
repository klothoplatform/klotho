package engine2

import (
	"encoding/json"
	"fmt"
	"os"
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
		err := zap.RegisterEncoder(consoleEncoderName, func(zcfg zapcore.EncoderConfig) (zapcore.Encoder, error) {
			return logging.NewConsoleEncoder(architectureEngineCfg.verbose, hadWarnings, hadErrors), nil
		})
		if err != nil {
			return nil, err
		}
		zapCfg.Encoding = consoleEncoderName
	}

	return zapCfg.Build(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		trackingCore := analyticsClient.NewFieldListener(zapcore.WarnLevel)
		return zapcore.NewTee(core, trackingCore)
	}))
}

func (em *EngineMain) AddEngineCli(root *cobra.Command) {
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
}

func (em *EngineMain) AddEngine() error {
	kb, err := knowledgebase.NewKBFromFs(templates.ResourceTemplates, templates.EdgeTemplates, templates.Models)
	if err != nil {
		return err
	}
	em.Engine = NewEngine(kb)
	return nil
}

type resourceInfo struct {
	Classifications []string       `json:"classifications"`
	DisplayName     string         `json:"displayName"`
	Properties      map[string]any `json:"properties"`
}

func addSubProperties(properties map[string]any, subProperties map[string]*knowledgebase.Property) {
	for _, subProperty := range subProperties {
		properties[subProperty.Name] = map[string]any{
			"type": subProperty.Type,
		}
		if subProperty.Properties != nil {
			properties[subProperty.Name].(map[string]any)["properties"] = map[string]any{}
			addSubProperties(properties[subProperty.Name].(map[string]any)["properties"].(map[string]any), subProperty.Properties)
		}
	}

}

func (em *EngineMain) ListResourceTypes(cmd *cobra.Command, args []string) error {
	err := em.AddEngine()
	if err != nil {
		return err
	}
	resourceTypes := em.Engine.Kb.ListResources()
	typeAndClassifications := map[string]resourceInfo{}

	for _, resourceType := range resourceTypes {
		properties := map[string]any{}
		for _, property := range resourceType.Properties {
			properties[property.Name] = map[string]any{
				"type": property.Type,
			}
			if property.Properties != nil {
				properties[property.Name].(map[string]any)["properties"] = map[string]any{}
				addSubProperties(properties[property.Name].(map[string]any)["properties"].(map[string]any), property.Properties)
			}
		}
		typeAndClassifications[resourceType.QualifiedTypeName] = resourceInfo{
			Classifications: resourceType.Classification.Is,
			Properties:      properties,
			DisplayName:     resourceType.DisplayName,
		}
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
	var input construct.YamlGraph
	if architectureEngineCfg.inputGraph != "" {
		inputF, err := os.Open(architectureEngineCfg.inputGraph)
		if err != nil {
			return err
		}
		defer inputF.Close()
		err = yaml.NewDecoder(inputF).Decode(&input)
		if err != nil {
			return err
		}
	} else {
		input.Graph = construct.NewGraph()
	}

	runConstraints, err := constraints.LoadConstraintsFromFile(architectureEngineCfg.constraints)
	if err != nil {
		return errors.Errorf("failed to load constraints: %s", err.Error())
	}
	context := &EngineContext{
		InitialState: input.Graph,
		Constraints:  runConstraints,
	}

	err = em.Engine.Run(context)
	if err != nil {
		fmt.Println(err)
		return errors.Errorf("failed to run engine: %s", err.Error())
	}
	var files []io.File
	files, err = em.Engine.VisualizeViews(context.Solutions[0])
	if err != nil {
		return errors.Errorf("failed to generate views %s", err.Error())
	}
	b, err := yaml.Marshal(construct.YamlGraph{Graph: context.Solutions[0].DataflowGraph()})
	if err != nil {
		return errors.Errorf("failed to marshal graph: %s", err.Error())
	}
	files = append(files, &io.RawFile{
		FPath:   "resources.yaml",
		Content: b,
	},
	)

	err = io.OutputTo(files, architectureEngineCfg.outputDir)
	if err != nil {
		return errors.Errorf("failed to write output files: %s", err.Error())
	}
	return nil
}
