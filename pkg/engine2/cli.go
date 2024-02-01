package engine2

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
	engine_errs "github.com/klothoplatform/klotho/pkg/engine2/errors"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	kio "github.com/klothoplatform/klotho/pkg/io"
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

type (
	EngineMain struct {
		Engine *Engine
	}
)

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
		Run: func(cmd *cobra.Command, args []string) {
			exitCode := em.RunEngine(cmd, args)
			os.Exit(exitCode)
		},
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

var validationFields = []string{"MinLength", "MaxLength", "MinValue", "MaxValue", "AllowedValues", "UniqueItems", "UniqueKeys", "MinSize", "MaxSize"}

func addSubProperties(properties map[string]any, subProperties map[string]knowledgebase.Property) {
	for _, subProperty := range subProperties {
		details := subProperty.Details()
		properties[details.Name] = map[string]any{
			"type":                  subProperty.Type(),
			"deployTime":            details.DeployTime,
			"configurationDisabled": details.ConfigurationDisabled,
			"required":              details.Required,
			"description":           details.Description,
			"important":             details.IsImportant,
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

func setupProfiling() func() {
	if engineCfg.profileTo != "" {
		err := os.MkdirAll(filepath.Dir(engineCfg.profileTo), 0755)
		if err != nil {
			panic(fmt.Errorf("failed to create profile directory: %w", err))
		}
		profileF, err := os.OpenFile(engineCfg.profileTo, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if err != nil {
			panic(fmt.Errorf("failed to open profile file: %w", err))
		}
		err = pprof.StartCPUProfile(profileF)
		if err != nil {
			panic(fmt.Errorf("failed to start profile: %w", err))
		}
		return func() {
			pprof.StopCPUProfile()
			profileF.Close()
		}
	}
	return func() {}
}

func (em *EngineMain) Run(context *EngineContext) (int, []engine_errs.EngineError) {
	returnCode := 0
	var engErrs []engine_errs.EngineError

	zap.S().Info("Running engine")
	err := em.Engine.Run(context)
	if err != nil {
		returnCode = 1
		if ee, ok := err.(engine_errs.EngineError); ok {
			engErrs = append(engErrs, ee)
		} else {
			engErrs = append(engErrs, engine_errs.InternalError{Err: engine_errs.ErrorsToTree(err)})
		}
	}

	if len(context.Solutions) > 0 {
		writeDebugGraphs(context.Solutions[0])
		for _, d := range context.Solutions[0].GetDecisions().GetRecords() {
			d, ok := d.(solution_context.MaybeErroDecision)
			if !ok {
				continue
			}
			ee := d.AsEngineError()
			if ee == nil {
				continue
			}
			engErrs = append(engErrs, ee)
			if returnCode != 1 {
				returnCode = 2
			}
		}
	}

	return returnCode, engErrs
}

func writeEngineErrsJson(errs []engine_errs.EngineError, out io.Writer) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	// NOTE: since this isn't used in a web context (it's a CLI), we can disable escaping.
	enc.SetEscapeHTML(false)

	outErrs := make([]map[string]any, len(errs))
	for i, e := range errs {
		outErrs[i] = e.ToJSONMap()
		outErrs[i]["error_code"] = e.ErrorCode()
		wrapped := errors.Unwrap(e)
		if wrapped != nil {
			outErrs[i]["error"] = engine_errs.ErrorsToTree(wrapped)
		}
	}
	return enc.Encode(outErrs)
}

func (em *EngineMain) RunEngine(cmd *cobra.Command, args []string) (exitCode int) {
	var engErrs []engine_errs.EngineError
	internalError := func(err error) {
		engErrs = append(engErrs, engine_errs.InternalError{Err: err})
		exitCode = 1
	}

	defer func() { // defer functions execute in FILO order, so this executes after the 'recover'.
		err := writeEngineErrsJson(engErrs, os.Stdout)
		if err != nil {
			zap.S().Errorf("failed to output errors to stdout: %v", err)
		}
	}()
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		zap.S().Errorf("panic: %v", r)
		switch r := r.(type) {
		case engine_errs.EngineError:
			engErrs = append(engErrs, r)
		case error:
			engErrs = append(engErrs, engine_errs.InternalError{Err: r})
		default:
			engErrs = append(engErrs, engine_errs.InternalError{Err: fmt.Errorf("panic: %v", r)})
		}
	}()
	defer setupProfiling()()

	// Set up analytics, and hook them up to the logs
	analyticsClient := analytics.NewClient()
	analyticsClient.AppendProperties(map[string]any{})
	z, err := setupLogger(analyticsClient)
	if err != nil {
		internalError(err)
		return
	}
	// nolint:errcheck
	defer z.Sync() // ignore errors from sync, it's always "ERROR: sync /dev/stderr: invalid argument"

	zap.ReplaceGlobals(z)

	err = em.AddEngine()
	if err != nil {
		internalError(err)
		return
	}

	context := &EngineContext{}

	if architectureEngineCfg.inputGraph != "" {
		var input FileFormat
		zap.S().Info("Loading input graph")
		inputF, err := os.Open(architectureEngineCfg.inputGraph)
		if err != nil {
			internalError(err)
			return
		}
		defer inputF.Close()
		err = yaml.NewDecoder(inputF).Decode(&input)
		if err != nil {
			internalError(fmt.Errorf("failed to decode input graph: %w", err))
			return
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
			internalError(fmt.Errorf("failed to load constraints: %w", err))
			return
		}
		context.Constraints = runConstraints
	}
	// len(engErrs) == 0 at this point so overwriting it is safe
	// All other assignments prior are via 'internalError' and return
	exitCode, engErrs = em.Run(context)
	if exitCode == 1 {
		return
	}

	var files []kio.File

	configErrors := new(bytes.Buffer)
	err = writeEngineErrsJson(engErrs, configErrors)
	if err != nil {
		internalError(fmt.Errorf("failed to write config errors: %w", err))
		return
	}
	files = append(files, &kio.RawFile{
		FPath:   "config_errors.json",
		Content: configErrors.Bytes(),
	})

	zap.S().Info("Engine finished running... Generating views")
	vizFiles, err := em.Engine.VisualizeViews(context.Solutions[0])
	if err != nil {
		internalError(fmt.Errorf("failed to generate views %w", err))
		return
	}
	files = append(files, vizFiles...)
	zap.S().Info("Generating resources.yaml")
	b, err := yaml.Marshal(construct.YamlGraph{Graph: context.Solutions[0].DataflowGraph()})
	if err != nil {
		internalError(fmt.Errorf("failed to marshal graph: %w", err))
		return
	}
	files = append(files,
		&kio.RawFile{
			FPath:   "resources.yaml",
			Content: b,
		},
	)

	err = kio.OutputTo(files, architectureEngineCfg.outputDir)
	if err != nil {
		internalError(fmt.Errorf("failed to write output files: %w", err))
		return
	}
	return
}

func (em *EngineMain) GetValidEdgeTargets(cmd *cobra.Command, args []string) error {
	defer setupProfiling()()

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
		return fmt.Errorf("failed to load constraints: %w", err)
	}
	context := &GetPossibleEdgesContext{
		InputGraph:                inputF,
		GetValidEdgeTargetsConfig: config,
	}

	zap.S().Info("getting valid edge targets")
	validTargets, err := em.Engine.GetValidEdgeTargets(context)
	if err != nil {
		return fmt.Errorf("failed to run engine: %w", err)
	}

	zap.S().Info("writing output files")
	b, err := yaml.Marshal(validTargets)
	if err != nil {
		return fmt.Errorf("failed to marshal possible edges: %w", err)
	}
	var files []kio.File
	files = append(files, &kio.RawFile{
		FPath:   "valid_edge_targets.yaml",
		Content: b,
	})

	err = kio.OutputTo(files, getValidEdgeTargetsCfg.outputDir)
	if err != nil {
		return fmt.Errorf("failed to write output files: %w", err)
	}
	return nil
}

func writeDebugGraphs(sol solution_context.SolutionContext) {
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		err := GraphToSVG(sol.KnowledgeBase(), sol.DataflowGraph(), "dataflow")
		if err != nil {
			zap.S().Errorf("failed to write dataflow graph: %w", err)
		}
	}()
	go func() {
		defer wg.Done()
		err := GraphToSVG(sol.KnowledgeBase(), sol.DeploymentGraph(), "iac")
		if err != nil {
			zap.S().Errorf("failed to write iac graph: %w", err)
		}
	}()
	wg.Wait()
}
