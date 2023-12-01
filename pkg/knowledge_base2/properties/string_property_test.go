package properties

import (
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/stretchr/testify/assert"
)

func Test_SetStringProperty(t *testing.T) {
	tests := []struct {
		name     string
		property *StringProperty
		resource *construct.Resource
		value    string
	}{
		{
			name:     "string property",
			resource: &construct.Resource{Properties: make(map[string]any)},
			property: &StringProperty{
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

func Test_AppendStringProperty(t *testing.T) {
	tests := []struct {
		name     string
		property *StringProperty
		resource *construct.Resource
		value    string
	}{
		{
			name:     "string property",
			resource: &construct.Resource{Properties: make(map[string]any)},
			property: &StringProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
			value: "test",
		},
		{
			name: "existing property",
			resource: &construct.Resource{
				Properties: map[string]any{
					"test": "test",
				},
			},
			property: &StringProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
			value: "test2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			err := tt.property.AppendProperty(tt.resource, tt.value)
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.value, tt.resource.Properties[tt.property.Path])
		})
	}
}

func Test_RemoveStringProperty(t *testing.T) {
	tests := []struct {
		name     string
		property *StringProperty
		resource *construct.Resource
		value    string
	}{
		{
			name:     "string property",
			resource: &construct.Resource{Properties: make(map[string]any)},
			property: &StringProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
			value: "test",
		},
		{
			name: "existing property",
			resource: &construct.Resource{
				Properties: map[string]any{
					"test": "test",
				},
			},
			property: &StringProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
			value: "test2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			err := tt.property.RemoveProperty(tt.resource, tt.value)
			if !assert.NoError(err) {
				return
			}
			assert.Empty(tt.resource.Properties)
		})
	}
}

func Test_StringDetails(t *testing.T) {
	tests := []struct {
		name     string
		property *StringProperty
	}{
		{
			name: "string property",
			property: &StringProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			details := tt.property.Details()
			assert.Equal(tt.property.Path, details.Path)
		})
	}
}

func Test_StringClone(t *testing.T) {
	tests := []struct {
		name     string
		property *StringProperty
	}{
		{
			name: "string property",
			property: &StringProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			clone := tt.property.Clone()
			assert.Equal(tt.property.Path, clone.Details().Path)
		})
	}
}

func Test_StringGetDefaultValue(t *testing.T) {
	tests := []struct {
		name     string
		property *StringProperty
		ctx      knowledgebase.DynamicValueContext
		data     knowledgebase.DynamicValueData
		value    any
	}{
		{
			name: "string property",
			property: &StringProperty{
				SharedPropertyFields: SharedPropertyFields{
					DefaultValue: "test",
				},
			},
			value: "test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			val, err := tt.property.GetDefaultValue(tt.ctx, tt.data)
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.value, val)
		})
	}
}
