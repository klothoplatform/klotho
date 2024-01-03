package properties

import (
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/enginetesting"
	knowledgebase2 "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_SetFloatProperty(t *testing.T) {
	tests := []struct {
		name     string
		property *FloatProperty
		resource *construct.Resource
		value    float64
	}{
		{
			name:     "float property",
			resource: &construct.Resource{Properties: make(map[string]any)},
			property: &FloatProperty{
				PropertyDetails: knowledgebase2.PropertyDetails{
					Path: "test",
				},
			},
			value: 1.0,
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

func Test_AppendFloatProperty(t *testing.T) {
	tests := []struct {
		name     string
		property *FloatProperty
		resource *construct.Resource
		value    float64
	}{
		{
			name:     "float property",
			resource: &construct.Resource{Properties: make(map[string]any)},
			property: &FloatProperty{
				PropertyDetails: knowledgebase2.PropertyDetails{
					Path: "test",
				},
			},
			value: 1.0,
		},
		{
			name: "existing property",
			resource: &construct.Resource{
				Properties: map[string]any{
					"test": 1.0,
				},
			},
			property: &FloatProperty{
				PropertyDetails: knowledgebase2.PropertyDetails{
					Path: "test",
				},
			},
			value: 2.0,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			err := test.property.AppendProperty(test.resource, test.value)
			if !assert.NoError(err) {
				return
			}
			assert.Equal(test.value, test.resource.Properties[test.property.Path])
		})
	}
}

func Test_RemoveFloatProperty(t *testing.T) {
	tests := []struct {
		name     string
		property *FloatProperty
		resource *construct.Resource
	}{
		{
			name: "float property",
			resource: &construct.Resource{
				Properties: map[string]any{
					"test": 1.0,
				},
			},
			property: &FloatProperty{
				PropertyDetails: knowledgebase2.PropertyDetails{
					Path: "test",
				},
			},
		},
		{
			name: "nonexistent property",
			resource: &construct.Resource{
				Properties: map[string]any{
					"test": 1.0,
				},
			},
			property: &FloatProperty{
				PropertyDetails: knowledgebase2.PropertyDetails{
					Path: "test2",
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			err := test.property.RemoveProperty(test.resource, nil)
			if !assert.NoError(err) {
				return
			}
			assert.NotContains(test.resource.Properties, test.property.Path)
		})
	}
}

func Test_ParseFloatValue(t *testing.T) {
	tests := []struct {
		name     string
		property *FloatProperty
		value    any
		expect   any
		wantErr  bool
	}{
		{
			name: "float value",
			property: &FloatProperty{
				PropertyDetails: knowledgebase2.PropertyDetails{
					Path: "test",
				},
			},
			value:  1.0,
			expect: 1.0,
		},
		{
			name: "template value",
			property: &FloatProperty{
				PropertyDetails: knowledgebase2.PropertyDetails{
					Path: "test",
				},
			},
			value:  "{{ 1.0 }}",
			expect: float32(1.0),
		},
		{
			name: "int value",
			property: &FloatProperty{
				PropertyDetails: knowledgebase2.PropertyDetails{
					Path: "test",
				},
			},
			value:  1,
			expect: 1.0,
		},
		{
			name: "invalid value",
			property: &FloatProperty{
				PropertyDetails: knowledgebase2.PropertyDetails{
					Path: "test",
				},
			},
			value:   "sdf",
			wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			ctx := knowledgebase2.DynamicValueContext{}
			data := knowledgebase2.DynamicValueData{}
			value, err := test.property.Parse(test.value, ctx, data)
			if test.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			assert.Equal(test.expect, value)
		})
	}
}

func Test_FloatProperty_Validate(t *testing.T) {
	tests := []struct {
		name          string
		property      *FloatProperty
		testResources []*construct.Resource
		mockKBCalls   []mock.Call
		value         any
		wantErr       bool
	}{
		{
			name: "float value",
			property: &FloatProperty{
				PropertyDetails: knowledgebase2.PropertyDetails{
					Path: "test",
				},
			},
			value: 1.0,
		},
		{
			name: "int value",
			property: &FloatProperty{
				PropertyDetails: knowledgebase2.PropertyDetails{
					Path: "test",
				},
			},
			value:   1,
			wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			resource := &construct.Resource{}
			graph := construct.NewGraph()
			for _, r := range test.testResources {
				graph.AddVertex(r)
			}
			mockKB := &enginetesting.MockKB{}
			for _, call := range test.mockKBCalls {
				mockKB.On(call.Method, call.Arguments...).Return(call.ReturnArguments...)
			}
			ctx := knowledgebase2.DynamicValueContext{
				Graph:         graph,
				KnowledgeBase: mockKB,
			}
			err := test.property.Validate(resource, test.value, ctx)
			if test.wantErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)
		})
	}
}

func Test_FloatProperty_GetDefaultValue(t *testing.T) {
	tests := []struct {
		name     string
		property *FloatProperty
		expect   any
	}{
		{
			name: "default value",
			property: &FloatProperty{
				SharedPropertyFields: SharedPropertyFields{
					DefaultValue: 1.0,
				},
			},
			expect: 1.0,
		},
		{
			name: "no default value",
			property: &FloatProperty{
				PropertyDetails: knowledgebase2.PropertyDetails{
					Path: "test",
				},
			},
			expect: nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			ctx := knowledgebase2.DynamicValueContext{}
			data := knowledgebase2.DynamicValueData{}
			value, err := test.property.GetDefaultValue(ctx, data)
			if !assert.NoError(err) {
				return
			}
			assert.Equal(test.expect, value)
		})
	}
}

func Test_FloatPRoperty_Clone(t *testing.T) {
	assert := assert.New(t)
	property := &FloatProperty{
		PropertyDetails: knowledgebase2.PropertyDetails{
			Path: "test",
		},
	}
	clone := property.Clone()
	assert.Equal(property.Path, clone.Details().Path)
	assert.NotSame(property, clone)
}

func Test_FloatProperty_Details(t *testing.T) {
	assert := assert.New(t)
	property := &FloatProperty{
		PropertyDetails: knowledgebase2.PropertyDetails{
			Path: "test",
		},
	}
	details := property.Details()
	assert.Equal(property.Path, details.Path)
}

func Test_FloatProperty_ZeroValue(t *testing.T) {
	assert := assert.New(t)
	property := &FloatProperty{
		PropertyDetails: knowledgebase2.PropertyDetails{
			Path: "test",
		},
	}
	assert.Equal(float64(0.0), property.ZeroValue())
}

func Test_FloatProperty_Contains(t *testing.T) {
	assert := assert.New(t)
	property := &FloatProperty{
		PropertyDetails: knowledgebase2.PropertyDetails{
			Path: "test",
		},
	}
	assert.False(property.Contains(1.0, 1.0))
}

func Test_FloatProperty_Type(t *testing.T) {
	assert := assert.New(t)
	property := &FloatProperty{
		PropertyDetails: knowledgebase2.PropertyDetails{
			Path: "test",
		},
	}
	assert.Equal("float", property.Type())
}

func Test_FloatProperty_SubProperties(t *testing.T) {
	assert := assert.New(t)
	property := &FloatProperty{
		PropertyDetails: knowledgebase2.PropertyDetails{
			Path: "test",
		},
	}
	assert.Empty(property.SubProperties())
}
