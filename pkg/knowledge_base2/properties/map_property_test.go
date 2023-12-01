package properties

import (
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase2 "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/stretchr/testify/assert"
)

func Test_SetMapProperty(t *testing.T) {
	tests := []struct {
		name     string
		property *MapProperty
		resource *construct.Resource
		value    any
	}{
		{
			name: "Set map property value",
			property: &MapProperty{
				PropertyDetails: knowledgebase2.PropertyDetails{
					Path: "test",
				},
				KeyProperty:   &StringProperty{},
				ValueProperty: &StringProperty{},
			},
			resource: &construct.Resource{
				Properties: make(construct.Properties),
			},
			value: map[string]interface{}{
				"key":   "test",
				"value": "test",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			err := test.property.SetProperty(test.resource, test.value)
			if !assert.NoError(err) {
				return
			}
			assert.Equal(test.value, test.resource.Properties[test.property.Path])
		})
	}

}
