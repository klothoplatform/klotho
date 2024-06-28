package orchestration

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/klothoplatform/klotho/pkg/engine/debug"
	"github.com/klothoplatform/klotho/pkg/k2/constructs"
	pb "github.com/klothoplatform/klotho/pkg/k2/language_host/go"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/klothoplatform/klotho/pkg/k2/stack"
	"github.com/klothoplatform/klotho/pkg/logging"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"github.com/klothoplatform/klotho/pkg/tui"
	"gopkg.in/yaml.v3"
)

type UpOrchestrator struct {
	*Orchestrator
	LanguageHostClient pb.KlothoServiceClient
	StackStateManager  *stack.StateManager
	ConstructEvaluator *constructs.ConstructEvaluator
}

func NewUpOrchestrator(sm *model.StateManager, languageHostClient pb.KlothoServiceClient, outputPath string) (*UpOrchestrator, error) {
	ssm := stack.NewStateManager()
	ce, err := constructs.NewConstructEvaluator(sm, ssm)
	if err != nil {
		return nil, err
	}
	return &UpOrchestrator{
		Orchestrator:       NewOrchestrator(sm, outputPath),
		LanguageHostClient: languageHostClient,
		StackStateManager:  ssm,
		ConstructEvaluator: ce,
	}, nil
}

func transitionPendingToDoing(sm *model.StateManager, construct *model.ConstructState) error {
	var nextStatus model.ConstructStatus

	switch construct.Status {
	case model.ConstructCreatePending:
		nextStatus = model.ConstructCreating
	case model.ConstructUpdatePending:
		nextStatus = model.ConstructUpdating
	case model.ConstructDeletePending:
		nextStatus = model.ConstructDeleting
	default:
		return fmt.Errorf("construct %s is not in a pending state", construct.URN.ResourceID)
	}

	return sm.TransitionConstructState(construct, nextStatus)
}

func (uo *UpOrchestrator) RunUpCommand(ctx context.Context, ir *model.ApplicationEnvironment, dryRun bool) error {
	actions, err := uo.resolveInitialState(ir)
	if err != nil {
		return fmt.Errorf("error resolving initial state: %w", err)
	}
	log := logging.GetLogger(ctx).Sugar()

	var cs []model.ConstructState
	constructState := uo.StateManager.GetState().Constructs
	for cURN := range actions {
		cs = append(cs, constructState[cURN.ResourceID])
	}

	deployOrder, err := sortConstructsByDependency(cs, actions)
	if err != nil {
		return fmt.Errorf("failed to determine deployment order: %w", err)
	}

	for _, group := range deployOrder {
		for _, cURN := range group {
			action := actions[cURN]
			ctx := ConstructContext(ctx, cURN)
			prog := tui.GetProgress(ctx)
			prog.UpdateIndeterminate(fmt.Sprintf("Pending %s", action))
		}
	}

	sm := uo.StateManager
	defer func() {
		err = sm.SaveState()
		if err != nil {
			log.Errorf("Error saving state: %v", err)
		}
	}()

	var wg sync.WaitGroup
	var allErrors multierr.Error
	for _, group := range deployOrder {
		for _, cURN := range group {
			wg.Add(1)
			go func(cURN model.URN) {
				defer wg.Done()
				c, exists := uo.StateManager.GetConstructState(cURN.ResourceID)
				if !exists {
					allErrors.Append(fmt.Errorf("construct %s not found in state", cURN.ResourceID))
					return
				}
				if err := uo.executeAction(ctx, c, actions[cURN], dryRun); err != nil {
					allErrors.Append(err)
				}
			}(cURN)
		}
		wg.Wait()
		if allErrors.ErrOrNil() != nil {
			return allErrors.ErrOrNil()
		}
	}
	return allErrors.ErrOrNil()
}

