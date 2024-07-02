package model

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func createTempStateFile(t *testing.T, content string) string {
	tmpFile, err := os.CreateTemp("", "state-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	if content != "" {
		if _, err := tmpFile.Write([]byte(content)); err != nil {
			t.Fatalf("Failed to write to temp file: %v", err)
		}
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}
	t.Logf("Created temp file: %s", tmpFile.Name())
	return tmpFile.Name()
}

func removeTempFile(t *testing.T, filePath string) {
	t.Logf("Removing temp file: %s", filePath)
	if err := os.Remove(filePath); err != nil {
		t.Logf("Failed to remove temp file: %v", err)
	}
}

func TestNewStateManager(t *testing.T) {
	tmpFile := createTempStateFile(t, "")
	defer removeTempFile(t, tmpFile)

	sm := NewStateManager(tmpFile)
	if sm.stateFile != tmpFile {
		t.Errorf("Expected stateFile to be %s, got %s", tmpFile, sm.stateFile)
	}
	if sm.state.SchemaVersion != 1 {
		t.Errorf("Expected SchemaVersion to be 1, got %d", sm.state.SchemaVersion)
	}
	if sm.state.Version != 1 {
		t.Errorf("Expected Version to be 1, got %d", sm.state.Version)
	}
}

func TestCheckStateFileExists(t *testing.T) {
	tmpFile := createTempStateFile(t, "")

	sm := NewStateManager(tmpFile)
	defer func() {
		removeTempFile(t, tmpFile)
		if sm.CheckStateFileExists() {
			t.Errorf("Expected CheckStateFileExists to return false")
		}
	}()
	if !sm.CheckStateFileExists() {
		t.Errorf("Expected CheckStateFileExists to return true")
	}
}

func TestLoadState(t *testing.T) {
	stateContent := `
schemaVersion: 1
version: 1
project_urn: "urn:project:example"
app_urn: "urn:app:example"
environment: "dev"
default_region: "us-west-2"
constructs:
  example-construct:
    status: "creating"
    last_updated: "2023-06-11T00:00:00Z"
    inputs: {}
    outputs: {}
    bindings: []
    options: {}
    dependsOn: []
    urn: "urn:construct:example"
`
	tmpFile := createTempStateFile(t, stateContent)
	defer removeTempFile(t, tmpFile)

	sm := NewStateManager(tmpFile)
	if err := sm.LoadState(); err != nil {
		t.Errorf("Failed to load state: %v", err)
	}
	if sm.state.ProjectURN != "urn:project:example" {
		t.Errorf("Expected ProjectURN to be urn:project:example, got %s", sm.state.ProjectURN)
	}
	if sm.state.AppURN != "urn:app:example" {
		t.Errorf("Expected AppURN to be urn:app:example, got %s", sm.state.AppURN)
	}
	if sm.state.Environment != "dev" {
		t.Errorf("Expected Environment to be dev, got %s", sm.state.Environment)
	}
	if sm.state.DefaultRegion != "us-west-2" {
		t.Errorf("Expected DefaultRegion to be us-west-2, got %s", sm.state.DefaultRegion)
	}
	if construct, exists := sm.state.Constructs["example-construct"]; !exists {
		t.Errorf("Expected construct example-construct to exist")
	} else {
		if construct.Status != ConstructCreating {
			t.Errorf("Expected status to be %s, got %s", ConstructCreating, construct.Status)
		}
		if construct.LastUpdated != "2023-06-11T00:00:00Z" {
			t.Errorf("Expected last updated to be 2023-06-11T00:00:00Z, got %s", construct.LastUpdated)
		}
		if construct.URN.String() != "urn:construct:example" {
			t.Errorf("Expected URN to be urn:construct:example, got %s", construct.URN.String())
		}
	}

	// Test case where os.ReadFile returns an error other than os.ErrNotExist
	removeTempFile(t, tmpFile)
	if err := os.WriteFile(tmpFile, []byte(stateContent), 0000); err != nil {
		t.Fatalf("Failed to write protected state file: %v", err)
	}
	if err := sm.LoadState(); err == nil {
		t.Errorf("Expected error when reading protected state file, got nil")
	} else {
		if !strings.Contains(err.Error(), os.ErrPermission.Error()) {
			t.Errorf("Expected error message to contain '%s', got '%s'", os.ErrPermission.Error(), err.Error())
		}
	}

	// Test case where os.ReadFile returns os.ErrNotExist
	removeTempFile(t, tmpFile)
	if err := sm.LoadState(); err != nil {
		t.Errorf("Expected no error when state file does not exist, got %v", err)
	}
	if sm.state != nil {
		t.Errorf("Expected state to be nil, got %+v", sm.state)
	}
}

