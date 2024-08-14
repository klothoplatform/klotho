package stack

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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
	Outputs(ctx context.Context) (auto.OutputMap, error)
}

// GetState retrieves the state of a stack
func GetState(ctx context.Context, stack StackInterface) (State, error) {

	rawOutputs, err := stack.Outputs(ctx)
	if err != nil {
		return State{}, err
	}

	stackOutputs, err := GetStackOutputs(rawOutputs)
	if err != nil {
		return State{}, err
	}

	resourceIdByUrn, err := GetResourceIdByURNMap(rawOutputs)
	if err != nil {
		return State{}, err

	}
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

	resourcesByResourceId := make(map[construct.ResourceId]apitype.ResourceV3)
	for _, res := range unmarshalledState.Resources {
		resType := res.URN.QualifiedType()
		switch {
		case strings.HasPrefix(string(resType), "pulumi:"), strings.HasPrefix(string(resType), "docker:"):
			// Skip known non-cloud / Pulumi internal resource (eg: Stack or Provider)
			continue
		}
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

func GetStackOutputs(rawOutputs auto.OutputMap) (map[string]any, error) {
	stackOutputs := make(map[string]any)
	outputs, ok := rawOutputs["$outputs"]
	if !ok {
		return nil, fmt.Errorf("$outputs not found in stack outputs")
	}

	outputsValue, ok := outputs.Value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("failed to parse stack outputs")
	}
	for key, value := range outputsValue {
		stackOutputs[key] = value
	}
	return stackOutputs, nil
}

func GetResourceIdByURNMap(rawOutputs auto.OutputMap) (map[string]string, error) {
	urns, ok := rawOutputs["$urns"]
	if !ok {
		return nil, fmt.Errorf("$urns not found in stack outputs")
	}
	urnsValue, ok := urns.Value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("failed to parse URNs")
	}
	resourceIdByUrn := make(map[string]string)
	for id, rawUrn := range urnsValue {
		if urn, ok := rawUrn.(string); ok {
			resourceIdByUrn[urn] = id
		} else {
			zap.S().Warnf("could not convert urn %v to string", rawUrn)
		}
	}
	return resourceIdByUrn, nil
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
