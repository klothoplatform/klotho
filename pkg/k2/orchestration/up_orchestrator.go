package orchestration

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/klothoplatform/klotho/pkg/engine/debug"
	"github.com/klothoplatform/klotho/pkg/k2/constructs"
	pb "github.com/klothoplatform/klotho/pkg/k2/language_host/go"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/klothoplatform/klotho/pkg/k2/stack"
	"github.com/klothoplatform/klotho/pkg/logging"
	"github.com/klothoplatform/klotho/pkg/tui"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

type UpOrchestrator struct {
	*Orchestrator
	LanguageHostClient pb.KlothoServiceClient
	StackStateManager  *stack.StateManager
	ConstructEvaluator *constructs.ConstructEvaluator
	FS                 afero.Fs
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
		FS:                 afero.NewOsFs(),
	}, nil
}

func (uo *UpOrchestrator) RunUpCommand(ctx context.Context, ir *model.ApplicationEnvironment, dryRun bool, maxConcurrency int) error {
	uo.ConstructEvaluator.DryRun = dryRun
	defer uo.FinalizeState(ctx)

	actions, err := uo.resolveInitialState(ir)
	if err != nil {
		return fmt.Errorf("error resolving initial state: %w", err)
	}

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

	sem := make(chan struct{}, maxConcurrency)
	for _, group := range deployOrder {
		errChan := make(chan error, len(group))

		for _, cURN := range group {
			sem <- struct{}{}
			go func(cURN model.URN) {
				defer func() { <-sem }()
				c, exists := uo.StateManager.GetConstructState(cURN.ResourceID)
				if !exists {
					errChan <- fmt.Errorf("construct %s not found in state", cURN.ResourceID)
					return
				}
				errChan <- uo.executeAction(ctx, c, actions[cURN], dryRun)
			}(cURN)
		}
		var errs []error
		for i := 0; i < len(group); i++ {
			if err := <-errChan; err != nil {
				errs = append(errs, err)
			}
		}

		if len(errs) > 0 {
			return errors.Join(errs...)
		}
	}
	return nil
}

func (uo *UpOrchestrator) executeAction(ctx context.Context, c model.ConstructState, action model.ConstructAction, dryRun bool) (err error) {
	sm := uo.StateManager
	log := logging.GetLogger(ctx).Sugar()
	outDir := filepath.Join(uo.OutputDirectory, c.URN.ResourceID)

	ctx = ConstructContext(ctx, *c.URN)
	ctx = debug.WithDebugDir(ctx, outDir)
	prog := tui.GetProgress(ctx)
	prog.UpdateIndeterminate(fmt.Sprintf("Starting %s", action))

	skipped := false

	defer func() {
		r := recover()
		msg := "Success"
		if err != nil || r != nil {
			msg = "Failed"
		} else if dryRun {
			msg += " (dry run)"
		}
		if skipped && err == nil {
			msg = "Skipped"
		}

		prog.Complete(msg)
		if r != nil {
			panic(r)
		}
	}()

	if action == model.ConstructActionDelete {
		if !model.IsDeletable(c.Status) {
			skipped = true
			log.Debugf("Skipping construct %s, status is %s", c.URN.ResourceID, c.Status)
			return nil
		}

		if dryRun {
			log.Infof("Dry run: Skipping pulumi down for deleted construct %s", c.URN.ResourceID)
			return nil
		}

		// Mark as deleting
		if err = sm.TransitionConstructState(&c, model.ConstructDeleting); err != nil {
			return err
		}

		err = stack.RunDown(ctx, uo.FS, stack.Reference{
			ConstructURN: *c.URN,
			Name:         c.URN.ResourceID,
			IacDirectory: outDir,
			AwsRegion:    sm.GetState().DefaultRegion,
		})

		if err != nil {
			if err2 := sm.TransitionConstructFailed(&c); err2 != nil {
				log.Errorf("Error transitioning construct state: %v", err2)
			}
			return fmt.Errorf("error running pulumi down command: %w", err)
		}

		// Mark as deleted
		return sm.TransitionConstructComplete(&c)
	}

	// Only proceed if the construct is deployable
	if !model.IsDeployable(c.Status) {
		skipped = true
		log.Debugf("Skipping construct %s, status is %s", c.URN.ResourceID, c.Status)
		return nil
	}

	// Evaluate the construct
	stackRef, err := uo.EvaluateConstruct(ctx, *uo.StateManager.GetState(), *c.URN)
	if err != nil {
		return fmt.Errorf("error evaluating construct: %w", err)
	}

	if dryRun {
		_, err = stack.RunPreview(ctx, uo.FS, stackRef)
		return err
	}

	// Run pulumi up command for the construct
	upResult, stackState, err := stack.RunUp(ctx, uo.FS, stackRef)
	if err != nil {
		if err2 := sm.TransitionConstructFailed(&c); err2 != nil {
			log.Errorf("Error transitioning construct state: %v", err2)
		}
		return fmt.Errorf("error running pulumi up command: %w", err)
	}
	uo.StackStateManager.ConstructStackState[stackRef.ConstructURN] = *stackState

	err = sm.RegisterOutputValues(stackRef.ConstructURN, stackState.Outputs)
	if err != nil {
		return fmt.Errorf("error registering output values: %w", err)
	}

	// Update construct state based on the up result
	err = stack.UpdateConstructStateFromUpResult(sm, stackRef, upResult)
	if err != nil {
		return err
	}

	// Resolve pending output values by calling the language host
	resolvedOutputs, err := uo.resolveOutputValues(stackRef, *stackState)
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
