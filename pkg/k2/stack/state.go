package stack

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optdestroy"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optpreview"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
	"go.uber.org/zap"
)

type State struct {
	Version    int
	Deployment apitype.DeploymentV3
	Outputs    map[string]any
	Resources  map[construct.ResourceId]apitype.ResourceV3
}

type StackInterface interface {
	Export(ctx context.Context) (apitype.UntypedDeployment, error)
	Up(ctx context.Context, opts ...optup.Option) (auto.UpResult, error)
	Preview(ctx context.Context, opts ...optpreview.Option) (auto.PreviewResult, error)
	Destroy(ctx context.Context, opts ...optdestroy.Option) (auto.DestroyResult, error)
	SetConfig(ctx context.Context, key string, value auto.ConfigValue) error
	Workspace() auto.Workspace
}

// GetState retrieves the state of a stack
func GetState(ctx context.Context, stack StackInterface) (State, error) {
	rawState, err := stack.Export(ctx)
	if err != nil {
		return State{}, err
	}

	unmarshalledState := apitype.DeploymentV3{}
	err = json.Unmarshal(rawState.Deployment, &unmarshalledState)
	if err != nil {
		return State{}, err
	}

	zap.S().Debugf("unmarshalled state: %v", unmarshalledState)

	var stackResource apitype.ResourceV3
	foundStackResource := false
	for _, res := range unmarshalledState.Resources {
		if res.Type == "pulumi:pulumi:Stack" {
			stackResource = res
			foundStackResource = true
			break
		}
	}
	if !foundStackResource {
		return State{}, fmt.Errorf("could not find pulumi:pulumi:Stack resource in state")
	}

	stackOutputs := make(map[string]any)
	outputs, ok := stackResource.Outputs["$outputs"].(map[string]any)
	if !ok {
		return State{}, fmt.Errorf("failed to parse stack outputs")
	}
	for key, value := range outputs {
		stackOutputs[key] = value
	}

	resourceIdByUrn := make(map[string]string)
	urns, ok := stackResource.Outputs["$urns"].(map[string]any)
	if !ok {
		return State{}, fmt.Errorf("failed to parse resource URNs")
	}
	for id, rawUrn := range urns {
		if urn, ok := rawUrn.(string); ok {
			resourceIdByUrn[urn] = id
		} else {
			zap.S().Warnf("could not convert urn %v to string", rawUrn)
		}
	}

	resourcesByResourceId := make(map[construct.ResourceId]apitype.ResourceV3)
	for _, res := range unmarshalledState.Resources {
		id, ok := resourceIdByUrn[string(res.URN)]
		if !ok {
			zap.S().Warnf("could not find resource id for urn %s", res.URN)
			continue
		}
		var parsedId construct.ResourceId
		err := parsedId.Parse(id)
		if err != nil {
			zap.S().Warnf("could not parse resource id %s: %v", id, err)
			continue
		}
		resourcesByResourceId[parsedId] = res
	}

	return State{
		Version:    rawState.Version,
		Deployment: unmarshalledState,
		Outputs:    stackOutputs,
		Resources:  resourcesByResourceId,
	}, nil
}

func UpdateConstructStateFromUpResult(sm *model.StateManager, stackReference Reference, summary *auto.UpResult) error {
	constructName := stackReference.ConstructURN.ResourceID
	c, exists := sm.GetConstructState(constructName)
	if !exists {
		return fmt.Errorf("construct %s not found in state", constructName)
	}

	nextStatus := determineNextStatus(c.Status, summary.Summary.Result)
	if err := sm.TransitionConstructState(&c, nextStatus); err != nil {
		return fmt.Errorf("failed to transition construct state: %v", err)
	}
	c.LastUpdated = time.Now().Format(time.RFC3339)
	sm.SetConstructState(c)

	return nil
}

func determineNextStatus(currentStatus model.ConstructStatus, result string) model.ConstructStatus {
	switch currentStatus {
	case model.ConstructCreating:
		if result == "succeeded" {
			return model.ConstructCreateComplete
		}
		return model.ConstructCreateFailed
	case model.ConstructUpdating:
		if result == "succeeded" {
			return model.ConstructUpdateComplete
		}
		return model.ConstructUpdateFailed
	case model.ConstructDeleting:
		if result == "succeeded" {
			return model.ConstructDeleteComplete
		}
		return model.ConstructDeleteFailed

	default:
		return model.ConstructUnknown
	}
}

type StateManager struct {
	ConstructStackState map[model.URN]State
}

func NewStateManager() *StateManager {
	return &StateManager{
		ConstructStackState: make(map[model.URN]State),
	}
}

func (sm *StateManager) GetResourceState(urn model.URN, id construct.ResourceId) (apitype.ResourceV3, bool) {
	stackState, exists := sm.ConstructStackState[urn]
	if !exists {
		return apitype.ResourceV3{}, false
	}

	res, exists := stackState.Resources[id]
	return res, exists
}
