package model

import (
	"os"
	"testing"

	"github.com/google/uuid"
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
		t.Fatalf("Failed to remove temp file: %v", err)
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
    status: "pending"
    last_updated: "2023-06-11T00:00:00Z"
    inputs: {}
    outputs: {}
    bindings: []
    options: {}
    dependsOn: []
    pulumi_stack: "123e4567-e89b-12d3-a456-426614174000"
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
		if construct.Status != ConstructPending {
			t.Errorf("Expected status to be %s, got %s", ConstructPending, construct.Status)
		}
		if construct.LastUpdated != "2023-06-11T00:00:00Z" {
			t.Errorf("Expected last updated to be 2023-06-11T00:00:00Z, got %s", construct.LastUpdated)
		}
		if construct.PulumiStack.String() != "123e4567-e89b-12d3-a456-426614174000" {
			t.Errorf("Expected PulumiStack to be 123e4567-e89b-12d3-a456-426614174000, got %s", construct.PulumiStack.String())
		}
		if construct.URN.String() != "urn:construct:example" {
			t.Errorf("Expected URN to be urn:construct:example, got %s", construct.URN.String())
		}
	}
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
			Status:      ConstructPending,
			LastUpdated: "2023-06-11T00:00:00Z",
			Inputs:      make(map[string]Input),
			Outputs:     make(map[string]string),
			Bindings:    []Binding{},
			Options:     make(map[string]interface{}),
			DependsOn:   []*URN{},
			PulumiStack: UUID{uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")},
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
		if construct.Status != ConstructPending {
			t.Errorf("Expected status to be %s, got %s", ConstructPending, construct.Status)
		}
		if construct.LastUpdated != "2023-06-11T00:00:00Z" {
			t.Errorf("Expected last updated to be 2023-06-11T00:00:00Z, got %s", construct.LastUpdated)
		}
		if construct.PulumiStack.String() != "123e4567-e89b-12d3-a456-426614174000" {
			t.Errorf("Expected PulumiStack to be 123e4567-e89b-12d3-a456-426614174000, got %s", construct.PulumiStack.String())
		}
		if construct.URN.String() != "urn:construct:example" {
			t.Errorf("Expected URN to be urn:construct:example, got %s", construct.URN.String())
		}
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
				Outputs:   make(map[string]string),
				Bindings:  []Binding{},
				Options:   make(map[string]interface{}),
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

	construct, exists := sm.GetConstruct("example-construct")
	if !exists {
		t.Fatalf("Expected construct example-construct to exist")
	}
	if construct.Status != ConstructPending {
		t.Errorf("Expected status to be %s, got %s", ConstructPending, construct.Status)
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
	URN, err := ParseURN("urn:accountid:my-project:dev:my-app:construct/klotho.aws.S3:example-construct")
	if err != nil {
		t.Fatalf("Failed to parse URN: %v", err)
	}
	construct := &ConstructState{Status: ConstructPending, URN: URN}
	if err := sm.TransitionConstructState(construct, ConstructCreatePending); err != nil {
		t.Errorf("Expected valid transition, got error: %v", err)
	}
	if construct.Status != ConstructCreatePending {
		t.Errorf("Expected status %s, got %s", ConstructCreatePending, construct.Status)
	}

	if err := sm.TransitionConstructState(construct, ConstructPending); err == nil {
		t.Errorf("Expected error for invalid transition, got nil")
	}
}
