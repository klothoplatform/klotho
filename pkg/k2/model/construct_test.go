package model

import (
	"testing"
)

func TestIsDeployable(t *testing.T) {
	tests := []struct {
		status   ConstructStatus
		expected bool
	}{
		{ConstructCreating, false},
		{ConstructUpdating, false},
		{ConstructCreateComplete, true},
	}

	for _, test := range tests {
		t.Run(string(test.status), func(t *testing.T) {
			if result := IsDeployable(test.status); result != test.expected {
				t.Errorf("IsDeployable(%s) = %v; want %v", test.status, result, test.expected)
			}
		})
	}
}

func TestIsDeletable(t *testing.T) {
	tests := []struct {
		status   ConstructStatus
		expected bool
	}{
		{ConstructCreateComplete, true},
		{ConstructDeleting, false},
		{ConstructUpdateComplete, true},
	}

	for _, test := range tests {
		t.Run(string(test.status), func(t *testing.T) {
			if result := IsDeletable(test.status); result != test.expected {
				t.Errorf("IsDeletable(%s) = %v; want %v", test.status, result, test.expected)
			}
		})
	}
}

func TestIsValidTransition(t *testing.T) {
	tests := []struct {
		currentStatus ConstructStatus
		nextStatus    ConstructStatus
		expected      bool
	}{
		{ConstructCreating, ConstructCreateComplete, true},
		{ConstructCreating, ConstructCreateFailed, true},
		{ConstructCreateComplete, ConstructUpdating, true},
		{ConstructCreateFailed, ConstructCreating, true},
		{ConstructDeleteComplete, ConstructCreating, true},
		{ConstructStatus("fake"), ConstructCreating, false}, // Invalid current status"}
		{ConstructDeleteComplete, ConstructUpdating, false}, // Invalid transition
	}

	for _, test := range tests {
		t.Run(string(test.currentStatus)+" to "+string(test.nextStatus), func(t *testing.T) {
			if result := isValidTransition(test.currentStatus, test.nextStatus); result != test.expected {
				t.Errorf("isValidTransition(%s, %s) = %v; want %v", test.currentStatus, test.nextStatus, result, test.expected)
			}
		})
	}
}
