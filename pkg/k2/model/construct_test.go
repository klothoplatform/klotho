package model

import (
	"testing"
)

func TestUpdateResourceState(t *testing.T) {
	tmpFile := createTempStateFile(t, "")
	defer removeTempFile(t, tmpFile)

	sm := NewStateManager(tmpFile)

	parsedURN, err := ParseURN("urn:accountid:my-project:dev:my-app:construct/klotho.aws.S3:example-construct")
	if err != nil {
		t.Fatalf("Failed to parse URN: %v", err)
	}
	// Initialize the construct state in the StateManager
	sm.SetConstruct(ConstructState{
		Status: ConstructPending,
		URN:    parsedURN,
	})

	if err := sm.UpdateResourceState("example-construct", ConstructCreatePending, "2023-06-11T00:00:00Z"); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	construct, exists := sm.GetConstruct("example-construct")
	if !exists {
		t.Errorf("Expected construct example-construct to exist")
	}
	if construct.Status != ConstructCreatePending {
		t.Errorf("Expected status to be %s, got %s", ConstructCreatePending, construct.Status)
	}
	if construct.LastUpdated != "2023-06-11T00:00:00Z" {
		t.Errorf("Expected last updated to be 2023-06-11T00:00:00Z, got %s", construct.LastUpdated)
	}
}
