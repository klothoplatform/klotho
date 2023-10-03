package solution_context

import (
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

type (
	NodeOperationalError struct {
		// Node is the node that the error is associated with
		Node construct.ResourceId
		// Properties are the properties that were being set when the error occurred
		Properties []knowledgebase.Property
		// Error is the error that occurred
		Cause error
	}
)

func (e NodeOperationalError) Error() string {
	return e.Cause.Error()
}
