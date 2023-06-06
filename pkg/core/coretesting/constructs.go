package coretesting

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/stretchr/testify/assert"
)

func TryGetProvenance(assert *assert.Assertions, bc core.BaseConstruct) core.AnnotationKey {
	cons, ok := bc.(core.Construct)
	if !assert.True(ok, `%s is not a Construct`, cons.Id()) {
		return core.AnnotationKey{
			Capability: "FAIL",
			ID:         "FAIL",
		}
	}
	return cons.Provenance()
}
