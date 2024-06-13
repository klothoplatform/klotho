package pulumi

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
	"go.uber.org/zap"
)

type StackState struct {
	Version    int
	Deployment apitype.DeploymentV3
	Outputs    map[string]any
	Resources  map[construct.ResourceId]apitype.ResourceV3
}

func GetStackState(ctx context.Context, stack auto.Stack) (StackState, error) {
	rawState, err := stack.Export(ctx)
	if err != nil {
		return StackState{}, err
	}

	unmarshalledState := apitype.DeploymentV3{}
	err = json.Unmarshal(rawState.Deployment, &unmarshalledState)
	if err != nil {
		return StackState{}, err
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
		return StackState{}, fmt.Errorf("could not find pulumi:pulumi:Stack resource in state")
	}

	stackOutputs := make(map[string]any)
	outputs, ok := stackResource.Outputs["$outputs"].(map[string]any)
	if !ok {
		return StackState{}, fmt.Errorf("failed to parse stack outputs")
	}
	for key, value := range outputs {
		stackOutputs[key] = value
	}

	resourceIdByUrn := make(map[string]string)
	urns, ok := stackResource.Outputs["$urns"].(map[string]any)
	if !ok {
		return StackState{}, fmt.Errorf("failed to parse resource URNs")
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

	return StackState{
		Version:    rawState.Version,
		Deployment: unmarshalledState,
		Outputs:    stackOutputs,
		Resources:  resourcesByResourceId,
	}, nil
}

func UpdateConstructStateFromUpResult(sm *model.StateManager, stackReference StackReference, summary *auto.UpResult) error {
	constructName := stackReference.ConstructURN.ResourceID
	construct, exists := sm.GetConstruct(constructName)
	if !exists {
		return fmt.Errorf("construct %s not found in state", constructName)
	}

	nextStatus := determineNextStatus(construct.Status, summary.Summary.Result)
	if err := sm.TransitionConstructState(&construct, nextStatus); err != nil {
		return fmt.Errorf("failed to transition construct state: %v", err)
	}
	construct.LastUpdated = time.Now().Format(time.RFC3339)
	sm.SetConstruct(construct)

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
