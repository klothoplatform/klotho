package properties

import (
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

func Test_SetSetProperty(t *testing.T) {
	tests := []struct {
		name     string
		property *SetProperty
		resource *construct.Resource
		value    string
	}{
		{
			name: "set property",
			property: &SetProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
			resource: &construct.Resource{Properties: construct.Properties{}},
			value:    "test",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.property.SetProperty(test.resource, test.value)
			if err != nil {
				t.Errorf("Expected no error, but got: %v", err)
			}
			if test.resource.Properties[test.property.Path] == nil {
				t.Errorf("Expected property %s to be set", test.property.Path)
			}
		})
	}
}
