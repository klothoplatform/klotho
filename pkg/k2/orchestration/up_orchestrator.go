package orchestration

import (
	"context"
	"fmt"
	"path/filepath"

	errors2 "github.com/klothoplatform/klotho/pkg/errors"
	"github.com/klothoplatform/klotho/pkg/k2/deployment"
	pb "github.com/klothoplatform/klotho/pkg/k2/language_host/go"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/klothoplatform/klotho/pkg/k2/pulumi"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// UpOrchestrator handles the "up" command
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

func (uo *UpOrchestrator) RunUpCommand(ir *model.ApplicationEnvironment, dryRun bool) error {
	actions, err := uo.resolveInitialState(ir)
	if err != nil {
		return errors2.WrapErrf(err, "error resolving initial state")
	}
	zap.S().Infof("Pending Actions:")
	for k, v := range actions {
		zap.S().Infof("%s: %s", k.String(), v)
	}

	var cs []model.ConstructState
	constructState := uo.StateManager.GetState().Constructs
	for cURN := range actions {
		cs = append(cs, constructState[cURN.ResourceID])
	}

	deployOrder, err := sortConstructsByDependency(cs, actions)
	if err != nil {
		return errors2.WrapErrf(err, "failed to determine deployment order")
	}

	deployer := deployment.Deployer{
		StateManager: uo.StateManager}

	sm := uo.StateManager
	defer func() {
		err = sm.SaveState()
		if err != nil {
			zap.S().Errorf("Error saving state: %v", err)
		}
	}()

	for _, group := range deployOrder {
		for _, cURN := range group {
			if err != nil {
				return err
			}

			c := uo.StateManager.GetState().Constructs[cURN.ResourceID]

			// Run pulumi down command for deleted constructs
			if actions[*c.URN] == model.ConstructActionDelete && model.IsDeletable(c.Status) {
				// Mark as destroyed
				if err := sm.TransitionConstructState(&c, model.ConstructDeleting); err != nil {
					return err
				}

				err = pulumi.RunStackDown(pulumi.StackReference{
					ConstructURN: *c.URN,
					Name:         c.URN.ResourceID,
					IacDirectory: filepath.Join(uo.OutputDirectory, c.URN.ResourceID),
					AwsRegion:    sm.GetState().DefaultRegion,
				})

				if err != nil {
					return errors2.WrapErrf(err, "error running pulumi down command")
				}

				// Mark as destroyed
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
			stackRef, err := uo.EvaluateConstruct(*uo.StateManager.GetState(), *c.URN)
			if err != nil {
				return errors2.WrapErrf(err, "error evaluating construct")
			}

			// Run pulumi up command for the construct
			if dryRun {
				_, err = deployer.RunStackPreviewCommand(stackRef)
				if err != nil {
					return err
				}
				return nil
			}

			if err := transitionPendingToDoing(sm, &c); err != nil {
				return errors2.WrapErrf(err, "error transitioning construct state")
			}

			upResult, stackState, err := deployer.RunStackUpCommand(stackRef)
			if err != nil {
				return err
			}

			// Resolve output values
			err2 := uo.resolveOutputValues(stackRef, stackState)
			if err2 != nil {
				return err2
			}

			// Update construct state based on the up result
			err = pulumi.UpdateConstructStateFromUpResult(sm, stackRef, &upResult)
			if err != nil {
				return err
			}
		}
	}
	return err
}

func (uo *UpOrchestrator) resolveOutputValues(stackReference pulumi.StackReference, stackState pulumi.StackState) error {
	// TODO: This is a demo implementation that passes the stack outputs to the language host
	//       and gets the resolved output references back.
	//       It doesn't actually do anything with the resolved outputs yet.
	outputs := map[string]map[string]interface{}{
		stackReference.ConstructURN.String(): stackState.Outputs,
	}
	payload, err := yaml.Marshal(outputs)
	if err != nil {
		return err
	}
	resp, err := uo.LanguageHostClient.RegisterConstruct(context.Background(), &pb.RegisterConstructRequest{
		YamlPayload: string(payload),
	})
	zap.S().Info(resp.GetMessage())
	var resolvedOutputs []any
	for _, o := range resp.GetResolvedOutputs() {
		if err != nil {
			return err
		}
		resolvedOutputs = append(resolvedOutputs, map[string]interface{}{
			"id":    o.GetId(),
			"value": o.GetYamlPayload(),
		})
	}
	zap.S().Infof("Resolved Outputs: %v", resolvedOutputs)
	if err != nil {
		return err
	}
	return nil
}
