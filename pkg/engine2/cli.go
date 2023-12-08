package engine2

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime/pprof"
	"strings"
	"sync"

	"github.com/iancoleman/strcase"
	"github.com/klothoplatform/klotho/pkg/analytics"
	"github.com/klothoplatform/klotho/pkg/closenicely"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/constraints"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	"github.com/klothoplatform/klotho/pkg/io"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/klothoplatform/klotho/pkg/knowledge_base2/reader"
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
	profileTo  string
}

var architectureEngineCfg struct {
	provider    string
	guardrails  string
	inputGraph  string
	constraints string
	outputDir   string
	verbose     bool
}

var getValidEdgeTargetsCfg struct {
	guardrails string
	inputGraph string
	configFile string
	outputDir  string
	verbose    bool
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
	flags.StringVar(&engineCfg.profileTo, "profiling", "", "Profile to file")

	getPossibleEdgesCmd := &cobra.Command{
		Use:     "GetValidEdgeTargets",
		Short:   "Get the valid topological edge targets for the supplied configuration and input graph",
		GroupID: engineGroup.ID,
		RunE:    em.GetValidEdgeTargets,
	}

	flags = getPossibleEdgesCmd.Flags()
	flags.StringVar(&getValidEdgeTargetsCfg.guardrails, "guardrails", "", "Guardrails file")
	flags.StringVarP(&getValidEdgeTargetsCfg.inputGraph, "input-graph", "i", "", "Input graph file")
	flags.StringVarP(&getValidEdgeTargetsCfg.configFile, "config", "c", "", "config file")
	flags.StringVarP(&getValidEdgeTargetsCfg.outputDir, "output-dir", "o", "", "Output directory")
	flags.BoolVarP(&getValidEdgeTargetsCfg.verbose, "verbose", "v", false, "Verbose flag")
	flags.BoolVar(&engineCfg.jsonLog, "json-log", false, "Output logs in JSON format.")
	flags.StringVar(&engineCfg.profileTo, "profiling", "", "Profile to file")

	root.AddGroup(engineGroup)
	root.AddCommand(listResourceTypesCmd)
	root.AddCommand(listAttributesCmd)
	root.AddCommand(runCmd)
	root.AddCommand(getPossibleEdgesCmd)
}

func (em *EngineMain) AddEngine() error {
	kb, err := reader.NewKBFromFs(templates.ResourceTemplates, templates.EdgeTemplates, templates.Models)
	if err != nil {
		return err
	}
	em.Engine = NewEngine(kb)
	return nil
}

type resourceInfo struct {
	Classifications []string          `json:"classifications"`
	DisplayName     string            `json:"displayName"`
	Properties      map[string]any    `json:"properties"`
	Views           map[string]string `json:"views"`
}

var validationFields = []string{"MinLength", "MaxLength", "MinValue", "MaxValue", "AllowedValues"}

