package solution_context

import (
	"fmt"

	construct "github.com/klothoplatform/klotho/pkg/construct"
	engine_errs "github.com/klothoplatform/klotho/pkg/engine/errors"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledgebase"
)

type (
	KV struct {
		Key   string
		Value any
	}

	DecisionRecords interface {
		// AddRecord stores each decision (the what) with the context (the why) in some datastore
		AddRecord(context []KV, decision SolveDecision)

		GetRecords() []SolveDecision
	}

	SolveDecision interface {
		// internal is a private method to prevent other packages from implementing this interface.
		// It's not necessary, but it could prevent some accidental bad practices from emerging.
		internal()
	}

	AsEngineError interface {
		// TryEngineError returns an EngineError if the decision is an error, otherwise nil.
		TryEngineError() engine_errs.EngineError
	}

	AddResourceDecision struct {
		Resource construct.ResourceId
	}

	RemoveResourceDecision struct {
		Resource construct.ResourceId
	}

	AddDependencyDecision struct {
		From construct.ResourceId
		To   construct.ResourceId
	}

	RemoveDependencyDecision struct {
		From construct.ResourceId
		To   construct.ResourceId
	}

	SetPropertyDecision struct {
		Resource construct.ResourceId
		Property string
		Value    any
	}

	PropertyValidationDecision struct {
		Resource construct.ResourceId
		Property knowledgebase.Property
		Value    any
		Error    error
	}

	ConfigValidationError struct {
		PropertyValidationDecision
	}
)

func (d AddResourceDecision) internal()        {}
func (d AddDependencyDecision) internal()      {}
func (d RemoveResourceDecision) internal()     {}
func (d RemoveDependencyDecision) internal()   {}
func (d SetPropertyDecision) internal()        {}
func (d PropertyValidationDecision) internal() {}

func (d PropertyValidationDecision) TryEngineError() engine_errs.EngineError {
	if d.Error == nil {
		return nil
	}
	return ConfigValidationError{
		PropertyValidationDecision: d,
	}
}

func (e ConfigValidationError) Error() string {
	return fmt.Sprintf(
		"config validation error on %s#%s: %v",
		e.Resource,
		e.Property.Details().Path,
		e.PropertyValidationDecision.Error,
	)
}

func (e ConfigValidationError) ErrorCode() engine_errs.ErrorCode {
	return engine_errs.ConfigInvalidCode
}

func (e ConfigValidationError) ToJSONMap() map[string]any {
	return map[string]any{
		"resource":         e.Resource,
		"property":         e.Property.Details().Path,
		"value":            e.Value,
		"validation_error": e.PropertyValidationDecision.Error.Error(),
	}
}

func (e ConfigValidationError) Unwrap() error {
	return e.PropertyValidationDecision.Error
}