// Test invalid YAML case using a custom marshaller to throw an error
type InvalidOutput struct{}

func (InvalidOutput) MarshalYAML() (interface{}, error) {
	return nil, fmt.Errorf("intentional marshal error")
}
func TestSaveState(t *testing.T) {
	tmpFile := createTempStateFile(t, "")
	defer removeTempFile(t, tmpFile)

	sm := NewStateManager(tmpFile)
	sm.state.ProjectURN = "urn:project:example"
	sm.state.AppURN = "urn:app:example"
	sm.state.Environment = "dev"
	sm.state.DefaultRegion = "us-west-2"
	constructURN, _ := ParseURN("urn:construct:example")
	sm.state.Constructs = map[string]ConstructState{
		"example-construct": {
			Status:      ConstructCreating,
			LastUpdated: "2023-06-11T00:00:00Z",
			Inputs:      make(map[string]Input),
			Outputs:     make(map[string]any),
			Bindings:    []Binding{},
			Options:     make(map[string]interface{}),
			DependsOn:   []*URN{},
			URN:         constructURN,
		},
	}

	if err := sm.SaveState(); err != nil {
		t.Errorf("Failed to save state: %v", err)
	}

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Errorf("Failed to read state file: %v", err)
	}

	var state State
	if err := yaml.Unmarshal(data, &state); err != nil {
		t.Errorf("Failed to unmarshal state: %v", err)
	}

	if state.ProjectURN != "urn:project:example" {
		t.Errorf("Expected ProjectURN to be urn:project:example, got %s", state.ProjectURN)
	}
	if state.AppURN != "urn:app:example" {
		t.Errorf("Expected AppURN to be urn:app:example, got %s", state.AppURN)
	}
	if state.Environment != "dev" {
		t.Errorf("Expected Environment to be dev, got %s", state.Environment)
	}
	if state.DefaultRegion != "us-west-2" {
		t.Errorf("Expected DefaultRegion to be us-west-2, got %s", state.DefaultRegion)
	}
	if construct, exists := state.Constructs["example-construct"]; !exists {
		t.Errorf("Expected construct example-construct to exist")
	} else {
		if construct.Status != ConstructCreating {
			t.Errorf("Expected status to be %s, got %s", ConstructCreating, construct.Status)
		}
		if construct.LastUpdated != "2023-06-11T00:00:00Z" {
			t.Errorf("Expected last updated to be 2023-06-11T00:00:00Z, got %s", construct.LastUpdated)
		}
		if construct.URN.String() != "urn:construct:example" {
			t.Errorf("Expected URN to be urn:construct:example, got %s", construct.URN.String())
		}
	}

	sm.state.Constructs["invalid-construct"] = ConstructState{
		URN: constructURN,
		Outputs: map[string]any{
			"invalid": InvalidOutput{},
		},
	}
	if err := sm.SaveState(); err == nil {
		t.Errorf("Expected error for invalid YAML, got nil")
	}
}

