package knowledgebase2

import (
	"fmt"
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/stretchr/testify/assert"
)

func Test_getPropertyType(t *testing.T) {
	tests := []struct {
		name     string
		property Property
		expected PropertyType
	}{
		{
			name: "Get string property type",
			property: Property{
				Type: "string",
			},
			expected: StringPropertyType{},
		},
		{
			name: "Get int property type",
			property: Property{
				Type: "int",
			},
			expected: IntPropertyType{},
		},
		{
			name: "Get float property type",
			property: Property{
				Type: "float",
			},
			expected: FloatPropertyType{},
		},
		{
			name: "Get bool property type",
			property: Property{
				Type: "bool",
			},
			expected: BoolPropertyType{},
		},
		{
			name: "Get resource property type",
			property: Property{
				Type: "resource",
			},
			expected: ResourcePropertyType{},
		},
		{
			name: "Get property ref property type",
			property: Property{
				Type: "propertyref",
			},
			expected: PropertyRefPropertyType{},
		},
		{
			name: "Get object property type",
			property: Property{
				Type: "object",
				Properties: map[string]Property{
					"nested": {
						Name: "nested",
					},
				},
			},
			expected: MapPropertyType{
				Property: Property{
					Type: "object",
					Properties: map[string]Property{
						"nested": {
							Name: "nested",
						},
					},
				},
			},
		},
		{
			name: "Get map property type",
			property: Property{
				Type: "map(string,string)",
			},
			expected: MapPropertyType{
				Key:   "string",
				Value: "string",
				Property: Property{
					Type: "map(string,string)",
				},
			},
		},
		{
			name: "Get list property type",
			property: Property{
				Type: "list(string)",
			},
			expected: ListPropertyType{
				Value: "string",
				Property: Property{
					Type: "list(string)",
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			actual, err := test.property.getPropertyType()
			if assert.NoError(err, "Expected no error, but got: %v", err) {
				return
			}
			assert.Equal(actual, test.expected, "expected %v, got %v", test.expected, actual)
		})
	}
}

func Test_parsePropertyValue(t *testing.T) {
	tests := []struct {
		name        string
		property    PropertyType
		value       any
		expected    any
		expectedErr bool
	}{
		{
			name:     "Parse string property value",
			property: StringPropertyType{},
			value:    "test",
			expected: "test",
		},
		{
			name:     "Parse int property value",
			property: IntPropertyType{},
			value:    1,
			expected: 1,
		},
		{
			name:     "Parse int property value as string",
			property: IntPropertyType{},
			value:    "{{ 1 }}",
			expected: 1,
		},
		{
			name:     "Parse float property value",
			property: FloatPropertyType{},
			value:    1.0,
			expected: 1.0,
		},
		{
			name:     "Parse float property value as string",
			property: FloatPropertyType{},
			value:    "{{ 1.0 }}",
			expected: float32(1.0),
		},
		{
			name:     "Parse bool property value",
			property: BoolPropertyType{},
			value:    true,
			expected: true,
		},
		{
			name:     "Parse bool property value as string template",
			property: BoolPropertyType{},
			value:    "{{ true }}",
			expected: true,
		},
		{
			name:     "Parse resource id property value",
			property: ResourcePropertyType{},
			value:    "test:resource:a",
			expected: construct.ResourceId{Provider: "test", Type: "resource", Name: "a"},
		},
		{
			name:     "Parse resource id property value as map",
			property: ResourcePropertyType{},
			value: map[string]interface{}{
				"provider": "test",
				"type":     "resource",
				"name":     "a",
			},
			expected: construct.ResourceId{Provider: "test", Type: "resource", Name: "a"},
		},
		{
			name:     "Parse resource id property value as resourceId",
			property: ResourcePropertyType{},
			value:    construct.ResourceId{Provider: "test", Type: "resource", Name: "a"},
			expected: construct.ResourceId{Provider: "test", Type: "resource", Name: "a"},
		},
		{
			name:     "Parse property ref property value",
			property: PropertyRefPropertyType{},
			value:    "test:resource:a#HOSTNAME",
			expected: construct.PropertyRef{Resource: construct.ResourceId{Provider: "test", Type: "resource", Name: "a"}, Property: "HOSTNAME"},
		},
		{
			name:     "Parse property ref property value as map",
			property: PropertyRefPropertyType{},
			value: map[string]interface{}{
				"resource": "test:resource:a",
				"property": "HOSTNAME",
			},
			expected: construct.PropertyRef{Resource: construct.ResourceId{Provider: "test", Type: "resource", Name: "a"}, Property: "HOSTNAME"},
		},
		{
			name:     "Parse property ref property value as property ref",
			property: PropertyRefPropertyType{},
			value:    construct.PropertyRef{Resource: construct.ResourceId{Provider: "test", Type: "resource", Name: "a"}, Property: "HOSTNAME"},
			expected: construct.PropertyRef{Resource: construct.ResourceId{Provider: "test", Type: "resource", Name: "a"}, Property: "HOSTNAME"},
		},
		{
			name:        "Parse invalid property value",
			property:    FloatPropertyType{},
			value:       "test",
			expectedErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			ctx := ConfigTemplateContext{}
			actual, err := test.property.Parse(test.value, ctx, ConfigTemplateData{})
			if test.expectedErr {
				assert.Error(err)
				return
			}
			assert.NoError(err, "Expected no error, but got: %v", err)
			assert.Equal(actual, test.expected, "expected %v, got %v", test.expected, actual)
		})
	}
}

