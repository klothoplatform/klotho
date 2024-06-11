package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"sync"

	"github.com/iancoleman/strcase"
	clicommon "github.com/klothoplatform/klotho/pkg/cli_common"
	construct "github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	engine_errs "github.com/klothoplatform/klotho/pkg/engine/errors"
	"github.com/klothoplatform/klotho/pkg/engine/solution_context"
	kio "github.com/klothoplatform/klotho/pkg/io"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledgebase"
	"github.com/klothoplatform/klotho/pkg/knowledgebase/reader"
	"github.com/klothoplatform/klotho/pkg/provider/aws"
	"github.com/klothoplatform/klotho/pkg/templates"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type (
	EngineMain struct {
		Engine *Engine
	}
)

var commonCfg clicommon.CommonConfig

var engineCfg struct {
	provider   string
	guardrails string
}

var architectureEngineCfg struct {
	provider    string
	guardrails  string
	inputGraph  string
	constraints string
	outputDir   string
	globalTag   string
}

var getValidEdgeTargetsCfg struct {
	guardrails string
	inputGraph string
	configFile string
	outputDir  string
}

func (em *EngineMain) AddEngineCli(root *cobra.Command) {
	clicommon.SetupRoot(root, &commonCfg)

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
	flags.StringVarP(&architectureEngineCfg.globalTag, "global-tag", "t", "", "Global tag")

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

func extractEngineErrors(err error) []engine_errs.EngineError {
	if err == nil {
		return nil
	}
	var errs []engine_errs.EngineError
	queue := []error{err}
	for len(queue) > 0 {
		err := queue[0]
		queue = queue[1:]
		switch err := err.(type) {
		case engine_errs.EngineError:
			errs = append(errs, err)
		case interface{ Unwrap() []error }:
			queue = append(queue, err.Unwrap()...)
		case interface{ Unwrap() error }:
			queue = append(queue, err.Unwrap())
		}
	}
	if len(errs) == 0 {
		errs = append(errs, engine_errs.InternalError{Err: err})
	}
	return errs
}

func (em *EngineMain) Run(context *EngineContext) (int, []engine_errs.EngineError) {
	returnCode := 0
	var engErrs []engine_errs.EngineError

	log := zap.S().Named("engine")

	log.Info("Running engine")
	err := em.Engine.Run(context)
	if err != nil {
		// When the engine returns an error, that indicates that it halted evaluation, thus is a fatal error.
		// This is returned as exit code 1, and add the details to be printed to stdout.
		returnCode = 1
		engErrs = append(engErrs, extractEngineErrors(err)...)
		log.Errorf("Engine returned error: %v", err)
	}

	if len(context.Solutions) > 0 {
		writeDebugGraphs(context.Solutions[0])

		// If there are any decisions that are engine errors, add them to the list of error details
		// to be printed to stdout. These are returned as exit code 2 unless it is already code 1.
		for _, d := range context.Solutions[0].GetDecisions().GetRecords() {
			d, ok := d.(solution_context.AsEngineError)
			if !ok {
				continue
			}
			ee := d.TryEngineError()
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
	log := zap.S().Named("engine")

	defer func() { // defer functions execute in FILO order, so this executes after the 'recover'.
		err := writeEngineErrsJson(engErrs, os.Stdout)
		if err != nil {
			log.Errorf("failed to output errors to stdout: %v", err)
		}
	}()
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		log.Errorf("panic: %v", r)
		switch r := r.(type) {
		case engine_errs.EngineError:
			engErrs = append(engErrs, r)
		case error:
			engErrs = append(engErrs, engine_errs.InternalError{Err: r})
		default:
			engErrs = append(engErrs, engine_errs.InternalError{Err: fmt.Errorf("panic: %v", r)})
		}
	}()

	err := em.AddEngine()
	if err != nil {
		internalError(err)
		return
	}

	context := &EngineContext{
		GlobalTag: architectureEngineCfg.globalTag,
	}

	if architectureEngineCfg.inputGraph != "" {
		var input FileFormat
		log.Info("Loading input graph")
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
	log.Info("Loading constraints")

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

	log.Info("Engine finished running... Generating views")
	vizFiles, err := em.Engine.VisualizeViews(context.Solutions[0])
	if err != nil {
		internalError(fmt.Errorf("failed to generate views %w", err))
		return
	}
	files = append(files, vizFiles...)
	log.Info("Generating resources.yaml")
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

	if architectureEngineCfg.provider == "aws" {
		polictBytes, err := aws.DeploymentPermissionsPolicy(context.Solutions[0])
		if err != nil {
			internalError(fmt.Errorf("failed to generate deployment permissions policy: %w", err))
			return
		}
		files = append(files,
			&kio.RawFile{
				FPath:   "deployment_permissions_policy.json",
				Content: polictBytes,
			},
		)
	}

	err = kio.OutputTo(files, architectureEngineCfg.outputDir)
	if err != nil {
		internalError(fmt.Errorf("failed to write output files: %w", err))
		return
	}
	return
}

func (em *EngineMain) GetValidEdgeTargets(cmd *cobra.Command, args []string) error {
	log := zap.S().Named("engine")

	err := em.AddEngine()
	if err != nil {
		return err
	}
	log.Info("loading config")

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

	log.Info("getting valid edge targets")
	validTargets, err := em.Engine.GetValidEdgeTargets(context)
	if err != nil {
		return fmt.Errorf("failed to run engine: %w", err)
	}

	log.Info("writing output files")
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
			zap.S().Named("engine").Errorf("failed to write dataflow graph: %w", err)
		}
	}()
	go func() {
		defer wg.Done()
		err := GraphToSVG(sol.KnowledgeBase(), sol.DeploymentGraph(), "iac")
		if err != nil {
			zap.S().Named("engine").Errorf("failed to write iac graph: %w", err)
		}
	}()
	wg.Wait()
}
