package knowledgebase2

import (
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/set"
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
			expected: &StringPropertyType{},
		},
		{
			name: "Get int property type",
			property: Property{
				Type: "int",
			},
			expected: &IntPropertyType{},
		},
		{
			name: "Get float property type",
			property: Property{
				Type: "float",
			},
			expected: &FloatPropertyType{},
		},
		{
			name: "Get bool property type",
			property: Property{
				Type: "bool",
			},
			expected: &BoolPropertyType{},
		},
		{
			name: "Get resource property type",
			property: Property{
				Type: "resource",
			},
			expected: &ResourcePropertyType{},
		},
		{
			name: "Get resource property type with resource value",
			property: Property{
				Type: "resource(test:type)",
			},
			expected: &ResourcePropertyType{
				Value: construct.ResourceId{Provider: "test", Type: "type"},
			},
		},
		{
			name: "Get map with sub fields property type",
			property: Property{
				Type: "map",
				Properties: map[string]Property{
					"nested": {
						Name: "nested",
						Type: "string",
					},
				},
			},
			expected: &MapPropertyType{
				Property: Property{
					Type: "map",
					Properties: map[string]Property{
						"nested": {
							Name: "nested",
							Type: "string",
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
			expected: &MapPropertyType{
				Key:   "string",
				Value: "string",
				Property: Property{
					Type: "map(string,string)",
				},
			},
		},
		{
			name: "Get map property type with nested value",
			property: Property{
				Type: "map(string,list(string))",
			},
			expected: &MapPropertyType{
				Key:   "string",
				Value: "list(string)",
				Property: Property{
					Type: "map(string,list(string))",
				},
			},
		},
		{
			name: "Get list with sub fields property type",
			property: Property{
				Type: "list",
				Properties: map[string]Property{
					"nested": {
						Name: "nested",
						Type: "string",
					},
				},
			},
			expected: &ListPropertyType{
				Property: Property{
					Type: "list",
					Properties: map[string]Property{
						"nested": {
							Name: "nested",
							Type: "string",
						},
					},
				},
			},
		},
		{
			name: "Get list property type",
			property: Property{
				Type: "list(string)",
			},
			expected: &ListPropertyType{
				Value: "string",
				Property: Property{
					Type: "list(string)",
				},
			},
		},
		{
			name: "Get list property type nested value",
			property: Property{
				Type: "list(map(string,list(string)))",
			},
			expected: &ListPropertyType{
				Value: "map(string,list(string))",
				Property: Property{
					Type: "list(map(string,list(string)))",
				},
			},
		},
		{
			name: "Get list property type with nested value",
			property: Property{
				Type: "list(map(string, string))",
			},
			expected: &ListPropertyType{
				Value: "map(string, string)",
				Property: Property{
					Type: "list(map(string, string))",
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			actual, err := test.property.PropertyType()
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
			property: &StringPropertyType{},
			value:    "test",
			expected: "test",
		},
		{
			name:     "Parse int property value",
			property: &IntPropertyType{},
			value:    1,
			expected: 1,
		},
		{
			name:     "Parse int property value as string",
			property: &IntPropertyType{},
			value:    "{{ 1 }}",
			expected: 1,
		},
		{
			name:     "Parse float property value",
			property: &FloatPropertyType{},
			value:    1.0,
			expected: 1.0,
		},
		{
			name:     "Parse float property value as string",
			property: &FloatPropertyType{},
			value:    "{{ 1.0 }}",
			expected: float32(1.0),
		},
		{
			name:     "Parse bool property value",
			property: &BoolPropertyType{},
			value:    true,
			expected: true,
		},
		{
			name:     "Parse bool property value as string template",
			property: &BoolPropertyType{},
			value:    "{{ true }}",
			expected: true,
		},
		{
			name:     "Parse resource id property value",
			property: &ResourcePropertyType{},
			value:    "test:resource:a",
			expected: construct.ResourceId{Provider: "test", Type: "resource", Name: "a"},
		},
		{
			name:     "Parse property ref property value",
			property: &PropertyRefPropertyType{},
			value:    "test:resource:a#HOSTNAME",
			expected: construct.PropertyRef{Resource: construct.ResourceId{Provider: "test", Type: "resource", Name: "a"}, Property: "HOSTNAME"},
		},
		{
			name:     "Parse property ref property value as map",
			property: &PropertyRefPropertyType{},
			value: map[string]interface{}{
				"resource": "test:resource:a",
				"property": "HOSTNAME",
			},
			expected: construct.PropertyRef{Resource: construct.ResourceId{Provider: "test", Type: "resource", Name: "a"}, Property: "HOSTNAME"},
		},
		{
			name:     "Parse property ref property value as property ref",
			property: &PropertyRefPropertyType{},
			value:    construct.PropertyRef{Resource: construct.ResourceId{Provider: "test", Type: "resource", Name: "a"}, Property: "HOSTNAME"},
			expected: construct.PropertyRef{Resource: construct.ResourceId{Provider: "test", Type: "resource", Name: "a"}, Property: "HOSTNAME"},
		},
		{
			name:        "Parse invalid property value",
			property:    &FloatPropertyType{},
			value:       "test",
			expectedErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			ctx := DynamicValueContext{}
			actual, err := test.property.Parse(test.value, ctx, DynamicValueData{})
			if test.expectedErr {
				assert.Error(err)
				return
			}
			assert.NoError(err, "Expected no error, but got: %v", err)
			assert.Equal(actual, test.expected, "expected %v, got %v", test.expected, actual)
		})
	}
}

func Test_parseResourcePropertyValue(t *testing.T) {
	tests := []struct {
		name        string
		property    PropertyType
		value       any
		expected    any
		expectedErr bool
	}{
		{
			name:     "Parse resource id property value as map",
			property: &ResourcePropertyType{},
			value: map[string]interface{}{
				"provider": "test",
				"type":     "resource",
				"name":     "a",
			},
			expected: construct.ResourceId{Provider: "test", Type: "resource", Name: "a"},
		},
		{
			name:     "Parse resource id property value as resourceId",
			property: &ResourcePropertyType{},
			value:    construct.ResourceId{Provider: "test", Type: "resource", Name: "a"},
			expected: construct.ResourceId{Provider: "test", Type: "resource", Name: "a"},
		},
		{
			name: "Parse resource id correct type",
			property: &ResourcePropertyType{
				Value: construct.ResourceId{Provider: "test", Type: "resource"},
			},
			value:    construct.ResourceId{Provider: "test", Type: "resource", Name: "a"},
			expected: construct.ResourceId{Provider: "test", Type: "resource", Name: "a"},
		},
		{
			name: "Parse resource id invalid type excpet err",
			property: &ResourcePropertyType{
				Value: construct.ResourceId{Provider: "test", Type: "r"},
			},
			value:       construct.ResourceId{Provider: "test", Type: "resource", Name: "a"},
			expectedErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			ctx := DynamicValueContext{}
			actual, err := test.property.Parse(test.value, ctx, DynamicValueData{})
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
			name: "Parse map property value with template",
			property: MapPropertyType{
				Key:   "string",
				Value: "string",
				Property: Property{
					Type: "map(string,string)",
				},
			},
			value: map[string]interface{}{
				"key":   "{{ \"test\" }}",
				"value": "{{ \"test\" }}",
			},
			expected: map[string]interface{}{
				"key":   "test",
				"value": "test",
			},
		},
		{
			name: "Parse map property value with nested type",
			property: MapPropertyType{
				Key:   "string",
				Value: "list(string)",
				Property: Property{
					Type: "map(string,list(string))",
				},
			},
			value: map[string]interface{}{
				"key":   []any{"test"},
				"value": []any{"test"},
			},
			expected: map[string]interface{}{
				"key":   []any{"test"},
				"value": []any{"test"},
			},
		},
		{
			name: "Parse map property with sub properties",
			property: MapPropertyType{
				Property: Property{
					Type: "map",
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
			value: map[string]interface{}{
				"nested": "test",
				"second": true,
			},
			expected: map[string]interface{}{
				"nested": "test",
				"second": true,
			},
		},
		{

			name: "Parse map property with sub properties and default values",
			property: MapPropertyType{
				Property: Property{
					Type: "map",
					Properties: map[string]Property{
						"nested": {
							Name:         "nested",
							Type:         "string",
							DefaultValue: "test",
						},
						"second": {
							Name:         "second",
							Type:         "bool",
							DefaultValue: true,
						},
					},
				},
			},
			value: map[string]interface{}{
				"nested": "notTest",
			},
			expected: map[string]interface{}{
				"nested": "notTest",
				"second": true,
			},
		},
		{
			name: "Parse map property with sub properties incorrect, should error",
			property: MapPropertyType{
				Property: Property{
					Type: "map",
					Properties: map[string]Property{
						"nested": {
							Name: "nested",
							Type: "object",
						},
					},
				},
			},
			value: map[string]interface{}{
				"nested": "test",
				"second": true,
			},
			wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			ctx := DynamicValueContext{}
			actual, err := test.property.Parse(test.value, ctx, DynamicValueData{})
			if test.wantErr {
				assert.Error(err)
				return
			}
			assert.NoError(err, "Expected no error, but got: %v", err)
			// Because it can be int64, int, etc just equals on the map can fail
			assert.Equal(actual, test.expected, "expected %v, got %v", test.expected, actual)

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
			expected: []any{
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
			expected: []any{
				"test",
				"test",
			},
		},
		{
			name: "Parse list property value with nested fields",
			property: ListPropertyType{
				Value: "map(string,list(string))",
				Property: Property{
					Type: "list(map(string,list(string)))",
				},
			},
			value: []interface{}{
				map[string]interface{}{
					"test": []any{"v"},
				},
			},
			expected: []any{
				map[string]any{
					"test": []any{"v"},
				},
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
			ctx := DynamicValueContext{}
			actual, err := test.property.Parse(test.value, ctx, DynamicValueData{})
			if test.wantErr {
				assert.Error(err)
				return
			}
			assert.NoError(err, "Expected no error, but got: %v", err)
			// Because it can be int64, int, etc just equals on the map can fail
			assert.Equal(actual, test.expected, "expected %v, got %v", test.expected, actual)
		})
	}
}

func Test_SetParse(t *testing.T) {
	tests := []struct {
		name     string
		property SetPropertyType
		value    any
		expected any
		wantErr  bool
	}{
		{
			name: "Parse set property value",
			property: SetPropertyType{
				Value: "string",
				Property: Property{
					Type: "list(string)",
				},
			},
			value: []interface{}{
				"test",
				"test",
			},
			expected: []any{
				"test",
			},
		},
		{
			name: "Parse list property value with template",
			property: SetPropertyType{
				Value: "string",
				Property: Property{
					Type: "list(string)",
				},
			},
			value: []interface{}{
				"{{ \"test\" }}",
				"{{ \"test\" }}",
			},
			expected: []any{
				"test",
			},
		},
		{
			name: "Parse list property value with nested fields",
			property: SetPropertyType{
				Value: "map(string,list(string))",
				Property: Property{
					Type: "list(map(string,list(string)))",
				},
			},
			value: []interface{}{
				map[string]interface{}{
					"test": []any{"v"},
				},
			},
			expected: []any{
				map[string]any{
					"test": []any{"v"},
				},
			},
		},
		{
			name: "Parse list property with sub properties",
			property: SetPropertyType{
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
			expected: []any{
				map[string]any{
					"nested": "test",
					"second": true,
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			ctx := DynamicValueContext{}
			actual, err := test.property.Parse(test.value, ctx, DynamicValueData{})
			if test.wantErr {
				assert.Error(err)
				return
			}
			hashedSet, ok := actual.(set.HashedSet[string, any])
			if !ok {
				assert.Fail("Expected set.HashedSet[string, any]")
			}
			assert.NoError(err, "Expected no error, but got: %v", err)
			// Because it can be int64, int, etc just equals on the map can fail
			assert.Equal(hashedSet.ToSlice(), test.expected, "expected %v, got %v", test.expected, actual)
		})
	}
}
