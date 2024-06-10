package pulumi

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/klothoplatform/klotho/pkg/construct"
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
	for key, value := range stackResource.Outputs["$outputs"].(map[string]any) {
		stackOutputs[key] = value
	}

	resourceIdByUrn := make(map[string]string)
	for id, rawUrn := range stackResource.Outputs["$urns"].(map[string]any) {
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
	}, err
}
