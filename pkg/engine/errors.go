package engine

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/construct"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
)

type (
	OperationalResourceError struct {
		Needs      []string
		Count      int
		Direction  knowledgebase.Direction
		Resource   construct.Resource
		Parent     construct.Resource
		MustCreate bool
		Cause      error
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
