package orchestration

import (
	"context"
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
			prog.UpdateIndeterminate(fmt.Sprintf("Starting %s", action))
		}
	}

	sm := uo.StateManager
	defer func() {
		err = sm.SaveState()
		if err != nil {
			log.Errorf("Error saving state: %v", err)
		}
	}()

	for _, group := range deployOrder {
		for _, cURN := range group {
			if err != nil {
				return err
			}

			c := uo.StateManager.GetState().Constructs[cURN.ResourceID]

			outDir := filepath.Join(uo.OutputDirectory, c.URN.ResourceID)

			ctx := ConstructContext(ctx, *c.URN)
			ctx = debug.WithDebugDir(ctx, outDir)

			// Run pulumi down command for deleted constructs
			if actions[*c.URN] == model.ConstructActionDelete && model.IsDeletable(c.Status) {

				if dryRun {
					log.Infof("Dry run: Skipping pulumi down for deleted construct %s", c.URN.ResourceID)
					continue
				}

				// Mark as deleting
				if err := sm.TransitionConstructState(&c, model.ConstructDeleting); err != nil {
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
				if err := sm.TransitionConstructState(&c, model.ConstructDeleteComplete); err != nil {
					return err
				}
				continue
			}

			// Only proceed if the construct is deployable
			if !model.IsDeployable(c.Status) {
				continue
			}

			// Evaluate the construct
			stackRef, err := uo.EvaluateConstruct(ctx, *uo.StateManager.GetState(), *c.URN)
			if err != nil {
				return fmt.Errorf("error evaluating construct: %w", err)
			}

			if dryRun {
				_, err = stack.RunPreview(ctx, stackRef)
				if err != nil {
					return err
				}
				continue
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
			err = sm.RegisterOutputValues(stackRef.ConstructURN, resolvedOutputs)
			if err != nil {
				return fmt.Errorf("error registering resolved output values: %w", err)
			}

			prog := tui.GetProgress(ctx)
			prog.Complete("Success")
		}
	}
	return err
}

func (uo *UpOrchestrator) resolveOutputValues(stackReference stack.Reference, stackState stack.State) (map[string]any, error) {
	outputs := map[string]map[string]interface{}{
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