func addSubProperties(properties map[string]any, subProperties map[string]knowledgebase.Property) {
	for _, subProperty := range subProperties {
		details := subProperty.Details()
		properties[details.Name] = map[string]any{
			"type":                  subProperty.Type(),
			"deployTime":            details.DeployTime,
			"configurationDisabled": details.ConfigurationDisabled,
			"required":              details.Required,
		}
		for _, validationField := range validationFields {
			valField := reflect.ValueOf(subProperty).Elem().FieldByName(validationField)
			if valField.IsValid() && !valField.IsZero() {
				val := valField.Interface()
				properties[details.Name].(map[string]any)[strcase.ToLowerCamel(validationField)] = val
			}
		}
		if subProperty.SubProperties() != nil {
			properties[details.Name].(map[string]any)["properties"] = map[string]any{}
			addSubProperties(properties[details.Name].(map[string]any)["properties"].(map[string]any), subProperty.SubProperties())
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
		addSubProperties(properties, resourceType.Properties)
		typeAndClassifications[resourceType.QualifiedTypeName] = resourceInfo{
			Classifications: resourceType.Classification.Is,
			Properties:      properties,
			DisplayName:     resourceType.DisplayName,
			Views:           resourceType.Views,
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
	if engineCfg.profileTo != "" {
		err := os.MkdirAll(filepath.Dir(engineCfg.profileTo), 0755)
		if err != nil {
			return fmt.Errorf("failed to create profile directory: %w", err)
		}
		profileF, err := os.OpenFile(engineCfg.profileTo, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
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

	context := &EngineContext{}

	if architectureEngineCfg.inputGraph != "" {
		var input FileFormat
		zap.S().Info("Loading input graph")
		inputF, err := os.Open(architectureEngineCfg.inputGraph)
		if err != nil {
			return err
		}
		defer inputF.Close()
		err = yaml.NewDecoder(inputF).Decode(&input)
		if err != nil {
			return err
		}
		context.InitialState = input.Graph
		if architectureEngineCfg.constraints == "" {
			context.Constraints = input.Constraints
		}
	} else {
		context.InitialState = construct.NewGraph()
	}
	zap.S().Info("Loading constraints")

	if architectureEngineCfg.constraints != "" {
		runConstraints, err := constraints.LoadConstraintsFromFile(architectureEngineCfg.constraints)
		if err != nil {
			return errors.Errorf("failed to load constraints: %s", err.Error())
		}
		context.Constraints = runConstraints
	}

	zap.S().Info("Running engine")
	err = em.Engine.Run(context)
	if err != nil {
		return errors.Errorf("failed to run engine: %s", err.Error())
	}
	writeDebugGraphs(context.Solutions[0])
	zap.S().Info("Engine finished running... Generating views")
	var files []io.File
	files, err = em.Engine.VisualizeViews(context.Solutions[0])
	if err != nil {
		return errors.Errorf("failed to generate views %s", err.Error())
	}
	zap.S().Info("Generating resources.yaml")
	b, err := yaml.Marshal(construct.YamlGraph{Graph: context.Solutions[0].DataflowGraph()})
	if err != nil {
		return errors.Errorf("failed to marshal graph: %s", err.Error())
	}
	files = append(files, &io.RawFile{
		FPath:   "resources.yaml",
		Content: b,
	},
	)

	configErrors, configErr := em.Engine.getPropertyValidation(context.Solutions[0])
	if len(configErrors) > 0 {
		configErrorData, err := json.Marshal(configErrors)
		if err != nil {
			return errors.Errorf("failed to marshal config errors: %s", err.Error())
		}
		files = append(files, &io.RawFile{
			FPath:   "config_errors.json",
			Content: configErrorData,
		})
	}

	err = io.OutputTo(files, architectureEngineCfg.outputDir)
	if err != nil {
		return errors.Errorf("failed to write output files: %s", err.Error())
	}
	if configErr != nil {
		return ConfigValidationError{Err: configErr}
	}
	return nil
}

func (em *EngineMain) GetValidEdgeTargets(cmd *cobra.Command, args []string) error {
	if engineCfg.profileTo != "" {
		err := os.MkdirAll(filepath.Dir(engineCfg.profileTo), 0755)
		if err != nil {
			return fmt.Errorf("failed to create profile directory: %w", err)
		}
		profileF, err := os.OpenFile(engineCfg.profileTo, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
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
	zap.S().Info("loading config")

	inputF, err := os.ReadFile(getValidEdgeTargetsCfg.inputGraph)
	if err != nil {
		return err
	}

	config, err := ReadGetValidEdgeTargetsConfig(getValidEdgeTargetsCfg.configFile)
	if err != nil {
		return errors.Errorf("failed to load constraints: %s", err.Error())
	}
	context := &GetPossibleEdgesContext{
		InputGraph:                inputF,
		GetValidEdgeTargetsConfig: config,
	}

	zap.S().Info("getting valid edge targets")
	validTargets, err := em.Engine.GetValidEdgeTargets(context)
	if err != nil {
		return errors.Errorf("failed to run engine: %s", err.Error())
	}

	zap.S().Info("writing output files")
	b, err := yaml.Marshal(validTargets)
	if err != nil {
		return errors.Errorf("failed to marshal possible edges: %s", err.Error())
	}
	var files []io.File
	files = append(files, &io.RawFile{
		FPath:   "valid_edge_targets.yaml",
		Content: b,
	})

	err = io.OutputTo(files, getValidEdgeTargetsCfg.outputDir)
	if err != nil {
		return errors.Errorf("failed to write output files: %s", err.Error())
	}
	return nil
}

func writeDebugGraphs(sol solution_context.SolutionContext) {
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		err := construct.GraphToSVG(sol.DataflowGraph(), "dataflow")
		if err != nil {
			zap.S().Errorf("failed to write dataflow graph: %s", err.Error())
		}
	}()
	go func() {
		defer wg.Done()
		err := construct.GraphToSVG(sol.DeploymentGraph(), "iac")
		if err != nil {
			zap.S().Errorf("failed to write iac graph: %s", err.Error())
		}
	}()
	wg.Wait()
}