func Test_MapParse(t *testing.T) {
	tests := []struct {
		name     string
		property MapPropertyType
		value    any
		expected any
		wantErr  bool
	}{
		{
			name: "Parse map property value",
			property: MapPropertyType{
				Key:   "string",
				Value: "string",
				Property: Property{
					Type: "map(string,string)",
				},
			},
			value: map[any]interface{}{
				"key":   "test",
				"value": "test",
			},
			expected: map[string]string{
				"key":   "test",
				"value": "test",
			},
		},
		{
			name: "Parse map property value with template",
			property: MapPropertyType{
				Key:   "string",
				Value: "string",
				Property: Property{
					Type: "map(string,string)",
				},
			},
			value: map[any]interface{}{
				"key":   "{{ \"test\" }}",
				"value": "{{ \"test\" }}",
			},
			expected: map[string]string{
				"key":   "test",
				"value": "test",
			},
		},
		{
			name: "Parse map property with sub properties",
			property: MapPropertyType{
				Property: Property{
					Type: "object",
					Properties: map[string]Property{
						"nested": {
							Name: "nested",
							Type: "string",
						},
						"second": {
							Name: "second",
							Type: "bool",
						},
					},
				},
			},
			value: map[any]interface{}{
				"nested": "test",
				"second": true,
			},
			expected: map[string]interface{}{
				"nested": "test",
				"second": true,
			},
		},
		{
			name: "Parse map property with sub properties incorrect, should error",
			property: MapPropertyType{
				Property: Property{
					Type: "object",
					Properties: map[string]Property{
						"nested": {
							Name: "nested",
							Type: "string",
						},
					},
				},
			},
			value: map[any]interface{}{
				"nested": "test",
				"second": true,
			},
			wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			ctx := ConfigTemplateContext{}
			actual, err := test.property.Parse(test.value, ctx, ConfigTemplateData{})
			if test.wantErr {
				assert.Error(err)
				return
			}
			assert.NoError(err, "Expected no error, but got: %v", err)
			// Because it can be int64, int, etc just equals on the map can fail
			assert.Equal(fmt.Sprintf("%v", actual), fmt.Sprintf("%v", test.expected), "expected %v, got %v", test.expected, actual)
		})
	}
}

func Test_ListParse(t *testing.T) {
	tests := []struct {
		name     string
		property ListPropertyType
		value    any
		expected any
		wantErr  bool
	}{
		{
			name: "Parse list property value",
			property: ListPropertyType{
				Value: "string",
				Property: Property{
					Type: "list(string)",
				},
			},
			value: []interface{}{
				"test",
				"test",
			},
			expected: []string{
				"test",
				"test",
			},
		},
		{
			name: "Parse list property value with template",
			property: ListPropertyType{
				Value: "string",
				Property: Property{
					Type: "list(string)",
				},
			},
			value: []interface{}{
				"{{ \"test\" }}",
				"{{ \"test\" }}",
			},
			expected: []string{
				"test",
				"test",
			},
		},
		{
			name: "Parse list property with sub properties",
			property: ListPropertyType{
				Property: Property{
					Type: "object",
					Properties: map[string]Property{
						"nested": {
							Name: "nested",
							Type: "string",
						},
						"second": {
							Name: "second",
							Type: "bool",
						},
					},
				},
			},
			value: []interface{}{
				map[string]interface{}{
					"nested": "test",
					"second": true,
				},
				map[string]interface{}{
					"nested": "test",
					"second": true,
				},
			},
			expected: []interface{}{
				map[string]interface{}{
					"nested": "test",
					"second": true,
				},
				map[string]interface{}{
					"nested": "test",
					"second": true,
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			ctx := ConfigTemplateContext{}
			actual, err := test.property.Parse(test.value, ctx, ConfigTemplateData{})
			if test.wantErr {
				assert.Error(err)
				return
			}
			assert.NoError(err, "Expected no error, but got: %v", err)
			// Because it can be int64, int, etc just equals on the map can fail
			assert.Equal(fmt.Sprintf("%v", actual), fmt.Sprintf("%v", test.expected), "expected %v, got %v", test.expected, actual)
		})
	}
}
