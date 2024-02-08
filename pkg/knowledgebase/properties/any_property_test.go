package properties

import (
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledgebase"
	"github.com/stretchr/testify/assert"
)

func Test_SetAnyProperty(t *testing.T) {
	tests := []struct {
		name     string
		property *AnyProperty
		resource *construct.Resource
		value    any
	}{
		{
			name:     "any property",
			resource: &construct.Resource{Properties: make(map[string]any)},
			property: &AnyProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
			value: "test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			err := tt.property.SetProperty(tt.resource, tt.value)
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.value, tt.resource.Properties[tt.property.Path])
		})
	}
}

func Test_AppendAnyProperty(t *testing.T) {
	tests := []struct {
		name     string
		property *AnyProperty
		resource *construct.Resource
		value    any
		expect   any
	}{
		{
			name:     "any property",
			resource: &construct.Resource{Properties: make(map[string]any)},
			property: &AnyProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
			value:  "test",
			expect: "test",
		},
		{
			name: "existing property",
			resource: &construct.Resource{
				Properties: map[string]any{
					"test": map[string]any{
						"test": "test",
					},
				},
			},
			property: &AnyProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
			value: map[string]any{
				"test2": "test2",
			},
			expect: map[string]any{
				"test":  "test",
				"test2": "test2",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			err := tt.property.AppendProperty(tt.resource, tt.value)
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.expect, tt.resource.Properties[tt.property.Path])
		})
	}
}
