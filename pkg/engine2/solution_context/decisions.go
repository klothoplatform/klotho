package solution_context

import (
	"encoding/json"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

type (
	KV struct {
		Key   string
		Value any
	}

	DecisionRecords interface {
		// AddRecord stores each decision (the what) with the context (the why) in some datastore
		AddRecord(context []KV, decision SolveDecision)
		// // FindDecision returns the context (the why) for a given decision (the what)
		// FindDecision(decision SolveDecision) []KV
		// // FindContext returns the various decisions (the what) for a given context (the why)
		// FindContext(key string, value any) []SolveDecision
		GetRecords() []SolveDecision
	}

	SolveDecision interface {
		// internal is a private method to prevent other packages from implementing this interface.
		// It's not necessary, but it could prevent some accidental bad practices from emerging.
		internal()
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
)

func (d AddResourceDecision) internal()        {}
func (d AddDependencyDecision) internal()      {}
func (d RemoveResourceDecision) internal()     {}
func (d RemoveDependencyDecision) internal()   {}
func (d SetPropertyDecision) internal()        {}
func (d PropertyValidationDecision) internal() {}

func (d PropertyValidationDecision) MarshalJSON() ([]byte, error) {
	if d.Value != nil {
		return json.Marshal(map[string]any{
			"resource": d.Resource,
			"property": d.Property.Details().Path,
			"value":    d.Value,
			"error":    d.Error,
		})
	}
	return json.Marshal(map[string]any{
		"resource": d.Resource,
		"property": d.Property.Details().Path,
		"error":    d.Error,
	})
}