func TestInitState(t *testing.T) {
	parsedURN, err := ParseURN("urn:accountid:my-project:dev:my-app:construct/klotho.aws.S3:example-construct")
	if err != nil {
		t.Fatalf("Failed to parse URN: %v", err)
	}

	ir := &ApplicationEnvironment{
		ProjectURN:    "urn:project:example",
		AppURN:        "urn:app:example",
		Environment:   "dev",
		DefaultRegion: "us-west-2",
		Constructs: map[string]Construct{
			"example-construct": {
				Inputs:    make(map[string]Input),
				Outputs:   make(map[string]any),
				Bindings:  []Binding{},
				Options:   make(map[string]any),
				DependsOn: []*URN{},
				URN:       parsedURN,
			},
		},
	}

	tmpFile := createTempStateFile(t, "")
	defer removeTempFile(t, tmpFile)

	sm := NewStateManager(tmpFile)
	sm.InitState(ir)

	if sm.state.ProjectURN != "urn:project:example" {
		t.Errorf("Expected ProjectURN to be urn:project:example, got %s", sm.state.ProjectURN)
	}
	if sm.state.AppURN != "urn:app:example" {
		t.Errorf("Expected AppURN to be urn:app:example, got %s", sm.state.AppURN)
	}
	if sm.state.Environment != "dev" {
		t.Errorf("Expected Environment to be dev, got %s", sm.state.Environment)
	}
	if sm.state.DefaultRegion != "us-west-2" {
		t.Errorf("Expected DefaultRegion to be us-west-2, got %s", sm.state.DefaultRegion)
	}

	construct, exists := sm.GetConstructState("example-construct")
	if !exists {
		t.Fatalf("Expected construct example-construct to exist")
	}
	if construct.Status != ConstructCreating {
		t.Errorf("Expected status to be %s, got %s", ConstructCreating, construct.Status)
	}
	if construct.LastUpdated == "" {
		t.Errorf("Expected last updated to be set, got empty string")
	}
	if construct.Inputs == nil || len(construct.Inputs) != 0 {
		t.Errorf("Expected Inputs to be empty map, got %v", construct.Inputs)
	}
	if construct.Outputs == nil || len(construct.Outputs) != 0 {
		t.Errorf("Expected Outputs to be empty map, got %v", construct.Outputs)
	}
	if construct.Bindings == nil || len(construct.Bindings) != 0 {
		t.Errorf("Expected Bindings to be empty slice, got %v", construct.Bindings)
	}
	if construct.Options == nil || len(construct.Options) != 0 {
		t.Errorf("Expected Options to be empty map, got %v", construct.Options)
	}
	if construct.DependsOn == nil || len(construct.DependsOn) != 0 {
		t.Errorf("Expected DependsOn to be empty slice, got %v", construct.DependsOn)
	}
	if construct.URN.String() != "urn:accountid:my-project:dev:my-app:construct/klotho.aws.S3:example-construct" {
		t.Errorf("Expected URN to be urn:accountid:my-project:dev:my-app:construct/klotho.aws.S3:example-construct, got %s", construct.URN.String())
	}
}

func TestTransitionConstructState(t *testing.T) {
	tmpFile := createTempStateFile(t, "")
	defer removeTempFile(t, tmpFile)

	sm := NewStateManager(tmpFile)
	parsedURN, err := ParseURN("urn:accountid:my-project:dev:my-app:construct/klotho.aws.S3:example-construct")
	if err != nil {
		t.Fatalf("Failed to parse URN: %v", err)
	}

	construct := ConstructState{
		Status: ConstructCreating,
		URN:    parsedURN,
	}
	sm.SetConstructState(construct)

	// Test valid transition
	if err := sm.TransitionConstructState(&construct, ConstructCreateComplete); err != nil {
		t.Errorf("Expected valid transition, got error: %v", err)
	}
	if construct.Status != ConstructCreateComplete {
		t.Errorf("Expected status %s, got %s", ConstructCreateComplete, construct.Status)
	}

	// Update the construct state in the state manager
	sm.SetConstructState(construct)

	// Test invalid transition
	if err := sm.TransitionConstructState(&construct, ConstructCreateComplete); err == nil {
		t.Errorf("Expected error for invalid transition, got nil")
	}
}

