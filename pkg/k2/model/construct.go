package model

import (
	"fmt"
	"time"
)

type ConstructState struct {
	Status      ConstructStatus        `yaml:"status,omitempty"`
	LastUpdated string                 `yaml:"last_updated,omitempty"`
	Inputs      map[string]Input       `yaml:"inputs,omitempty"`
	Outputs     map[string]string      `yaml:"outputs,omitempty"`
	Bindings    []Binding              `yaml:"bindings,omitempty"`
	Options     map[string]interface{} `yaml:"options,omitempty"`
	DependsOn   []*URN                 `yaml:"dependsOn,omitempty"`
	PulumiStack UUID                   `yaml:"pulumi_stack,omitempty"`
	URN         *URN                   `yaml:"urn,omitempty"`
}

type ConstructStatus string

const (
	// Create-related statuses
	ConstructCreating       ConstructStatus = "creating"
	ConstructCreateComplete ConstructStatus = "create_complete"
	ConstructCreateFailed   ConstructStatus = "create_failed"
	ConstructCreatePending  ConstructStatus = "create_pending"

	// Update-related statuses
	ConstructUpdating       ConstructStatus = "updating"
	ConstructUpdateComplete ConstructStatus = "update_complete"
	ConstructUpdateFailed   ConstructStatus = "update_failed"
	ConstructUpdatePending  ConstructStatus = "update_pending"

	// Delete-related statuses
	ConstructDeleting       ConstructStatus = "deleting"
	ConstructDeleteComplete ConstructStatus = "delete_complete"
	ConstructDeleteFailed   ConstructStatus = "delete_failed"
	ConstructDeletePending  ConstructStatus = "delete_pending"

	// Pending status
	ConstructPending ConstructStatus = "pending"

	// Operational statuses
	ConstructOperational ConstructStatus = "operational"
	ConstructInoperative ConstructStatus = "inoperative"
	ConstructDeleted     ConstructStatus = "deleted"
	ConstructNoChange    ConstructStatus = "no_change"

	// Unknown status
	ConstructUnknown ConstructStatus = "unknown"
)

var validTransitions = map[ConstructStatus][]ConstructStatus{
	ConstructPending:        {ConstructCreatePending, ConstructUpdatePending, ConstructDeletePending},
	ConstructCreatePending:  {ConstructCreating, ConstructDeletePending},
	ConstructCreating:       {ConstructCreateComplete, ConstructCreateFailed},
	ConstructCreateComplete: {ConstructUpdating, ConstructDeleting},
	ConstructCreateFailed:   {ConstructPending, ConstructDeletePending},
	ConstructUpdating:       {ConstructUpdateComplete, ConstructUpdateFailed},
	ConstructUpdateComplete: {ConstructOperational},
	ConstructUpdateFailed:   {ConstructUpdatePending, ConstructDeletePending},
	ConstructDeleting:       {ConstructDeleteComplete, ConstructDeleteFailed},
	ConstructDeleteComplete: {ConstructDeleted},
	ConstructDeleteFailed:   {ConstructDeletePending},
	ConstructUpdatePending:  {ConstructUpdating},
	ConstructDeletePending:  {ConstructDeleting},
	ConstructOperational:    {ConstructUpdating, ConstructDeleting, ConstructInoperative},
	ConstructInoperative:    {ConstructOperational, ConstructUpdating, ConstructDeleting},
	ConstructUnknown:        {ConstructPending, ConstructCreating, ConstructUpdatePending, ConstructDeletePending},
}

func IsDeployable(status ConstructStatus) bool {
	for _, nextStatus := range validTransitions[status] {
		if nextStatus == ConstructCreating || nextStatus == ConstructUpdating {
			return true
		}
	}
	return false
}

func IsDeletable(status ConstructStatus) bool {
	for _, nextStatus := range validTransitions[status] {
		if nextStatus == ConstructDeleting {
			return true
		}
	}
	return false
}

func isValidTransition(currentStatus, nextStatus ConstructStatus) bool {
	for _, validStatus := range validTransitions[currentStatus] {
		if validStatus == nextStatus {
			return true
		}
	}
	return false
}

func TransitionConstructState(construct *ConstructState, nextStatus ConstructStatus) error {
	if isValidTransition(construct.Status, nextStatus) {
		construct.Status = nextStatus
		construct.LastUpdated = time.Now().Format(time.RFC3339)
		return nil
	}
	return fmt.Errorf("invalid state transition from %s to %s", construct.Status, nextStatus)
}

type (
	ConstructActionType string
)

const (
	ConstructActionCreate ConstructActionType = "create"
	ConstructActionUpdate ConstructActionType = "update"
	ConstructActionDelete ConstructActionType = "delete"
)
