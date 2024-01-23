package properties

import (
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/enginetesting"
	knowledgebase2 "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

func Test_AppendMapProperty(t *testing.T) {
	tests := []struct {
		name     string
		property *MapProperty
		resource *construct.Resource
		value    any
		expected any
	}{
		{
			name: "Append map property value",
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
			expected: map[string]interface{}{
				"key":   "test",
				"value": "test",
			},
		},
		{
			name: "Append existing map property value",
			property: &MapProperty{
				PropertyDetails: knowledgebase2.PropertyDetails{
					Path: "test",
				},
				KeyProperty:   &StringProperty{},
				ValueProperty: &StringProperty{},
			},
			resource: &construct.Resource{
				Properties: map[string]any{
					"test": map[string]interface{}{
						"key":   "test",
						"value": "test",
					},
				},
			},
			value: map[string]interface{}{
				"key2":   "test",
				"value2": "test",
			},
			expected: map[string]interface{}{
				"key":    "test",
				"value":  "test",
				"key2":   "test",
				"value2": "test",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			err := test.property.AppendProperty(test.resource, test.value)
			if !assert.NoError(err) {
				return
			}
			assert.Equal(test.expected, test.resource.Properties[test.property.Path])
		})
	}
}

func Test_RemoveMapProperty(t *testing.T) {
	tests := []struct {
		name     string
		property *MapProperty
		resource *construct.Resource
		value    any
		expected any
	}{
		{
			name: "Remove map property value",
			property: &MapProperty{
				PropertyDetails: knowledgebase2.PropertyDetails{
					Path: "test",
				},
				KeyProperty:   &StringProperty{},
				ValueProperty: &StringProperty{},
			},
			resource: &construct.Resource{
				Properties: map[string]any{
					"test": map[string]interface{}{
						"key":   "test",
						"value": "test",
					},
				},
			},
			value: map[string]interface{}{
				"key":   "test",
				"value": "test",
			},
			expected: map[string]interface{}{},
		},
		{
			name: "Remove portion of value",
			property: &MapProperty{
				PropertyDetails: knowledgebase2.PropertyDetails{
					Path: "test",
				},
				KeyProperty:   &StringProperty{},
				ValueProperty: &StringProperty{},
			},
			resource: &construct.Resource{
				Properties: map[string]any{
					"test": map[string]interface{}{
						"key":    "test",
						"value":  "test",
						"key2":   "test",
						"value2": "test",
					},
				},
			},
			value: map[string]interface{}{
				"key2":   "test",
				"value2": "test",
			},
			expected: map[string]interface{}{
				"key":   "test",
				"value": "test",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			err := test.property.RemoveProperty(test.resource, test.value)
			if !assert.NoError(err) {
				return
			}
			assert.Equal(test.expected, test.resource.Properties[test.property.Path])
		})
	}
}

func Test_MapProperty_Details(t *testing.T) {
	assert := assert.New(t)
	property := &MapProperty{
		PropertyDetails: knowledgebase2.PropertyDetails{
			Path: "test",
		},
	}
	details := property.Details()
	assert.Same(&property.PropertyDetails, details)
}

func Test_MapProperty_Clone(t *testing.T) {
	assert := assert.New(t)
	property := &MapProperty{
		PropertyDetails: knowledgebase2.PropertyDetails{
			Path: "test",
		},
	}
	clone := property.Clone()
	assert.Equal(property, clone)
	assert.NotSame(property, clone)
}

func Test_MapProperty_Type(t *testing.T) {
	assert := assert.New(t)
	property := &MapProperty{}
	assert.Equal("map", property.Type())
	property2 := &MapProperty{KeyProperty: &StringProperty{}, ValueProperty: &StringProperty{}}
	assert.Equal("map(string,string)", property2.Type())
}

func Test_MapProperty_Validate(t *testing.T) {
	tests := []struct {
		name          string
		property      *MapProperty
		testResources []*construct.Resource
		mockKBCalls   []mock.Call
		value         any
		wantErr       bool
	}{
		{
			name: "valid map property",
			property: &MapProperty{
				KeyProperty:   &StringProperty{},
				ValueProperty: &StringProperty{},
			},
			value: map[string]interface{}{
				"key":   "test",
				"value": "test",
			},
		},
		{
			name: "tests key validity",
			property: &MapProperty{
				KeyProperty:   &StringProperty{AllowedValues: []string{"test"}},
				ValueProperty: &StringProperty{},
			},
			value: map[string]interface{}{
				"key": "test",
			},
			wantErr: true,
		},
		{
			name: "tests val validity",
			property: &MapProperty{
				KeyProperty:   &StringProperty{},
				ValueProperty: &StringProperty{AllowedValues: []string{"test"}},
			},
			value: map[string]interface{}{
				"key": "not-test",
			},
			wantErr: true,
		},
		{
			name: "invalid map property",
			property: &MapProperty{
				KeyProperty:   &StringProperty{},
				ValueProperty: &StringProperty{},
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

func Test_MapProperty_GetDefaultValue(t *testing.T) {
	tests := []struct {
		name     string
		property *MapProperty
		want     any
	}{
		{
			name: "returns empty map",
			property: &MapProperty{
				KeyProperty:   &StringProperty{},
				ValueProperty: &StringProperty{},
			},
		},
		{
			name: "returns default map",
			property: &MapProperty{
				KeyProperty:   &StringProperty{},
				ValueProperty: &StringProperty{},
				SharedPropertyFields: SharedPropertyFields{
					DefaultValue: map[string]interface{}{
						"key":   "test",
						"value": "test",
					},
				},
			},
			want: map[string]interface{}{
				"key":   "test",
				"value": "test",
			},
		},
		{
			name: "returns default map with templates",
			property: &MapProperty{
				KeyProperty:   &StringProperty{},
				ValueProperty: &StringProperty{},
				SharedPropertyFields: SharedPropertyFields{
					DefaultValue: map[string]interface{}{
						"{{ \"key\" }}":   "{{ \"Name\" }}",
						"{{ \"value\" }}": "{{ \"Name\" }}",
					},
				},
			},
			want: map[string]interface{}{
				"key":   "Name",
				"value": "Name",
			},
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
			assert.Equal(test.want, actual)
		})
	}
}

func Test_MapProperty_Parse(t *testing.T) {
	tests := []struct {
		name     string
		property *MapProperty
		value    any
		want     any
		wantErr  bool
	}{
		{
			name: "parses map property",
			property: &MapProperty{
				KeyProperty:   &StringProperty{},
				ValueProperty: &StringProperty{},
			},
			value: map[string]interface{}{
				"key":   "test",
				"value": "test",
			},
			want: map[string]interface{}{
				"key":   "test",
				"value": "test",
			},
		},
		{
			name: "parses map property with templates",
			property: &MapProperty{
				KeyProperty:   &StringProperty{},
				ValueProperty: &StringProperty{},
			},
			value: map[string]interface{}{
				"{{ \"key\" }}":   "{{ \"Name\" }}",
				"{{ \"value\" }}": "{{ \"Name\" }}",
			},
			want: map[string]interface{}{
				"key":   "Name",
				"value": "Name",
			},
		},
		{
			name: "parses sub properties",
			property: &MapProperty{
				Properties: knowledgebase2.Properties{
					"key":   &StringProperty{},
					"value": &StringProperty{},
				},
			},
			value: map[string]interface{}{
				"key":   "test",
				"value": "test",
			},
			want: map[string]interface{}{
				"key":   "test",
				"value": "test",
			},
		},
		{
			name: "parses sub properties with default values if they dont exist",
			property: &MapProperty{
				Properties: knowledgebase2.Properties{
					"key": &StringProperty{},
					"value": &StringProperty{
						SharedPropertyFields: SharedPropertyFields{
							DefaultValue: "test",
						},
					},
				},
			},
			value: map[string]interface{}{
				"key": "test",
			},
			want: map[string]interface{}{
				"key":   "test",
				"value": "test",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			ctx := knowledgebase2.DynamicValueContext{}
			data := knowledgebase2.DynamicValueData{}
			actual, err := test.property.Parse(test.value, ctx, data)
			if test.wantErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)
			assert.Equal(test.want, actual)
		})
	}
}

func Test_MapProperty_Contains(t *testing.T) {
	tests := []struct {
		name     string
		property *MapProperty
		value    any
		contains any
		want     bool
	}{
		{
			name: "contains value",
			property: &MapProperty{
				KeyProperty:   &StringProperty{},
				ValueProperty: &StringProperty{},
			},
			value: map[string]interface{}{
				"key":   "test",
				"value": "test",
			},
			contains: map[string]interface{}{
				"key": "test",
			},
			want: true,
		},
		{
			name: "does not contain value",
			property: &MapProperty{
				KeyProperty:   &StringProperty{},
				ValueProperty: &StringProperty{},
			},
			value: map[string]interface{}{
				"key":   "test",
				"value": "test",
			},
			contains: map[string]interface{}{
				"key": "not-test",
			},
			want: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			actual := test.property.Contains(test.value, test.contains)
			assert.Equal(test.want, actual)
		})
	}
}

func Test_MapProperty_ZeroValue(t *testing.T) {
	assert := assert.New(t)
	property := &MapProperty{}
	assert.Nil(property.ZeroValue())
}

func Test_MapProperty_SubProperties(t *testing.T) {
	assert := assert.New(t)
	property := &MapProperty{
		Properties: knowledgebase2.Properties{},
	}
	assert.NotNil(property.SubProperties())
}
