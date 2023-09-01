package engine

import (
	"encoding/json"
	"fmt"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	"github.com/klothoplatform/klotho/pkg/graph"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
)

type (
	EngineError interface {
		error
		Type() string
		json.Marshaler
	}

	OperationalResourceError struct {
		Needs      []string
		Count      int
		Direction  knowledgebase.Direction
		Resource   construct.Resource
		Parent     construct.Resource
		MustCreate bool
		Cause      error
	}

	EdgeExpansionError struct {
		Edge       graph.Edge[construct.Resource]
		Constraint *constraints.EdgeConstraint
		Cause      error
	}

	EdgeConfigurationError struct {
		Edge       graph.Edge[construct.Resource]
		Config     knowledgebase.Configuration
		Cause      error
		Constraint constraints.Constraint
	}

	ResourceNotOperationalError struct {
		Resource                 construct.Resource
		Cause                    error
		OperationalResourceError OperationalResourceError
		Constraint               constraints.Constraint
	}

	ResourceConfigurationError struct {
		Resource   construct.Resource
		Config     knowledgebase.Configuration
		Constraint constraints.Constraint
		Cause      error
	}

	ConstructExpansionError struct {
		Construct  construct.BaseConstruct
		Constraint *constraints.ConstructConstraint
		Cause      error
	}

	InternalError struct {
		Child EngineError
		Cause error
	}
)

func NewOperationalResourceError(resource construct.Resource, needs []string, cause error) *OperationalResourceError {
	return &OperationalResourceError{
		Resource: resource,
		Needs:    needs,
		Cause:    cause,
		Count:    1,
	}
}

func (err *OperationalResourceError) Error() string {
	return fmt.Sprintf("error in making resource %s operational: %v", err.Resource.Id(), err.Cause)
}

func (err *OperationalResourceError) Format(s fmt.State, verb rune) {
	if formatter, ok := err.Cause.(fmt.Formatter); ok {
		formatter.Format(s, verb)
	} else {
		fmt.Fprint(s, err.Error())
	}
}

func (err *OperationalResourceError) Unwrap() error {
	return err.Cause
}

func (err *EdgeExpansionError) Error() string {
	return fmt.Sprintf("error in expanding edge %s-> %s: %v", err.Edge.Source.Id(), err.Edge.Destination.Id(), err.Cause)
}

func (err *EdgeConfigurationError) Error() string {
	return fmt.Sprintf("error in configuring edge %s -> %s: %v", err.Edge.Source.Id(), err.Edge.Destination.Id(), err.Cause)
}

func (err *ResourceNotOperationalError) Error() string {
	return fmt.Sprintf("resource %s is not operational: %v", err.Resource.Id(), err.Cause)
}

func (err *ResourceConfigurationError) Error() string {
	return fmt.Sprintf("error in configuring resource %s: %v", err.Resource.Id(), err.Cause)
}

func (err *ConstructExpansionError) Error() string {
	return fmt.Sprintf("error in expanding construct %s: %v", err.Construct.Id(), err.Cause)
}

func (err *InternalError) Error() string {
	return fmt.Sprintf("internal error: %v", err.Cause)
}

func (err *OperationalResourceError) Type() string {
	return "OperationalResourceError"
}

func (err *EdgeExpansionError) Type() string {
	return "EdgeExpansionError"
}

func (err *EdgeConfigurationError) Type() string {
	return "EdgeConfigurationError"
}

func (err *ResourceNotOperationalError) Type() string {
	return "ResourceNotOperationalError"
}

func (err *ResourceConfigurationError) Type() string {
	return "ResourceConfigurationError"
}

func (err *ConstructExpansionError) Type() string {
	return "ConstructExpansionError"
}

func (err *InternalError) Type() string {
	return "InternalError"
}

func (err *OperationalResourceError) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":    err.Type(),
		"needs":   err.Needs,
		"count":   err.Count,
		"cause":   err.Cause.Error(),
		"parent":  err.Parent.Id().String(),
		"created": err.MustCreate,
	})
}

func (err *EdgeExpansionError) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":       err.Type(),
		"constraint": err.Constraint,
		"cause":      err.Cause.Error(),
		"edge":       fmt.Sprintf("%s,%s", err.Edge.Source.Id(), err.Edge.Destination.Id()),
	})
}

func (err *EdgeConfigurationError) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":       err.Type(),
		"constraint": err.Constraint,
		"cause":      err.Cause.Error(),
		"edge":       fmt.Sprintf("%s,%s", err.Edge.Source.Id(), err.Edge.Destination.Id()),
	})
}

func (err *ResourceNotOperationalError) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":       err.Type(),
		"constraint": err.Constraint,
		"cause":      err.Cause.Error(),
		"resource":   err.Resource.Id().String(),
	})
}

func (err *ResourceConfigurationError) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":       err.Type(),
		"constraint": err.Constraint,
		"cause":      err.Cause.Error(),
		"resource":   err.Resource.Id().String(),
	})
}

func (err *ConstructExpansionError) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":       err.Type(),
		"constraint": err.Constraint,
		"cause":      err.Cause.Error(),
		"construct":  err.Construct.Id().String(),
	})
}

func (err *InternalError) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":  err.Type(),
		"cause": err.Cause.Error(),
		"child": err.Child,
	})
}
