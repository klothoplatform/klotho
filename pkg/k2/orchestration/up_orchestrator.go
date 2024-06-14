package orchestration

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	pb "github.com/klothoplatform/klotho/pkg/k2/language_host/go"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/klothoplatform/klotho/pkg/k2/pulumi"
	"github.com/klothoplatform/klotho/pkg/logging"
	"gopkg.in/yaml.v3"
)

type UpOrchestrator struct {
	*Orchestrator
	LanguageHostClient pb.KlothoServiceClient
}

func NewUpOrchestrator(sm *model.StateManager, languageHostClient pb.KlothoServiceClient, outputPath string) *UpOrchestrator {
	return &UpOrchestrator{
		Orchestrator:       NewOrchestrator(sm, outputPath),
		LanguageHostClient: languageHostClient,
	}
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

	log.Infof("Pending Actions:")
	for k, v := range actions {
		log.Infof("%s: %s", k.String(), v)
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
			ctx := ConstructContext(ctx, *c.URN)

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

				err = pulumi.RunStackDown(ctx, pulumi.StackReference{
					ConstructURN: *c.URN,
					Name:         c.URN.ResourceID,
					IacDirectory: filepath.Join(uo.OutputDirectory, c.URN.ResourceID),
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
				_, err = pulumi.RunStackPreview(ctx, stackRef)
				if err != nil {
					return err
				}
				continue
			}

			if err = transitionPendingToDoing(sm, &c); err != nil {
				return fmt.Errorf("error transitioning construct state: %w", err)
			}

			// Run pulumi up command for the construct
			upResult, stackState, err := pulumi.RunStackUp(ctx, stackRef)
			if err != nil {
				return fmt.Errorf("error running pulumi up command: %w", err)
			}

			err = sm.RegisterOutputValues(stackRef.ConstructURN, stackState.Outputs)
			if err != nil {
				return fmt.Errorf("error registering output values: %w", err)
			}

			// Update construct state based on the up result
			err = pulumi.UpdateConstructStateFromUpResult(sm, stackRef, &upResult)
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

		}
	}
	return err
}

func (uo *UpOrchestrator) resolveOutputValues(stackReference pulumi.StackReference, stackState pulumi.StackState) (map[string]any, error) {
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
