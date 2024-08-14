package stack

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
)

// Mock stack that embeds auto.Stack and overrides the Export method
type mockStack struct {
	auto.Stack
	rawState apitype.UntypedDeployment
	err      error
	outputs  auto.OutputMap
}

func (m *mockStack) Export(ctx context.Context) (apitype.UntypedDeployment, error) {
	return m.rawState, m.err
}
func (m *mockStack) Outputs(ctx context.Context) (auto.OutputMap, error) {
	return m.outputs, nil
}

func emptyOutputs() auto.OutputMap {
	return auto.OutputMap{
		"$outputs": auto.OutputValue{Value: map[string]any{}},
		"$urns":    auto.OutputValue{Value: map[string]any{}},
	}

}

func TestGetState(t *testing.T) {
	tests := []struct {
		name          string
		rawState      apitype.UntypedDeployment
		outputs       auto.OutputMap
		expectedError string
		expectedState State
	}{
		{
			name:    "Empty State",
			outputs: emptyOutputs(),
			rawState: apitype.UntypedDeployment{
				Version:    3,
				Deployment: json.RawMessage(`{"resources": [{"type": "pulumi:pulumi:Stack", "urn": "urn:pulumi:stack::project::resource", "outputs": {"$outputs": {}, "$urns": {}}}]}`),
			},
			expectedError: "",
			expectedState: State{
				Version:    3,
				Deployment: apitype.DeploymentV3{},
				Outputs:    map[string]any{},
				Resources:  map[construct.ResourceId]apitype.ResourceV3{},
			},
		},
		{
			name:    "Malformed URN and Outputs",
			outputs: auto.OutputMap{},
			rawState: apitype.UntypedDeployment{
				Version:    3,
				Deployment: json.RawMessage(`{"resources": [{"type": "pulumi:pulumi:Stack", "urn": "urn:pulumi:stack::project::resource", "outputs": {}}]}`),
			},
			expectedError: "$outputs not found in stack outputs",
		},
		{
			name: "Valid State with Multiple Resources",
			outputs: auto.OutputMap{
				"$outputs": auto.OutputValue{Value: map[string]any{"outputKey": "outputValue"}},
				"$urns":    auto.OutputValue{Value: map[string]any{"provider:type:namespace:name": "urn:pulumi:stack::project::resource"}},
			},
			rawState: apitype.UntypedDeployment{
				Version: 3,
				Deployment: json.RawMessage(`{
					"resources": [
						{
							"type": "pulumi:pulumi:Stack",
							"urn": "urn:pulumi:stack::project::resource",
							"outputs": {
								"$outputs": {
									"outputKey": "outputValue"
								},
								"$urns": {
									"provider:type:namespace:name": "urn:pulumi:stack::project::resource"
								}
							}
						},
						{
							"type": "example:type:Resource",
							"urn": "urn:pulumi:stack::project::example-resource",
							"outputs": {}
						}
					]
				}`),
			},
			expectedError: "",
			expectedState: State{
				Version: 3,
				Deployment: apitype.DeploymentV3{
					Resources: []apitype.ResourceV3{
						{
							Type: "pulumi:pulumi:Stack",
							URN:  "urn:pulumi:stack::project::resource",
							Outputs: map[string]any{
								"$outputs": map[string]any{
									"outputKey": "outputValue",
								},
								"$urns": map[string]any{
									"provider:type:namespace:name": "urn:pulumi:stack::project::resource",
								},
							},
						},
						{
							Type:    "example:type:Resource",
							URN:     "urn:pulumi:stack::project::example-resource",
							Outputs: map[string]any{},
						},
					},
				},
				Outputs: map[string]any{
					"outputKey": "outputValue",
				},
				Resources: map[construct.ResourceId]apitype.ResourceV3{
					{Provider: "provider", Type: "type", Namespace: "namespace", Name: "name"}: {
						Type: "pulumi:pulumi:Stack",
						URN:  "urn:pulumi:stack::project::resource",
						Outputs: map[string]any{
							"$outputs": map[string]any{
								"outputKey": "outputValue",
							},
							"$urns": map[string]any{
								"provider:type:namespace:name": "urn:pulumi:stack::project::resource",
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockStack{
				rawState: tt.rawState,
				outputs:  tt.outputs,
				err:      nil,
			}

			ctx := context.Background()
			state, err := GetState(ctx, mock)

			if tt.expectedError != "" {
				if err == nil || err.Error() != tt.expectedError {
					t.Fatalf("expected error '%v', got '%v'", tt.expectedError, err)
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if state.Version != tt.expectedState.Version {
					t.Fatalf("expected version %d, got %d", tt.expectedState.Version, state.Version)
				}
				if len(state.Resources) != len(tt.expectedState.Resources) {
					t.Fatalf("expected %d resources, got %d", len(tt.expectedState.Resources), len(state.Resources))
				}
				if len(state.Outputs) != len(tt.expectedState.Outputs) {
					t.Fatalf("expected %d outputs, got %d", len(tt.expectedState.Outputs), len(state.Outputs))
				}
			}
		})
	}
}

func TestUpdateConstructStateFromUpResult(t *testing.T) {
	sm := model.NewStateManager(nil, "")
	stackReference := Reference{
		ConstructURN: model.URN{ResourceID: "constructName"},
	}
	summary := &auto.UpResult{
		Summary: auto.UpdateSummary{
			Result: "succeeded",
		},
	}

	constructState := model.ConstructState{
		URN:    &stackReference.ConstructURN,
		Status: model.ConstructCreating,
	}
	sm.SetConstructState(constructState)

	err := UpdateConstructStateFromUpResult(sm, stackReference, summary)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	c, exists := sm.GetConstructState("constructName")
	if !exists {
		t.Fatal("expected construct state to exist")
	}

	if c.Status != model.ConstructCreateComplete {
		t.Fatalf("expected status %s, got %s", model.ConstructCreateComplete, c.Status)
	}

	if c.LastUpdated == "" {
		t.Fatal("expected last updated to be set")
	}
}

func TestDetermineNextStatus(t *testing.T) {
	tests := []struct {
		currentStatus model.ConstructStatus
		result        string
		expected      model.ConstructStatus
	}{
		{model.ConstructCreating, "succeeded", model.ConstructCreateComplete},
		{model.ConstructCreating, "failed", model.ConstructCreateFailed},
		{model.ConstructUpdating, "succeeded", model.ConstructUpdateComplete},
		{model.ConstructUpdating, "failed", model.ConstructUpdateFailed},
		{model.ConstructDeleting, "succeeded", model.ConstructDeleteComplete},
		{model.ConstructDeleting, "failed", model.ConstructDeleteFailed},
		{model.ConstructUnknown, "succeeded", model.ConstructUnknown},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s-%s", test.currentStatus, test.result), func(t *testing.T) {
			actual := determineNextStatus(test.currentStatus, test.result)
			if actual != test.expected {
				t.Fatalf("expected %s, got %s", test.expected, actual)
			}
		})
	}
}

func TestStateManager_GetResourceState(t *testing.T) {
	// Create a new StateManager
	sm := NewStateManager()

	// Define a resource ID and its state
	resourceId := construct.ResourceId{
		Provider:  "example",
		Type:      "type",
		Namespace: "namespace",
		Name:      "name",
	}
	resourceState := apitype.ResourceV3{
		URN: "urn:pulumi:stack::project::example-resource",
	}

	// Create and set stack state with the resource
	state := State{
		Version:   1,
		Resources: map[construct.ResourceId]apitype.ResourceV3{resourceId: resourceState},
	}
	sm.ConstructStackState[model.URN{ResourceID: "testConstruct"}] = state

	// Retrieve the resource state
	res, exists := sm.GetResourceState(model.URN{ResourceID: "testConstruct"}, resourceId)

	// Verify the resource exists and matches the expected state
	if !exists {
		t.Fatalf("expected resource %v to exist", resourceId)
	}
	if res.URN != resourceState.URN {
		t.Fatalf("expected URN %v, got %v", resourceState.URN, res.URN)
	}
}

func TestStateManager_SetConstructState(t *testing.T) {
	// Create a new StateManager
	sm := model.NewStateManager(nil, "")

	// Define a construct state
	constructState := model.ConstructState{
		URN:    &model.URN{ResourceID: "testConstruct"},
		Status: model.ConstructCreating,
	}

	// Set the construct state
	sm.SetConstructState(constructState)

	// Retrieve the construct state
	retrievedState, exists := sm.GetConstructState("testConstruct")

	// Verify the construct state exists and matches the expected state
	if !exists {
		t.Fatalf("expected construct state to exist")
	}
	if retrievedState.Status != constructState.Status {
		t.Fatalf("expected status %v, got %v", constructState.Status, retrievedState.Status)
	}
	if retrievedState.URN.ResourceID != constructState.URN.ResourceID {
		t.Fatalf("expected URN %v, got %v", constructState.URN.ResourceID, retrievedState.URN.ResourceID)
	}
}
