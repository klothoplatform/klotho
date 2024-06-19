package orchestration

import (
	"context"
	"errors"
	"fmt"
	"github.com/klothoplatform/klotho/pkg/k2/stack"

	"github.com/klothoplatform/klotho/pkg/k2/model"
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

	for _, ref := range request.StackReferences {
		c, exists := sm.GetConstruct(ref.ConstructURN.ResourceID)
		if !exists {
			// This means there's a construct in our StackReferences that doesn't exist in the state
			// This should never happen as we just build StackReferences from the state
			return fmt.Errorf("construct %s not found in state", ref.ConstructURN.ResourceID)
		}
		ctx := ConstructContext(ctx, *c.URN)
		if err := sm.TransitionConstructState(&c, model.ConstructDeletePending); err != nil {
			return err
		}
		if err := sm.TransitionConstructState(&c, model.ConstructDeleting); err != nil {
			return err
		}
		err := stack.RunDown(ctx, ref)
		if err != nil {
			if err2 := sm.TransitionConstructState(&c, model.ConstructDeleteFailed); err != nil {
				return fmt.Errorf("%v: error transitioning construct state to delete failed: %v", err, err2)
			}
			return err
		} else {
			if err := sm.TransitionConstructState(&c, model.ConstructDeleteComplete); err != nil {
				return err
			}
		}
	}
	return nil
}
