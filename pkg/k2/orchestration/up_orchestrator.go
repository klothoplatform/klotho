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
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
	"gopkg.in/yaml.v3"
)

type UpOrchestrator struct {
	*Orchestrator
	LanguageHostClient pb.KlothoServiceClient
	StackStateManager  *stack.StateManager
	ConstructEvaluator *constructs.ConstructEvaluator
}

func NewUpOrchestrator(
	sm *model.StateManager, languageHostClient pb.KlothoServiceClient, fs afero.Fs, outputPath string,
) (*UpOrchestrator, error) {
	ssm := stack.NewStateManager()
	ce, err := constructs.NewConstructEvaluator(sm, ssm)
	if err != nil {
		return nil, err
	}
	return &UpOrchestrator{
		Orchestrator:       NewOrchestrator(sm, fs, outputPath),
		LanguageHostClient: languageHostClient,
		StackStateManager:  ssm,
		ConstructEvaluator: ce,
	}, nil
}

func (uo *UpOrchestrator) RunUpCommand(
	ctx context.Context, ir *model.ApplicationEnvironment, dryRun model.DryRun, sem *semaphore.Weighted,
) error {
	uo.ConstructEvaluator.DryRun = dryRun
	if dryRun == model.DryRunNone {
		// We don't finalize for dryrun as this updates/creates the state file
		defer uo.FinalizeState(ctx)
	}
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

	for _, group := range deployOrder {
		errChan := make(chan error, len(group))

		for _, cURN := range group {
			if err := sem.Acquire(ctx, 1); err != nil {
				errChan <- fmt.Errorf("error acquiring semaphore: %w", err)
				continue
			}
			go func(cURN model.URN) {
				defer sem.Release(1)
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

// placeholderOutputs sends placeholder values to TUI for cases where they cannot be taken from the state when
// running in dry run modes.
func (uo *UpOrchestrator) placeholderOutputs(ctx context.Context, cURN model.URN) {
	c, ok := uo.ConstructEvaluator.Constructs.Get(cURN)
	if !ok {
		return
	}
	prog := tui.GetProgram(ctx)
	if prog == nil {
		return
	}
	outputs := c.Outputs
	if len(outputs) == 0 && c.Solution != nil {
		outputs = make(map[string]any)
		for name, o := range c.Solution.Outputs() {
			if !o.Ref.IsZero() {
				outputs[name] = fmt.Sprintf("<%s>", o.Ref)
				continue
			}
			outputs[name] = o.Value
		}
	}
	for key, value := range outputs {
		prog.Send(tui.OutputMessage{
			Construct: cURN.ResourceID,
			Name:      key,
			Value:     value,
		})
	}
}

func (uo *UpOrchestrator) executeAction(ctx context.Context, c model.ConstructState, action model.ConstructAction, dryRun model.DryRun) (err error) {
	sm := uo.StateManager
	log := logging.GetLogger(ctx).Sugar()
	outDir := filepath.Join(uo.OutputDirectory, c.URN.ResourceID)

	ctx = ConstructContext(ctx, *c.URN)
	if debugDir := debug.GetDebugDir(ctx); debugDir != "" {
		ctx = debug.WithDebugDir(ctx, filepath.Join(debugDir, outDir))
	}
	prog := tui.GetProgress(ctx)
	prog.UpdateIndeterminate(fmt.Sprintf("Starting %s", action))

	skipped := false

	defer func() {
		r := recover()
		msg := "Success"
		if err != nil || r != nil {
			msg = "Failed"
		} else if dryRun > 0 {
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

		if dryRun > 0 {
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

	switch dryRun {
	case model.DryRunPreview:
		_, err = stack.RunPreview(ctx, uo.FS, stackRef)
		uo.placeholderOutputs(ctx, *c.URN)
		return err

	case model.DryRunCompile:
		err = stack.InstallDependencies(ctx, stackRef.IacDirectory)
		if err != nil {
			return err
		}

		cmd := logging.Command(ctx,
			logging.CommandLogger{
				RootLogger:  log.Desugar().Named("pulumi.tsc"),
				StdoutLevel: zap.DebugLevel,
				StderrLevel: zap.DebugLevel,
			},
			"tsc", "--noEmit", "index.ts",
		)
		cmd.Dir = stackRef.IacDirectory
		err := cmd.Run()
		uo.placeholderOutputs(ctx, *c.URN)
		if err != nil {
			return fmt.Errorf("error running tsc: %w", err)
		}
		return nil

	case model.DryRunFileOnly:
		// file already written, nothing left to do
		uo.placeholderOutputs(ctx, *c.URN)
		return nil
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

	err = sm.RegisterOutputValues(ctx, stackRef.ConstructURN, stackState.Outputs)
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
	return sm.RegisterOutputValues(ctx, stackRef.ConstructURN, resolvedOutputs)
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