func TestTransitionConstructFailed(t *testing.T) {
	tmpFile := createTempStateFile(t, "")
	defer removeTempFile(t, tmpFile)

	sm := NewStateManager(tmpFile)
	parsedURN, err := ParseURN("urn:accountid:my-project:dev:my-app:construct/klotho.aws.S3:example-construct")
	if err != nil {
		t.Fatalf("Failed to parse URN: %v", err)
	}

	construct := ConstructState{
		Status: ConstructCreating,
		URN:    parsedURN,
	}
	sm.SetConstructState(construct)

	// Test valid transition from Creating to CreateFailed
	if err := sm.TransitionConstructFailed(&construct); err != nil {
		t.Errorf("Expected valid transition, got error: %v", err)
	}
	if construct.Status != ConstructCreateFailed {
		t.Errorf("Expected status %s, got %s", ConstructCreateFailed, construct.Status)
	}

	// Update the construct state in the state manager
	sm.SetConstructState(construct)

	// Test valid transition from Updating to UpdateFailed
	construct.Status = ConstructUpdating
	if err := sm.TransitionConstructFailed(&construct); err != nil {
		t.Errorf("Expected valid transition, got error: %v", err)
	}
	if construct.Status != ConstructUpdateFailed {
		t.Errorf("Expected status %s, got %s", ConstructUpdateFailed, construct.Status)
	}

	// Test valid transition from Deleting to DeleteFailed
	construct.Status = ConstructDeleting
	if err := sm.TransitionConstructFailed(&construct); err != nil {
		t.Errorf("Expected valid transition, got error: %v", err)
	}
	if construct.Status != ConstructDeleteFailed {
		t.Errorf("Expected status %s, got %s", ConstructDeleteFailed, construct.Status)
	}

	// Test invalid initial state
	construct.Status = ConstructUnknown
	if err := sm.TransitionConstructFailed(&construct); err == nil {
		t.Errorf("Expected error for invalid initial state, got nil")
	} else {
		expectedErrMsg := fmt.Sprintf("Initial state %s must be one of Creating, Updating, or Deleting", ConstructUnknown)
		if err.Error() != expectedErrMsg {
			t.Errorf("Expected error message to be '%s', got '%s'", expectedErrMsg, err.Error())
		}
	}
}

func TestTransitionConstructComplete(t *testing.T) {
	tmpFile := createTempStateFile(t, "")
	defer removeTempFile(t, tmpFile)

	sm := NewStateManager(tmpFile)
	parsedURN, err := ParseURN("urn:accountid:my-project:dev:my-app:construct/klotho.aws.S3:example-construct")
	if err != nil {
		t.Fatalf("Failed to parse URN: %v", err)
	}

	construct := ConstructState{
		Status: ConstructCreating,
		URN:    parsedURN,
	}
	sm.SetConstructState(construct)

	// Test valid transition from Creating to CreateComplete
	if err := sm.TransitionConstructComplete(&construct); err != nil {
		t.Errorf("Expected valid transition, got error: %v", err)
	}
	if construct.Status != ConstructCreateComplete {
		t.Errorf("Expected status %s, got %s", ConstructCreateComplete, construct.Status)
	}

	// Update the construct state in the state manager
	sm.SetConstructState(construct)

	// Test valid transition from Updating to UpdateComplete
	construct.Status = ConstructUpdating
	if err := sm.TransitionConstructComplete(&construct); err != nil {
		t.Errorf("Expected valid transition, got error: %v", err)
	}
	if construct.Status != ConstructUpdateComplete {
		t.Errorf("Expected status %s, got %s", ConstructUpdateComplete, construct.Status)
	}

	// Test valid transition from Deleting to DeleteComplete
	construct.Status = ConstructDeleting
	if err := sm.TransitionConstructComplete(&construct); err != nil {
		t.Errorf("Expected valid transition, got error: %v", err)
	}
	if construct.Status != ConstructDeleteComplete {
		t.Errorf("Expected status %s, got %s", ConstructDeleteComplete, construct.Status)
	}

	// Test invalid initial state
	construct.Status = ConstructUnknown
	if err := sm.TransitionConstructComplete(&construct); err == nil {
		t.Errorf("Expected error for invalid initial state, got nil")
	} else {
		expectedErrMsg := fmt.Sprintf("Initial state %s must be one of Creating, Updating, or Deleting", ConstructUnknown)
		if err.Error() != expectedErrMsg {
			t.Errorf("Expected error message to be '%s', got '%s'", expectedErrMsg, err.Error())
		}
	}
}

