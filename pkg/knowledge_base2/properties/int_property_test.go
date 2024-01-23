package properties

import (
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/enginetesting"
	knowledgebase2 "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_SetIntPropertyProperty(t *testing.T) {
	tests := []struct {
		name     string
		property *IntProperty
		resource *construct.Resource
		value    int
	}{
		{
			name:     "float property",
			resource: &construct.Resource{Properties: make(map[string]any)},
			property: &IntProperty{
				PropertyDetails: knowledgebase2.PropertyDetails{
					Path: "test",
				},
			},
			value: 1,
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

func Test_AppendIntProperty(t *testing.T) {
	tests := []struct {
		name     string
		property *IntProperty
		resource *construct.Resource
		value    int
	}{
		{
			name:     "float property",
			resource: &construct.Resource{Properties: make(map[string]any)},
			property: &IntProperty{
				PropertyDetails: knowledgebase2.PropertyDetails{
					Path: "test",
				},
			},
			value: 1.0,
		},
		{
			name: "existing property",
			resource: &construct.Resource{
				Properties: construct.Properties{
					"test": 1.0,
				},
			},
			property: &IntProperty{
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

func Test_RemoveIntProperty(t *testing.T) {
	tests := []struct {
		name     string
		property *IntProperty
		resource *construct.Resource
	}{
		{
			name: "int property",
			resource: &construct.Resource{
				Properties: construct.Properties{
					"test": 1.0,
				},
			},
			property: &IntProperty{
				PropertyDetails: knowledgebase2.PropertyDetails{
					Path: "test",
				},
			},
		},
		{
			name: "non existing property",
			resource: &construct.Resource{
				Properties: construct.Properties{
					"test": 1.0,
				},
			},
			property: &IntProperty{
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

func Test_IntProperty_Parse(t *testing.T) {
	tests := []struct {
		name     string
		property *IntProperty
		value    any
		expected int
		wantErr  bool
	}{
		{
			name: "int property",
			property: &IntProperty{
				PropertyDetails: knowledgebase2.PropertyDetails{
					Path: "test",
				},
			},
			value:    1,
			expected: 1,
		},
		{
			name: "template value",
			property: &IntProperty{
				PropertyDetails: knowledgebase2.PropertyDetails{
					Path: "test2",
				},
			},
			value:    "{{ 1 }}",
			expected: 1,
		},
		{
			name: "float32 property",
			property: &IntProperty{
				PropertyDetails: knowledgebase2.PropertyDetails{
					Path: "test",
				},
			},
			value:    float32(1.0),
			expected: 1,
		},
		{
			name: "float64 property",
			property: &IntProperty{
				PropertyDetails: knowledgebase2.PropertyDetails{
					Path: "test",
				},
			},
			value:    float64(1.0),
			expected: 1,
		},
		{
			name: "non int property",
			property: &IntProperty{
				PropertyDetails: knowledgebase2.PropertyDetails{
					Path: "test",
				},
			},
			value:   "test",
			wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			ctx := &knowledgebase2.DynamicValueContext{}
			data := knowledgebase2.DynamicValueData{}
			value, err := test.property.Parse(test.value, ctx, data)
			if test.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			assert.Equal(test.expected, value)
		})
	}
}

func Test_IntProperty_Validate(t *testing.T) {
	upperBound := 10
	lowerBound := 0
	tests := []struct {
		name          string
		property      *IntProperty
		testResources []*construct.Resource
		mockKBCalls   []mock.Call
		value         any
		wantErr       bool
	}{
		{
			name: "int property",
			property: &IntProperty{
				PropertyDetails: knowledgebase2.PropertyDetails{
					Path: "test",
				},
			},
			value: 1,
		},
		{
			name: "within bounds",
			property: &IntProperty{
				PropertyDetails: knowledgebase2.PropertyDetails{
					Path: "test",
				},
				MaxValue: &upperBound,
				MinValue: &lowerBound,
			},
			value: 5,
		},
		{
			name: "above upper bound",
			property: &IntProperty{
				PropertyDetails: knowledgebase2.PropertyDetails{
					Path: "test",
				},
				MaxValue: &upperBound,
				MinValue: &lowerBound,
			},
			value:   11,
			wantErr: true,
		},
		{
			name: "below lower bound",
			property: &IntProperty{
				PropertyDetails: knowledgebase2.PropertyDetails{
					Path: "test",
				},
				MaxValue: &upperBound,
				MinValue: &lowerBound,
			},
			value:   -1,
			wantErr: true,
		},
		{
			name: "non int property",
			property: &IntProperty{
				PropertyDetails: knowledgebase2.PropertyDetails{
					Path: "test",
				},
			},
			value:   "test",
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

func Test_IntProperty_Clone(t *testing.T) {
	assert := assert.New(t)
	property := &IntProperty{
		PropertyDetails: knowledgebase2.PropertyDetails{
			Path: "test",
		},
	}
	actual := property.Clone()
	assert.Equal(property, actual)
	assert.NotSame(property, actual)
}

func Test_IntProperty_Details(t *testing.T) {
	assert := assert.New(t)
	property := &IntProperty{
		PropertyDetails: knowledgebase2.PropertyDetails{
			Path: "test",
		},
	}
	actual := property.Details()
	assert.Equal(property.PropertyDetails, *actual)
	assert.Same(&property.PropertyDetails, actual)
}

func Test_IntProperty_Type(t *testing.T) {
	assert := assert.New(t)
	property := &IntProperty{}
	assert.Equal(property.Type(), "int")
}

func Test_IntProperty_ZeroValue(t *testing.T) {
	assert := assert.New(t)
	property := &IntProperty{}
	assert.Equal(property.ZeroValue(), 0)
}

func Test_IntProperty_GetDefaultValue(t *testing.T) {
	tests := []struct {
		name     string
		property *IntProperty
		expect   any
	}{
		{
			name: "default value",
			property: &IntProperty{
				SharedPropertyFields: SharedPropertyFields{
					DefaultValue: 1,
				},
			},
			expect: 1,
		},
		{
			name:     "no default value",
			property: &IntProperty{},
			expect:   nil,
		},
		{
			name: "default value with template",
			property: &IntProperty{
				SharedPropertyFields: SharedPropertyFields{
					DefaultValue: "{{ 1 }}",
				},
			},
			expect: 1,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			ctx := knowledgebase2.DynamicValueContext{}
			data := knowledgebase2.DynamicValueData{}
			actual, err := test.property.GetDefaultValue(ctx, data)
			if !assert.NoError(err) {
				return
			}
			assert.Equal(test.expect, actual)
		})
	}
}

func Test_IntProperty_Contains(t *testing.T) {
	assert := assert.New(t)
	property := &IntProperty{
		PropertyDetails: knowledgebase2.PropertyDetails{
			Path: "test",
		},
	}
	assert.False(property.Contains(1, 1))
}

func Test_IntProperty_SubProperties(t *testing.T) {
	assert := assert.New(t)
	property := &IntProperty{}
	assert.Nil(property.SubProperties())
}
