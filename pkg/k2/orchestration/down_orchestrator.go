package orchestration

import (
	"context"
	"errors"
	"fmt"

	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/klothoplatform/klotho/pkg/k2/stack"
	"github.com/klothoplatform/klotho/pkg/logging"
	"github.com/klothoplatform/klotho/pkg/tui"
	"github.com/spf13/afero"
)

type (
	DownOrchestrator struct {
		*Orchestrator
		FS afero.Fs
	}

	DownRequest struct {
		StackReferences []stack.Reference
		DryRun          bool
	}
)

func NewDownOrchestrator(sm *model.StateManager, outputPath string) *DownOrchestrator {
	return &DownOrchestrator{
		Orchestrator: NewOrchestrator(sm, outputPath),
		FS:           afero.NewOsFs(),
	}
}

func (do *DownOrchestrator) RunDownCommand(ctx context.Context, request DownRequest, maxConcurrency int) error {
	log := logging.GetLogger(ctx).Sugar()
	if request.DryRun {
		// TODO Stack.Destroy hard-codes the flag to "--skip-preview"
		// and doesn't have any options for "--preview-only"
		// which was added in https://github.com/pulumi/pulumi/pull/15336
		return errors.New("Dryrun not supported in Down Command yet")
	}

	sm := do.StateManager
	defer func() {
		// update constructs that are still operating to failed
		for _, c := range sm.GetState().Constructs {
			if sm.IsOperating(&c) {
				if err := sm.TransitionConstructFailed(&c); err != nil {
					log.Errorf("Error transitioning construct state: %v", err)
				}
			}
		}
		if err := sm.SaveState(); err != nil {
			log.Errorf("Error saving state: %v", err)
		}
	}()

	stackRefCache := make(map[string]stack.Reference)

	actions := make(map[model.URN]model.ConstructAction)
	var constructsToDelete []model.ConstructState
	for _, ref := range request.StackReferences {
		c, exists := sm.GetConstructState(ref.ConstructURN.ResourceID)
		if !exists {
			// This means there's a construct in our StackReferences that doesn't exist in the state
			// This should never happen as we just build StackReferences from the state
			return fmt.Errorf("construct %s not found in state", ref.ConstructURN.ResourceID)
		}
		if c.Status == model.ConstructDeleteComplete {
			continue
		}
		constructsToDelete = append(constructsToDelete, c)

		// Cache the stack reference for later use outside this loop
		stackRefCache[ref.ConstructURN.ResourceID] = ref

		actions[*c.URN] = model.ConstructActionDelete
	}

	deleteOrder, err := sortConstructsByDependency(constructsToDelete, actions)
	if err != nil {
		return fmt.Errorf("failed to determine deployment order: %w", err)
	}

	for _, group := range deleteOrder {
		for _, cURN := range group {
			action := actions[cURN]
			ctx := ConstructContext(ctx, cURN)
			prog := tui.GetProgress(ctx)
			prog.UpdateIndeterminate(fmt.Sprintf("Starting %s", action))
		}
	}

	sem := make(chan struct{}, maxConcurrency)
	for _, group := range deleteOrder {
		errChan := make(chan error, len(group))

		for _, cURN := range group {
			sem <- struct{}{}
			go func(cURN model.URN) {
				defer func() { <-sem }()
				construct, exists := sm.GetConstructState(cURN.ResourceID)
				if !exists {
					errChan <- fmt.Errorf("construct %s not found in state", cURN.ResourceID)
					return
				}
				ctx := ConstructContext(ctx, *construct.URN)
				prog := tui.GetProgress(ctx)

				if construct.Status == model.ConstructDeleteComplete || construct.Status == model.ConstructCreating {
					prog.Complete("Skipped")
					errChan <- sm.TransitionConstructState(&construct, model.ConstructDeleteComplete)
					return
				}

				if err := sm.TransitionConstructState(&construct, model.ConstructDeleting); err != nil {
					prog.Complete("Failed")
					errChan <- err
					return
				}

				stackRef := stackRefCache[construct.URN.ResourceID]

				err := stack.RunDown(ctx, do.FS, stackRef)
				if err != nil {
					prog.Complete("Failed")

					if err2 := sm.TransitionConstructFailed(&construct); err2 != nil {
						err = fmt.Errorf("%v: error transitioning construct state to delete failed: %v", err, err2)
					}
					errChan <- err
					return
				} else if err := sm.TransitionConstructComplete(&construct); err != nil {
					prog.Complete("Failed")
					errChan <- err
					return
				}
				prog.Complete("Success")
				errChan <- nil
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