func (uo *UpOrchestrator) executeAction(ctx context.Context, c model.ConstructState, action model.ConstructAction, dryRun bool) error {
	sm := uo.StateManager
	log := logging.GetLogger(ctx).Sugar()
	outDir := filepath.Join(uo.OutputDirectory, c.URN.ResourceID)

	ctx = ConstructContext(ctx, *c.URN)
	ctx = debug.WithDebugDir(ctx, outDir)
	prog := tui.GetProgress(ctx)
	prog.UpdateIndeterminate(fmt.Sprintf("Starting %s", action))

	var err error
	skipped := false

	defer func() {
		msg := "Success"
		if err != nil {
			msg = "Failed"
		}
		if skipped {
			msg = "Skipped"
		}
		prog.Complete(msg)
	}()

	// Run pulumi down command for deleted constructs
	if action == model.ConstructActionDelete && model.IsDeletable(c.Status) {
		if dryRun {
			log.Infof("Dry run: Skipping pulumi down for deleted construct %s", c.URN.ResourceID)
			return nil
		}

		// Mark as deleting
		if err = sm.TransitionConstructState(&c, model.ConstructDeleting); err != nil {
			return err
		}

		err = stack.RunDown(ctx, stack.Reference{
			ConstructURN: *c.URN,
			Name:         c.URN.ResourceID,
			IacDirectory: outDir,
			AwsRegion:    sm.GetState().DefaultRegion,
		})

		if err != nil {
			return fmt.Errorf("error running pulumi down command: %w", err)
		}

		// Mark as deleted
		return sm.TransitionConstructState(&c, model.ConstructDeleteComplete)
	}

	// Only proceed if the construct is deployable
	if !model.IsDeployable(c.Status) {
		skipped = true
		return nil
	}

	// Evaluate the construct
	stackRef, err := uo.EvaluateConstruct(ctx, *uo.StateManager.GetState(), *c.URN)
	if err != nil {
		return fmt.Errorf("error evaluating construct: %w", err)
	}

	if dryRun {
		_, err = stack.RunPreview(ctx, stackRef)
		return err
	}

	if err = transitionPendingToDoing(sm, &c); err != nil {
		return fmt.Errorf("error transitioning construct state: %w", err)
	}

	// Run pulumi up command for the construct
	upResult, stackState, err := stack.RunUp(ctx, stackRef)
	if err != nil {
		return fmt.Errorf("error running pulumi up command: %w", err)
	}
	uo.StackStateManager.ConstructStackState[stackRef.ConstructURN] = stackState

	err = sm.RegisterOutputValues(stackRef.ConstructURN, stackState.Outputs)
	if err != nil {
		return fmt.Errorf("error registering output values: %w", err)
	}

	// Update construct state based on the up result
	err = stack.UpdateConstructStateFromUpResult(sm, stackRef, &upResult)
	if err != nil {
		return err
	}

	// Resolve pending output values by calling the language host
	resolvedOutputs, err := uo.resolveOutputValues(stackRef, stackState)
	if err != nil {
		return fmt.Errorf("error resolving output values: %w", err)
	}
	uo.ConstructEvaluator.RegisterOutputValues(stackRef.ConstructURN, stackState.Outputs)
	return sm.RegisterOutputValues(stackRef.ConstructURN, resolvedOutputs)
}

func (uo *UpOrchestrator) resolveOutputValues(stackReference stack.Reference, stackState stack.State) (map[string]any, error) {
	outputs := map[string]map[string]any{
		stackReference.ConstructURN.String(): stackState.Outputs,
	}
	payload, err := yaml.Marshal(outputs)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*300)
	defer cancel()
	resp, err := uo.LanguageHostClient.RegisterConstruct(ctx, &pb.RegisterConstructRequest{
		YamlPayload: string(payload),
	})
	if err != nil {
		return nil, err
	}
	var resolvedOutputs map[string]any
	err = yaml.Unmarshal([]byte(resp.GetYamlPayload()), &resolvedOutputs)
	if err != nil {
		return nil, err
	}
	return resolvedOutputs, nil
}
