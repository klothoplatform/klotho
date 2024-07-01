package model

type ConstructState struct {
	Status      ConstructStatus  `yaml:"status,omitempty"`
	LastUpdated string           `yaml:"last_updated,omitempty"`
	Inputs      map[string]Input `yaml:"inputs,omitempty"`
	Outputs     map[string]any   `yaml:"outputs,omitempty"`
	Bindings    []Binding        `yaml:"bindings,omitempty"`
	Options     map[string]any   `yaml:"options,omitempty"`
	DependsOn   []*URN           `yaml:"dependsOn,omitempty"`
	PulumiStack UUID             `yaml:"pulumi_stack,omitempty"`
	URN         *URN             `yaml:"urn,omitempty"`
}

type ConstructStatus string

const (
	// Create-related statuses
	ConstructCreating       ConstructStatus = "creating"
	ConstructCreateComplete ConstructStatus = "create_complete"
	ConstructCreateFailed   ConstructStatus = "create_failed"

	// Update-related statuses
	ConstructUpdating       ConstructStatus = "updating"
	ConstructUpdateComplete ConstructStatus = "update_complete"
	ConstructUpdateFailed   ConstructStatus = "update_failed"

	// Delete-related statuses
	ConstructDeleting       ConstructStatus = "deleting"
	ConstructDeleteComplete ConstructStatus = "delete_complete"
	ConstructDeleteFailed   ConstructStatus = "delete_failed"

	// Unknown status
	ConstructUnknown ConstructStatus = "unknown"
)

var validTransitions = map[ConstructStatus][]ConstructStatus{
	ConstructCreating:       {ConstructCreateComplete, ConstructCreateFailed},
	ConstructCreateComplete: {ConstructUpdating, ConstructDeleting},
	ConstructCreateFailed:   {ConstructCreating, ConstructDeleting},
	ConstructUpdating:       {ConstructUpdateComplete, ConstructUpdateFailed},
	ConstructUpdateComplete: {ConstructUpdating, ConstructDeleting},
	ConstructUpdateFailed:   {ConstructUpdating, ConstructDeleting},
	ConstructDeleting:       {ConstructDeleteComplete, ConstructDeleteFailed},
	ConstructDeleteComplete: {ConstructCreating},
	ConstructDeleteFailed:   {ConstructUpdating, ConstructDeleting},
	ConstructUnknown:        {},
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
	validTransitions, exists := validTransitions[currentStatus]
	if !exists {
		return false
	}
	for _, validStatus := range validTransitions {
		if validStatus == nextStatus {
			return true
		}
	}
	return false
}

type (
	ConstructAction string
)

const (
	ConstructActionCreate ConstructAction = "create"
	ConstructActionUpdate ConstructAction = "update"
	ConstructActionDelete ConstructAction = "delete"
)
