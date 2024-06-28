package orchestration

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/klothoplatform/klotho/pkg/k2/stack"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"github.com/klothoplatform/klotho/pkg/tui"
	"go.uber.org/zap"
)

type (
	DownOrchestrator struct {
		*Orchestrator
	}

	DownRequest struct {
		StackReferences []stack.Reference
		DryRun          bool
	}
)

func NewDownOrchestrator(sm *model.StateManager, outputPath string) *DownOrchestrator {
	return &DownOrchestrator{
		Orchestrator: NewOrchestrator(sm, outputPath),
	}
}

func (do *DownOrchestrator) RunDownCommand(ctx context.Context, request DownRequest) error {
	if request.DryRun {
		// TODO Stack.Destroy hard-codes the flag to "--skip-preview"
		// and doesn't have any options for "--preview-only"
		// which was added in https://github.com/pulumi/pulumi/pull/15336
		return errors.New("Dryrun not supported in Down Command yet")
	}

	sm := do.StateManager
	defer func() {
		err := sm.SaveState()
		if err != nil {
			zap.S().Errorf("Error saving state: %v", err)
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

	var wg sync.WaitGroup
	var allErrors multierr.Error
	for _, group := range deleteOrder {
		for _, cURN := range group {
			wg.Add(1)
			go func(cURN model.URN) {
				defer wg.Done()
				construct, exists := sm.GetConstructState(cURN.ResourceID)
				if !exists {
					allErrors.Append(fmt.Errorf("construct %s not found in state", cURN.ResourceID))
					return
				}
				ctx := ConstructContext(ctx, *construct.URN)
				prog := tui.GetProgress(ctx)

				// All resources need to be deleted so they have to start in a delete pending state initially.
				// This is a bit awkward since we have to transition twice, but these states are used at different
				// times for things like the up command
				if err := sm.TransitionConstructState(&construct, model.ConstructDeletePending); err != nil {
					prog.Complete("Failed")
					allErrors.Append(err)
					return
				}
				if err := sm.TransitionConstructState(&construct, model.ConstructDeleting); err != nil {
					prog.Complete("Failed")
					allErrors.Append(err)
					return
				}

				stackRef := stackRefCache[construct.URN.ResourceID]

				err := stack.RunDown(ctx, stackRef)
				if err != nil {
					prog.Complete("Failed")

					if err2 := sm.TransitionConstructState(&construct, model.ConstructDeleteFailed); err2 != nil {
						allErrors.Append(fmt.Errorf("%v: error transitioning construct state to delete failed: %v", err, err2))
						return
					}
					allErrors.Append(err)
					return
				} else if err := sm.TransitionConstructState(&construct, model.ConstructDeleteComplete); err != nil {
					prog.Complete("Failed")
					allErrors.Append(err)
					return
				}
				prog.Complete("Success")
			}(cURN)
		}
		wg.Wait()
		if allErrors.ErrOrNil() != nil {
			return allErrors.ErrOrNil()
		}
	}
	return allErrors.ErrOrNil()
}
