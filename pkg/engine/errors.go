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
		json.Marshaler
	}

	OperationalResourceError struct {
		Rule     knowledgebase.OperationalRule
		ToCreate construct.ResourceId
		Count    int
		Resource construct.Resource
		Parent   construct.Resource
		Cause    error
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

func causeString(cause error) string {
	if cause == nil {
		return ""
	}
	return cause.Error()
}

func (err *OperationalResourceError) MarshalJSON() ([]byte, error) {
	var parentId construct.ResourceId
	if err.Parent != nil {
		parentId = err.Parent.Id()
	}
	return json.Marshal(map[string]interface{}{
		"type":     fmt.Sprintf("%T", err),
		"rule":     err.Rule,
		"toCreate": err.ToCreate,
		"cause":    causeString(err.Cause),
		"parent":   parentId.String(),
	})
}

func (err *EdgeExpansionError) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":       fmt.Sprintf("%T", err),
		"constraint": err.Constraint,
		"cause":      causeString(err.Cause),
		"edge":       fmt.Sprintf("%s,%s", err.Edge.Source.Id(), err.Edge.Destination.Id()),
	})
}

func (err *EdgeConfigurationError) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":       fmt.Sprintf("%T", err),
		"constraint": err.Constraint,
		"cause":      causeString(err.Cause),
		"edge":       fmt.Sprintf("%s,%s", err.Edge.Source.Id(), err.Edge.Destination.Id()),
	})
}

func (err *ResourceNotOperationalError) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":       fmt.Sprintf("%T", err),
		"constraint": err.Constraint,
		"cause":      causeString(err.Cause),
		"resource":   err.Resource.Id().String(),
	})
}

func (err *ResourceConfigurationError) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":       fmt.Sprintf("%T", err),
		"constraint": err.Constraint,
		"cause":      causeString(err.Cause),
		"resource":   err.Resource.Id().String(),
	})
}

func (err *ConstructExpansionError) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":       fmt.Sprintf("%T", err),
		"constraint": err.Constraint,
		"cause":      causeString(err.Cause),
		"construct":  err.Construct.Id().String(),
	})
}

func (err *InternalError) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":  fmt.Sprintf("%T", err),
		"cause": causeString(err.Cause),
		"child": err.Child,
	})
}
