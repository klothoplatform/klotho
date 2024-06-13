package orchestration

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/klothoplatform/klotho/pkg/k2/constructs"
	"github.com/klothoplatform/klotho/pkg/k2/constructs/graph"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/klothoplatform/klotho/pkg/k2/pulumi"
	"github.com/klothoplatform/klotho/pkg/multierr"
)

// Orchestrator is the base orchestrator for the K2 platform
type Orchestrator struct {
	StateManager    *model.StateManager
	OutputDirectory string
}

func NewOrchestrator(sm *model.StateManager, outputPath string) *Orchestrator {
	return &Orchestrator{
		StateManager:    sm,
		OutputDirectory: outputPath,
	}
}

// Shared and helper functions
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
		return pulumi.StackReference{}, fmt.Errorf("error creating construct evaluator: %w", err)
	}
	_, cs, err := constructEvaluator.Evaluate()
	if err != nil {
		return pulumi.StackReference{}, fmt.Errorf("error evaluating construct: %w", err)
	}

	ig := &InfraGenerator{}
	err = ig.Run(cs, constructOutDir)
	if err != nil {
		return pulumi.StackReference{}, fmt.Errorf("error running infra generator: %w", err)
	}

	return pulumi.StackReference{
		ConstructURN: urn,
		Name:         urn.ResourceID,
		IacDirectory: constructOutDir,
		AwsRegion:    state.DefaultRegion,
	}, nil
}

func (o *Orchestrator) resolveInitialState(ir *model.ApplicationEnvironment) (map[model.URN]model.ConstructActionType, error) {
	actions := make(map[model.URN]model.ConstructActionType)
	state := o.StateManager.GetState()

	//TODO: implement some kind of versioning check
	state.Version += 1

	// Check for default region mismatch
	if state.DefaultRegion != ir.DefaultRegion {
		return nil, fmt.Errorf("default region mismatch: %s != %s", state.DefaultRegion, ir.DefaultRegion)
	}

	// Check for schema version mismatch
	if state.SchemaVersion != ir.SchemaVersion {
		return nil, fmt.Errorf("state schema version mismatch")
	}

	for _, c := range ir.Constructs {
		var status model.ConstructStatus
		var action model.ConstructActionType

		construct, exists := o.StateManager.GetConstruct(c.URN.ResourceID)
		if !exists {
			// If the construct doesn't exist in the current state, it's a create action
			action = model.ConstructActionCreate
			status = model.ConstructCreatePending
			construct = model.ConstructState{
				Status:      model.ConstructPending,
				LastUpdated: time.Now().Format(time.RFC3339),
				Inputs:      c.Inputs,
				Outputs:     c.Outputs,
				Bindings:    c.Bindings,
				Options:     c.Options,
				DependsOn:   c.DependsOn,
				URN:         c.URN,
			}
		} else {
			// If the construct exists, it's an update action
			action = model.ConstructActionUpdate
			status = model.ConstructUpdatePending
		}

		actions[*c.URN] = action
		err := o.StateManager.TransitionConstructState(&construct, status)
		if err != nil {
			return nil, err
		}
	}

	// Find deleted constructs
	for k, v := range o.StateManager.GetState().Constructs {
		if _, ok := ir.Constructs[k]; !ok {
			actions[*v.URN] = model.ConstructActionDelete
			err := o.StateManager.TransitionConstructState(&v, model.ConstructDeletePending)
			if err != nil {
				return nil, err
			}
		}
	}

	return actions, nil
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
	}
	for _, c := range constructs {
		for _, dep := range c.DependsOn {
			var source, target model.URN
			if actions[*c.URN] == model.ConstructActionDelete {
				source = *dep
				target = *c.URN
			} else {
				source = *c.URN
				target = *dep
			}
			err := constructGraph.AddEdge(source, target)
			if err != nil {
				return nil, err
			}
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
			err := constructGraph.AddEdge(source, target)
			if err != nil {
				return nil, err
			}
		}
	}
	return graph.ResolveDeploymentGroups(constructGraph)
}
