package properties

import (
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/stretchr/testify/assert"
)

func Test_SetBoolProperty(t *testing.T) {
	tests := []struct {
		name     string
		property *BoolProperty
		resource *construct.Resource
		value    bool
	}{
		{
			name:     "bool property",
			resource: &construct.Resource{Properties: make(map[string]any)},
			property: &BoolProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
			value: true,
		},
		{
			name:     "existing property",
			resource: &construct.Resource{Properties: map[string]any{"test": false}},
			property: &BoolProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
			value: true,
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

func Test_AppendBoolProperty(t *testing.T) {
	tests := []struct {
		name     string
		property *BoolProperty
		resource *construct.Resource
		value    bool
	}{
		{
			name:     "bool property",
			resource: &construct.Resource{Properties: make(map[string]any)},
			property: &BoolProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
			value: true,
		},
		{
			name:     "existing property",
			resource: &construct.Resource{Properties: map[string]any{"test": false}},
			property: &BoolProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
			value: true,
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

func Test_RemoveBoolProperty(t *testing.T) {
	tests := []struct {
		name     string
		property *BoolProperty
		resource *construct.Resource
	}{
		{
			name:     "bool property",
			resource: &construct.Resource{Properties: make(map[string]any)},
			property: &BoolProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
		},
		{
			name:     "existing property",
			resource: &construct.Resource{Properties: map[string]any{"test": false}},
			property: &BoolProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			err := tt.property.RemoveProperty(tt.resource, nil)
			if !assert.NoError(err) {
				return
			}
			assert.NotContains(tt.resource.Properties, tt.property.Path)
		})
	}
}

func Test_GetDefaultBoolValue(t *testing.T) {
	tests := []struct {
		name     string
		property *BoolProperty
		value    any
	}{
		{
			name:     "bool property",
			property: &BoolProperty{},
		},
		{
			name: "existing default",
			property: &BoolProperty{
				SharedPropertyFields: SharedPropertyFields{
					DefaultValue: true,
				},
			},
			value: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			ctx := knowledgebase.DynamicValueContext{}
			data := knowledgebase.DynamicValueData{}
			result, err := tt.property.GetDefaultValue(ctx, data)
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.value, result)
		})
	}
}

func Test_CloneBoolProperty(t *testing.T) {
	tests := []struct {
		name     string
		property *BoolProperty
	}{
		{
			name:     "bool property",
			property: &BoolProperty{},
		},
		{
			name: "existing default",
			property: &BoolProperty{
				SharedPropertyFields: SharedPropertyFields{
					DefaultValue: true,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			result := tt.property.Clone()
			assert.Equal(tt.property, result)
			assert.NotSame(tt.property, result)
		})
	}
}

func Test_BoolPropertyParse(t *testing.T) {
	tests := []struct {
		name     string
		property *BoolProperty
		value    any
		ctx      knowledgebase.DynamicValueContext
		data     knowledgebase.DynamicValueData
		want     any
		wantErr  bool
	}{
		{
			name: "bool property",
			property: &BoolProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
			value:   true,
			want:    true,
			wantErr: false,
		},
		{
			name: "template value",
			property: &BoolProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
			value:   "{{ true }}",
			want:    true,
			wantErr: false,
		},
		{
			name: "invalid value",
			property: &BoolProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
			value:   1,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			result, err := tt.property.Parse(tt.value, tt.ctx, tt.data)
			if !assert.Equal(tt.wantErr, err != nil) {
				return
			}
			assert.Equal(tt.want, result)
		})
	}
}

func Test_BoolPropertyZeroValue(t *testing.T) {
	assert := assert.New(t)
	property := &BoolProperty{}
	assert.Equal(property.ZeroValue(), false)
}

func Test_BoolPropertyDetails(t *testing.T) {
	assert := assert.New(t)
	property := &BoolProperty{}
	assert.Same(property.Details(), &property.PropertyDetails)
}

func Test_BoolPropertyContains(t *testing.T) {
	assert := assert.New(t)
	property := &BoolProperty{}
	assert.False(property.Contains(nil, nil))
}

func Test_BoolPropertyType(t *testing.T) {
	assert := assert.New(t)
	property := &BoolProperty{}
	assert.Equal(property.Type(), "bool")
}

func Test_BoolPropertyValidate(t *testing.T) {
	tests := []struct {
		name     string
		property *BoolProperty
		value    any
		wantErr  bool
	}{
		{
			name: "bool property",
			property: &BoolProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
			value:   true,
			wantErr: false,
		},
		{
			name: "invalid value",
			property: &BoolProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
			value:   1,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			resource := &construct.Resource{}
			err := tt.property.Validate(resource, tt.value)
			assert.Equal(tt.wantErr, err != nil)
		})
	}
}

func Test_BoolPropertySubProperties(t *testing.T) {
	assert := assert.New(t)
	property := &BoolProperty{}
	assert.Nil(property.SubProperties())
}
