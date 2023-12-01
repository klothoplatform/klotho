package properties

import (
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
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
				KeyProperty:   &StringProperty{},
				ValueProperty: &StringProperty{},
			},
			resource: &construct.Resource{
				Properties: map[string]any{
					"key":   "test",
					"value": "test",
				},
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
			assert.NoError(err, "Expected no error, but got: %v", err)
			assert.Equal(test.resource.Properties, test.value, "expected %v, got %v", test.value, test.resource.Properties)
		})
	}

}

// func Test_MapParse(t *testing.T) {
// 	tests := []struct {
// 		name     string
// 		property MapPropertyType
// 		value    any
// 		expected any
// 		wantErr  bool
// 	}{
// 		{
// 			name: "Parse map property value",
// 			property: MapPropertyType{
// 				Key:   "string",
// 				Value: "string",
// 				Property: &Property{
// 					Type: "map(string,string)",
// 				},
// 			},
// 			value: map[string]interface{}{
// 				"key":   "test",
// 				"value": "test",
// 			},
// 			expected: map[string]interface{}{
// 				"key":   "test",
// 				"value": "test",
// 			},
// 		},
// 		{
// 			name: "Parse map property value with template",
// 			property: MapPropertyType{
// 				Key:   "string",
// 				Value: "string",
// 				Property: &Property{
// 					Type: "map(string,string)",
// 				},
// 			},
// 			value: map[string]interface{}{
// 				"key":   "{{ \"test\" }}",
// 				"value": "{{ \"test\" }}",
// 			},
// 			expected: map[string]interface{}{
// 				"key":   "test",
// 				"value": "test",
// 			},
// 		},
// 		{
// 			name: "Parse map property value with nested type",
// 			property: MapPropertyType{
// 				Key:   "string",
// 				Value: "list(string)",
// 				Property: &Property{
// 					Type: "map(string,list(string))",
// 				},
// 			},
// 			value: map[string]interface{}{
// 				"key":   []any{"test"},
// 				"value": []any{"test"},
// 			},
// 			expected: map[string]interface{}{
// 				"key":   []any{"test"},
// 				"value": []any{"test"},
// 			},
// 		},
// 		{
// 			name: "Parse map property with sub properties",
// 			property: MapPropertyType{
// 				Property: &Property{
// 					Type: "map",
// 					Properties: map[string]*Property{
// 						"nested": {
// 							Name: "nested",
// 							Type: "string",
// 						},
// 						"second": {
// 							Name: "second",
// 							Type: "bool",
// 						},
// 					},
// 				},
// 			},
// 			value: map[string]interface{}{
// 				"nested": "test",
// 				"second": true,
// 			},
// 			expected: map[string]interface{}{
// 				"nested": "test",
// 				"second": true,
// 			},
// 		},
// 		{
// 			name: "Parse map property with sub properties incorrect, should error",
// 			property: MapPropertyType{
// 				Property: &Property{
// 					Type: "map",
// 					Properties: map[string]*Property{
// 						"nested": {
// 							Name: "nested",
// 							Type: "object",
// 						},
// 					},
// 				},
// 			},
// 			value: map[string]interface{}{
// 				"nested": "test",
// 				"second": true,
// 			},
// 			wantErr: true,
// 		},
// 	}
// 	for _, test := range tests {
// 		t.Run(test.name, func(t *testing.T) {
// 			assert := assert.New(t)
// 			ctx := DynamicValueContext{}
// 			actual, err := test.property.Parse(test.value, ctx, DynamicValueData{})
// 			if test.wantErr {
// 				assert.Error(err)
// 				return
// 			}
// 			assert.NoError(err, "Expected no error, but got: %v", err)
// 			// Because it can be int64, int, etc just equals on the map can fail
// 			assert.Equal(actual, test.expected, "expected %v, got %v", test.expected, actual)

// 		})
// 	}
// }
