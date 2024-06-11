package orchestration

import (
	"context"
	"fmt"
	errors2 "github.com/klothoplatform/klotho/pkg/errors"
	"github.com/klothoplatform/klotho/pkg/k2/constructs"
	"github.com/klothoplatform/klotho/pkg/k2/constructs/graph"
	"github.com/klothoplatform/klotho/pkg/k2/deployment"
	pb "github.com/klothoplatform/klotho/pkg/k2/language_host/go"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/klothoplatform/klotho/pkg/k2/pulumi"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	"path/filepath"
)

// Orchestrator is the main orchestrator for the K2 platform

type (
	Orchestrator struct {
		StateManager       *model.StateManager
		LanguageHostClient pb.KlothoServiceClient
		OutputDirectory    string
		AwsRegion          string
	}
)

func NewOrchestrator(sm *model.StateManager, languageHostClient pb.KlothoServiceClient, outputPath string, awsRegion string) *Orchestrator {
	return &Orchestrator{
		StateManager:       sm,
		LanguageHostClient: languageHostClient,
		OutputDirectory:    outputPath,
		AwsRegion:          awsRegion,
	}
}

func (o *Orchestrator) RunUpCommand(ir *model.ApplicationEnvironment, dryRun bool) error {

	var cs []model.Construct
	for _, c := range ir.Constructs {
		cs = append(cs, c)
	}

	deployOrder, err := sortConstructsByDependency(cs)
	if err != nil {
		return errors2.WrapErrf(err, "failed to determine deployment order")
	}

	deployer := deployment.Deployer{
		StateManager: o.StateManager, LanguageHostClient: o.LanguageHostClient}

	sm := o.StateManager
	defer func() {
		err = sm.SaveState()
	}()

	//TODO: execute runStackDown for removed construct stack references when we have state management
	for _, group := range deployOrder {
		for _, cURN := range group {
			if err != nil {
				return err
			}

			c := ir.Constructs[cURN.ResourceID]

			// Evaluate the construct
			stackRef, err := o.EvaluateConstruct(*o.StateManager.GetState(), c)
			if err != nil {
				return errors2.WrapErrf(err, "error evaluating construct")
			}

			// Run pulumi up command for the construct
			stackState, err := deployer.RunStackUpCommand(stackRef, dryRun)
			if err != nil {
				return err
			}

			// Resolve output values
			err2 := o.resolveOutputValues(stackRef, stackState)
			if err2 != nil {
				return err2
			}
		}
	}
	return err
}

func (o *Orchestrator) RunDownCommand(request deployment.DownRequest) error {
	deployer := deployment.Deployer{}
	err := deployer.RunApplicationDownCommand(request)
	return err
}

func (o *Orchestrator) EvaluateConstruct(state model.State, c model.Construct) (pulumi.StackReference, error) {
	constructOutDir := filepath.Join(o.OutputDirectory, c.URN.ResourceID)
	inputs := make(map[string]any)
	var merr multierr.Error
	for k, v := range c.Inputs {
		if v.Status != "" && v.Status != model.Resolved {
			merr.Append(fmt.Errorf("input '%s' is not resolved", k))
			continue
		}
		inputs[k] = v.Value
	}
	if len(merr) > 0 {
		return pulumi.StackReference{}, merr.ErrOrNil()
	}

	urn := *c.URN
	constructEvaluator, err := constructs.NewConstructEvaluator(urn, inputs, state)
	if err != nil {
		return pulumi.StackReference{}, errors2.WrapErrf(err, "error creating construct evaluator")
	}
	_, cs, err := constructEvaluator.Evaluate()
	if err != nil {
		return pulumi.StackReference{}, errors2.WrapErrf(err, "error evaluating construct")
	}

	ig := &InfraGenerator{}
	err = ig.Run(cs, constructOutDir)
	if err != nil {
		return pulumi.StackReference{}, errors2.WrapErrf(err, "error running infra generator")
	}

	return pulumi.StackReference{
		ConstructURN: urn,
		Name:         urn.ResourceID,
		IacDirectory: constructOutDir,
		AwsRegion:    o.AwsRegion,
	}, nil
}

func sortConstructsByDependency(constructs []model.Construct) ([][]model.URN, error) {
	constructGraph := graph.NewAcyclicGraph()

	for _, c := range constructs {
		_ = constructGraph.AddVertex(*c.URN)
		for _, dep := range c.DependsOn {
			_ = constructGraph.AddEdge(*c.URN, *dep)
		}
		for _, b := range c.Bindings {
			_ = constructGraph.AddEdge(*c.URN, *b.URN)
		}
		for _, i := range c.Inputs {
			for _, dep := range i.DependsOn {
				_ = constructGraph.AddEdge(*c.URN, *dep)
			}
		}
	}
	return graph.ResolveDeploymentGroups(constructGraph)
}

func (o *Orchestrator) resolveOutputValues(stackReference pulumi.StackReference, stackState pulumi.StackState) error {
	// TODO: This is a demo implementation that passes the stack outputs to the language host
	//       and gets the resolved output references back.
	//       It doesn't actually do anything with the resolved outputs yet.
	outputs := map[string]map[string]interface{}{
		stackReference.ConstructURN.String(): stackState.Outputs,
	}
	payload, err := yaml.Marshal(outputs)
	if err != nil {
		return err
	}
	resp, err := o.LanguageHostClient.RegisterConstruct(context.Background(), &pb.RegisterConstructRequest{
		YamlPayload: string(payload),
	})
	zap.S().Info(resp.GetMessage())
	var resolvedOutputs []any
	for _, o := range resp.GetResolvedOutputs() {
		if err != nil {
			return err
		}
		resolvedOutputs = append(resolvedOutputs, map[string]interface{}{
			"id":    o.GetId(),
			"value": o.GetYamlPayload(),
		})
	}
	zap.S().Infof("Resolved Outputs: %v", resolvedOutputs)
	if err != nil {
		return err
	}
	return nil
}