func TestUpdateResourceState(t *testing.T) {
	tmpFile := createTempStateFile(t, "")
	defer removeTempFile(t, tmpFile)

	sm := NewStateManager(tmpFile)

	parsedURN, err := ParseURN("urn:accountid:my-project:dev:my-app:construct/klotho.aws.S3:example-construct")
	if err != nil {
		t.Fatalf("Failed to parse URN: %v", err)
	}

	// Initialize the construct state in the StateManager
	sm.SetConstructState(ConstructState{
		Status: ConstructCreating,
		URN:    parsedURN,
	})

	// Test valid update
	if err := sm.UpdateResourceState("example-construct", ConstructCreateComplete, "2023-06-11T00:00:00Z"); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	construct, exists := sm.GetConstructState("example-construct")
	if !exists {
		t.Errorf("Expected construct example-construct to exist")
	}
	if construct.Status != ConstructCreateComplete {
		t.Errorf("Expected status to be %s, got %s", ConstructCreateComplete, construct.Status)
	}
	if construct.LastUpdated != "2023-06-11T00:00:00Z" {
		t.Errorf("Expected last updated to be 2023-06-11T00:00:00Z, got %s", construct.LastUpdated)
	}

	// Test invalid transition
	if err := sm.UpdateResourceState("example-construct", ConstructCreateComplete, "2023-06-12T00:00:00Z"); err == nil {
		t.Errorf("Expected error for invalid transition, got nil")
	} else {
		expectedErrMsg := "invalid transition from create_complete to create_complete"
		if err.Error() != expectedErrMsg {
			t.Errorf("Expected error message to be '%s', got '%s'", expectedErrMsg, err.Error())
		}
	}

	// Test construct not found
	if err := sm.UpdateResourceState("non-existent-construct", ConstructCreateComplete, "2023-06-11T00:00:00Z"); err == nil {
		t.Errorf("Expected error for non-existent construct, got nil")
	} else {
		expectedErrMsg := "construct non-existent-construct not found"
		if err.Error() != expectedErrMsg {
			t.Errorf("Expected error message to be '%s', got '%s'", expectedErrMsg, err.Error())
		}
	}

	// Test case where sm.state.Constructs is nil
	sm.state.Constructs = nil
	if err := sm.UpdateResourceState("example-construct", ConstructCreateComplete, "2023-06-11T00:00:00Z"); err == nil {
		t.Errorf("Expected error for construct not found when state constructs is nil, got nil")
	}
}

func TestGetState(t *testing.T) {
	tmpFile := createTempStateFile(t, "")
	defer removeTempFile(t, tmpFile)

	sm := NewStateManager(tmpFile)
	sm.state.ProjectURN = "urn:project:example"
	sm.state.AppURN = "urn:app:example"
	sm.state.Environment = "dev"
	sm.state.DefaultRegion = "us-west-2"

	state := sm.GetState()
	if state.ProjectURN != "urn:project:example" {
		t.Errorf("Expected ProjectURN to be urn:project:example, got %s", state.ProjectURN)
	}
	if state.AppURN != "urn:app:example" {
		t.Errorf("Expected AppURN to be urn:app:example, got %s", state.AppURN)
	}
	if state.Environment != "dev" {
		t.Errorf("Expected Environment to be dev, got %s", state.Environment)
	}
	if state.DefaultRegion != "us-west-2" {
		t.Errorf("Expected DefaultRegion to be us-west-2, got %s", state.DefaultRegion)
	}
}

func TestGetAllConstructs(t *testing.T) {
	tmpFile := createTempStateFile(t, "")
	defer removeTempFile(t, tmpFile)

	sm := NewStateManager(tmpFile)
	constructURN, _ := ParseURN("urn:construct:example")
	sm.state.Constructs = map[string]ConstructState{
		"example-construct": {
			Status:      ConstructCreating,
			LastUpdated: "2023-06-11T00:00:00Z",
			Inputs:      make(map[string]Input),
			Outputs:     make(map[string]any),
			Bindings:    []Binding{},
			Options:     make(map[string]interface{}),
			DependsOn:   []*URN{},
			URN:         constructURN,
		},
	}

	constructs := sm.GetAllConstructs()
	if len(constructs) != 1 {
		t.Errorf("Expected 1 construct, got %d", len(constructs))
	}
	if construct, exists := constructs["example-construct"]; !exists {
		t.Errorf("Expected construct example-construct to exist")
	} else {
		if construct.Status != ConstructCreating {
			t.Errorf("Expected status to be %s, got %s", ConstructCreating, construct.Status)
		}
		if construct.LastUpdated != "2023-06-11T00:00:00Z" {
			t.Errorf("Expected last updated to be 2023-06-11T00:00:00Z, got %s", construct.LastUpdated)
		}
		if construct.URN.String() != "urn:construct:example" {
			t.Errorf("Expected URN to be urn:construct:example, got %s", construct.URN.String())
		}
	}
}

