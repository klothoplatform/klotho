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
	"time"
)

// Orchestrator is the main orchestrator for the K2 platform

type (
	Orchestrator struct {
		StateManager       *model.StateManager
		LanguageHostClient pb.KlothoServiceClient
		OutputDirectory    string
	}
)

func NewOrchestrator(sm *model.StateManager, languageHostClient pb.KlothoServiceClient, outputPath string) *Orchestrator {
	return &Orchestrator{
		StateManager:       sm,
		LanguageHostClient: languageHostClient,
		OutputDirectory:    outputPath,
	}
}

func (o *Orchestrator) RunUpCommand(ir *model.ApplicationEnvironment, dryRun bool) error {
	actions, err := o.resolveInitialState(ir)
	if err != nil {
		return errors2.WrapErrf(err, "error resolving initial state")
	}
	zap.S().Infof("Pending Actions:")
	for k, v := range actions {
		zap.S().Infof("%s: %s", k.String(), v)
	}

	var cs []model.ConstructState
	constructState := o.StateManager.GetState().Constructs
	for cURN := range actions {
		cs = append(cs, constructState[cURN.ResourceID])
	}

	deployOrder, err := sortConstructsByDependency(cs, actions)
	if err != nil {
		return errors2.WrapErrf(err, "failed to determine deployment order")
	}

	deployer := deployment.Deployer{
		StateManager: o.StateManager, LanguageHostClient: o.LanguageHostClient}

	sm := o.StateManager
	defer func() {
		err = sm.SaveState()
	}()

	for _, group := range deployOrder {
		for _, cURN := range group {
			if err != nil {
				return err
			}

			c := o.StateManager.GetState().Constructs[cURN.ResourceID]

			// Run pulumi down command for deleted constructs
			if actions[*c.URN] == model.ConstructActionDelete {
				err = pulumi.RunStackDown(pulumi.StackReference{
					ConstructURN: *c.URN,
					Name:         c.URN.ResourceID,
					IacDirectory: filepath.Join(o.OutputDirectory, c.URN.ResourceID),
					AwsRegion:    sm.GetState().DefaultRegion,
				}, dryRun)

				if err != nil {
					return errors2.WrapErrf(err, "error running pulumi down command")
				}
				// DS: should this be removed from the state or marked as destroyed?
				delete(sm.GetState().Constructs, cURN.ResourceID)
				continue
			}

			// Evaluate constructs and run pulumi up for create and update actions

			// Evaluate the construct
			stackRef, err := o.EvaluateConstruct(*o.StateManager.GetState(), *c.URN)
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
			sm.UpdateResourceState(c.URN.ResourceID, model.Updated, time.Now().String())
		}
	}
	return err
}

func (o *Orchestrator) RunDownCommand(request deployment.DownRequest) error {
	deployer := deployment.Deployer{}
	err := deployer.RunApplicationDownCommand(request)
	return err
}

func (o *Orchestrator) EvaluateConstruct(state model.State, constructUrn model.URN) (pulumi.StackReference, error) {
	constructOutDir := filepath.Join(o.OutputDirectory, constructUrn.ResourceID)
	c := state.Constructs[constructUrn.ResourceID]
	inputs := make(map[string]any)
	var merr multierr.Error
	for k, v := range c.Inputs {
		if v.Status != "" && v.Status != model.InputStatusResolved {
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
		AwsRegion:    state.DefaultRegion,
	}, nil
}

// sortConstructsByDependency sorts the constructs based on their dependencies and returns the deployment order
// in the form of sequential construct groups that can be deployed in parallel
func sortConstructsByDependency(constructs []model.ConstructState, actions map[model.URN]model.ConstructActionType) ([][]model.URN, error) {
	constructGraph := graph.NewAcyclicGraph()

	// Add vertices and edges to the graph based on the construct dependencies.
	// Edges are reversed for delete actions
	// (i.e., if 'a' depends on 'b', and 'a' is to be deleted, the edge is from 'b' to 'a' otherwise from 'a' to 'b')
	for _, c := range constructs {
		_ = constructGraph.AddVertex(*c.URN)
		for _, dep := range c.DependsOn {
			var source, target model.URN
			if actions[*c.URN] == model.ConstructActionDelete {
				source = *dep
				target = *c.URN
			} else {
				source = *c.URN
				target = *dep
			}
			_ = constructGraph.AddEdge(source, target)
		}
		for _, b := range c.Bindings {
			var source, target model.URN
			if actions[*c.URN] == model.ConstructActionDelete {
				source = *b.URN
				target = *c.URN
			} else {
				source = *c.URN
				target = *b.URN
			}
			_ = constructGraph.AddEdge(source, target)
		}
		for _, i := range c.Inputs {
			for _, dep := range i.DependsOn {
				var source, target model.URN
				if actions[*c.URN] == model.ConstructActionDelete {
					source = *dep
					target = *c.URN
				} else {
					source = *c.URN
					target = *dep
				}
				_ = constructGraph.AddEdge(source, target)
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

// resolveInitialState resolves the initial state of the constructs in the application environment
// and returns the actions to be taken
func (o *Orchestrator) resolveInitialState(ir *model.ApplicationEnvironment) (map[model.URN]model.ConstructActionType, error) {
	actions := make(map[model.URN]model.ConstructActionType)
	state := o.StateManager.GetState()

	//TODO: implement some kind of versioning check
	state.Version += 1

	if state.DefaultRegion != ir.DefaultRegion {
		return nil, fmt.Errorf("default region mismatch: %s != %s", state.DefaultRegion, ir.DefaultRegion)
	}

	if state.SchemaVersion != ir.SchemaVersion {
		return nil, fmt.Errorf("state schema version mismatch")
	}

	currentConstructs := o.StateManager.GetState().Constructs
	for _, c := range ir.Constructs {
		var status model.ConstructStatus
		if _, ok := currentConstructs[c.URN.ResourceID]; !ok {
			actions[*c.URN] = model.ConstructActionCreate
			status = model.New
		} else {
			actions[*c.URN] = model.ConstructActionUpdate
			status = model.UpdatePending
		}

		currentConstructs[c.URN.ResourceID] = model.ConstructState{
			Status:      status,
			LastUpdated: time.Now().String(),
			Inputs:      c.Inputs,
			Outputs:     c.Outputs,
			Bindings:    c.Bindings,
			Options:     c.Options,
			DependsOn:   c.DependsOn,
			PulumiStack: model.UUID{}, // TODO: set the pulumi stack identifier
			URN:         c.URN,
		}
	}

	// find deleted constructs
	for k, v := range currentConstructs {
		if _, ok := ir.Constructs[k]; !ok {
			actions[*v.URN] = model.ConstructActionDelete
			v.Status = model.DestroyPending
		}
	}

	return actions, nil
}
