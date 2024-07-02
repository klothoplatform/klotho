package orchestration

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/klothoplatform/klotho/pkg/k2/constructs/graph"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/klothoplatform/klotho/pkg/k2/stack"
)

// Orchestrator is the base orchestrator for the K2 platform
type Orchestrator struct {
	StateManager    *model.StateManager
	OutputDirectory string

	mu             sync.Mutex // guards the following fields
	infraGenerator *InfraGenerator
}

func NewOrchestrator(sm *model.StateManager, outputPath string) *Orchestrator {
	return &Orchestrator{
		StateManager:    sm,
		OutputDirectory: outputPath,
	}
}

func (o *Orchestrator) InfraGenerator() (*InfraGenerator, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.infraGenerator == nil {
		var err error
		o.infraGenerator, err = NewInfraGenerator()
		if err != nil {
			return nil, err
		}
	}
	return o.infraGenerator, nil
}

func (uo *UpOrchestrator) EvaluateConstruct(ctx context.Context, state model.State, constructUrn model.URN) (stack.Reference, error) {
	constructOutDir := filepath.Join(uo.OutputDirectory, constructUrn.ResourceID)
	cs, err := uo.ConstructEvaluator.Evaluate(constructUrn, state, ctx)
	if err != nil {
		return stack.Reference{}, err
	}

	ig, err := uo.InfraGenerator()
	if err != nil {
		return stack.Reference{}, fmt.Errorf("error getting infra generator: %w", err)
	}

	err = ig.Run(ctx, cs, constructOutDir)
	if err != nil {
		return stack.Reference{}, fmt.Errorf("error running infra generator: %w", err)
	}

	return stack.Reference{
		ConstructURN: constructUrn,
		Name:         constructUrn.ResourceID,
		IacDirectory: constructOutDir,
		AwsRegion:    uo.StateManager.GetState().DefaultRegion,
	}, nil
}

func (o *Orchestrator) resolveInitialState(ir *model.ApplicationEnvironment) (map[model.URN]model.ConstructAction, error) {
	actions := make(map[model.URN]model.ConstructAction)
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
		var action model.ConstructAction

		construct, exists := o.StateManager.GetConstructState(c.URN.ResourceID)
		if !exists {
			// If the construct doesn't exist in the current state, it's a create action
			action = model.ConstructActionCreate
			status = model.ConstructCreating
			construct = model.ConstructState{
				Status:      model.ConstructCreating,
				LastUpdated: time.Now().Format(time.RFC3339),
				Inputs:      c.Inputs,
				Outputs:     c.Outputs,
				Bindings:    c.Bindings,
				Options:     c.Options,
				DependsOn:   c.DependsOn,
				URN:         c.URN,
			}
		} else {
			if model.IsCreatable(construct.Status) {
				action = model.ConstructActionCreate
				status = model.ConstructCreating
			} else if model.IsUpdatable(construct.Status) {
				action = model.ConstructActionUpdate
				status = model.ConstructUpdating
			}
			construct.Inputs = c.Inputs
			construct.Outputs = c.Outputs
			construct.Bindings = c.Bindings
			construct.Options = c.Options
			construct.DependsOn = c.DependsOn
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
			if v.Status == model.ConstructDeleteComplete {
				continue
			}
			actions[*v.URN] = model.ConstructActionDelete
			if !model.IsDeletable(v.Status) {
				return nil, fmt.Errorf("construct %s is not deletable", v.URN.ResourceID)
			}
			err := o.StateManager.TransitionConstructState(&v, model.ConstructDeleting)
			if err != nil {
				return nil, err
			}
		}
	}

	return actions, nil
}

// sortConstructsByDependency sorts the constructs based on their dependencies and returns the deployment order
// in the form of sequential construct groups that can be deployed in parallel
func sortConstructsByDependency(constructs []model.ConstructState, actions map[model.URN]model.ConstructAction) ([][]model.URN, error) {
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
