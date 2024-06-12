package model

import (
	"testing"
)

func TestUpdateResourceState(t *testing.T) {
	tmpFile := createTempStateFile(t, "")
	defer removeTempFile(t, tmpFile)

	sm := NewStateManager(tmpFile)
	sm.UpdateResourceState("example-construct", ConstructCreating, "2023-06-11T00:00:00Z")
	construct, exists := sm.GetConstruct("example-construct")
	if !exists {
		t.Errorf("Expected construct example-construct to exist")
	}
	if construct.Status != ConstructCreating {
		t.Errorf("Expected status to be %s, got %s", ConstructCreating, construct.Status)
	}
	if construct.LastUpdated != "2023-06-11T00:00:00Z" {
		t.Errorf("Expected last updated to be 2023-06-11T00:00:00Z, got %s", construct.LastUpdated)
	}
}

func TestTransitionConstructState(t *testing.T) {
	construct := &ConstructState{Status: ConstructNew}
	if err := TransitionConstructState(construct, ConstructPending); err != nil {
		t.Errorf("Expected valid transition, got error: %v", err)
	}
	if construct.Status != ConstructPending {
		t.Errorf("Expected status %s, got %s", ConstructPending, construct.Status)
	}

	if err := TransitionConstructState(construct, ConstructCreateComplete); err == nil {
		t.Errorf("Expected error for invalid transition, got nil")
	}
}