func TestRegisterOutputValues(t *testing.T) {
	tmpFile := createTempStateFile(t, "")
	defer removeTempFile(t, tmpFile)

	sm := NewStateManager(tmpFile)

	constructURN, _ := ParseURN("urn:accountid:my-project:dev:my-app:construct/klotho.aws.Container:my-container")
	dependentURN, _ := ParseURN("urn:accountid:my-project:dev:my-app:construct/klotho.aws.Service:dependent-service")

	construct := ConstructState{
		Status:      ConstructCreating,
		URN:         constructURN,
		Outputs:     nil,
		Inputs:      make(map[string]Input),
		LastUpdated: "2023-06-11T00:00:00Z",
	}
	dependentConstruct := ConstructState{
		Status: ConstructCreating,
		URN:    dependentURN,
		Inputs: map[string]Input{
			"image": {
				Value:     nil,
				Encrypted: false,
				Status:    InputStatusPending,
				DependsOn: "urn:accountid:my-project:dev:my-app:construct/klotho.aws.Container:my-container",
			},
			"port": {
				Value:     nil,
				Encrypted: false,
				Status:    InputStatusPending,
				DependsOn: "urn:accountid:my-project:dev:my-app:construct/klotho.aws.Container:my-container",
			},
		},
	}

	sm.SetConstructState(construct)
	sm.SetConstructState(dependentConstruct)

	// Case with valid outputs
	outputs := map[string]any{
		"image": "nginx:latest",
		"port":  80,
	}

	err := sm.RegisterOutputValues(*constructURN, outputs)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	updatedConstruct, exists := sm.GetConstructState("my-container")
	if !exists {
		t.Errorf("Expected construct my-container to exist")
	}

	if !reflect.DeepEqual(updatedConstruct.Outputs, outputs) {
		t.Errorf("Expected Outputs to be %v, got %v", outputs, updatedConstruct.Outputs)
	}

	updatedDependentConstruct, exists := sm.GetConstructState("dependent-service")
	if !exists {
		t.Errorf("Expected construct dependent-service to exist")
	}

	expectedInputs := map[string]Input{
		"image": {
			Value:     "nginx:latest",
			Encrypted: false,
			Status:    InputStatusResolved,
			DependsOn: "urn:accountid:my-project:dev:my-app:construct/klotho.aws.Container:my-container",
		},
		"port": {
			Value:     80,
			Encrypted: false,
			Status:    InputStatusResolved,
			DependsOn: "urn:accountid:my-project:dev:my-app:construct/klotho.aws.Container:my-container",
		},
	}

	if !reflect.DeepEqual(updatedDependentConstruct.Inputs, expectedInputs) {
		t.Errorf("Expected Inputs to be %v, got %v", expectedInputs, updatedDependentConstruct.Inputs)
	}

	// Test with invalid construct URN
	invalidURN, _ := ParseURN("urn:accountid:my-project:dev:my-app:construct/klotho.aws.Container:invalid-container")
	err = sm.RegisterOutputValues(*invalidURN, outputs)
	if err == nil {
		t.Errorf("Expected error for non-existent construct, got nil")
	}

	// Case with no outputs
	err = sm.RegisterOutputValues(*constructURN, nil)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	updatedConstruct, exists = sm.GetConstructState("my-container")
	if !exists {
		t.Errorf("Expected construct my-container to exist")
	}

	// Expected Outputs should still be the original outputs since nil should not change the map
	if !reflect.DeepEqual(updatedConstruct.Outputs, outputs) {
		t.Errorf("Expected Outputs to be %v, got %v", outputs, updatedConstruct.Outputs)
	}

	// Case where sm.state.Constructs is nil
	sm.state.Constructs = nil
	err = sm.RegisterOutputValues(*constructURN, outputs)
	if err == nil {
		t.Errorf("Expected error for constructs not being initialized, got nil")
	}
}
