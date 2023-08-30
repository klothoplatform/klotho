package input

import (
	"errors"

	"github.com/klothoplatform/klotho/pkg/construct"
)

// validateNodes provices common validation that either 'node' or ('source' and 'target') must be specified
func validateNodes(node, source, target construct.ResourceId) error {
	if node.IsZero() && source.IsZero() && target.IsZero() {
		return errors.New("one of 'node' or ('source' and 'target') must be specified")
	}
	if node.IsZero() {
		if source.IsZero() || target.IsZero() {
			return errors.New("both 'source' and 'target' must be specified")
		}
	} else if !source.IsZero() || !target.IsZero() {
		return errors.New("only one of 'node' or ('source' and 'target') must be specified")
	}
	return nil
}
