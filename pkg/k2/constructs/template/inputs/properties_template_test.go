package inputs

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/k2/constructs/template/properties"
	"github.com/klothoplatform/klotho/pkg/k2/constructs/template/property"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ConvertProperty(t *testing.T) {
	tests := []struct {
		name     string
		property InputTemplate
		expected property.Property
	}{
		{
			name: "Convert string property type",
			property: InputTemplate{
				Type:          "string",
				Name:          "test",
				Path:          "test",
				Required:      true,
				AllowedValues: []string{"test1", "test2"},
				DefaultValue:  "test",
			},
			expected: &properties.StringProperty{
				SharedPropertyFields: properties.SharedPropertyFields{
					DefaultValue: "test",
				},
				PropertyDetails: property.PropertyDetails{
					Name:     "test",
					Required: true,
					Path:     "test",
				},
				AllowedValues: []string{"test1", "test2"},
			},
		},
		{
			name: "Convert int property type",
			property: InputTemplate{
				Type:     "int",
				Name:     "test",
				Path:     "test",
				Required: true,
			},
			expected: &properties.IntProperty{
				PropertyDetails: property.PropertyDetails{
					Name:     "test",
					Required: true,
					Path:     "test",
				},
			},
		},
		{
			name: "Convert float property type",
			property: InputTemplate{
				Type:     "float",
				Name:     "test",
				Path:     "test",
				Required: true,
			},
			expected: &properties.FloatProperty{
				PropertyDetails: property.PropertyDetails{
					Name:     "test",
					Required: true,
					Path:     "test",
				},
			},
		},
		{
			name: "Convert bool property type",
			property: InputTemplate{
				Type:     "bool",
				Name:     "test",
				Path:     "test",
				Required: true,
			},
			expected: &properties.BoolProperty{
				PropertyDetails: property.PropertyDetails{
					Name:     "test",
					Required: true,
					Path:     "test",
				},
			},
		},
		{
			name: "Convert map property type",
			property: InputTemplate{
				Type:     "map(string,string)",
				Name:     "test",
				Path:     "test",
				Required: true,
				KeyProperty: &InputTemplate{
					Type: "string",
				},
				ValueProperty: &InputTemplate{
					Type: "string",
				},
			},
			expected: &properties.MapProperty{
				PropertyDetails: property.PropertyDetails{
					Name:     "test",
					Required: true,
					Path:     "test",
				},
				Properties:    map[string]property.Property{},
				KeyProperty:   &properties.StringProperty{},
				ValueProperty: &properties.StringProperty{},
			},
		},
		{
			name: "Convert list property type",
			property: InputTemplate{
				Type:     "list(string)",
				Name:     "test",
				Path:     "test",
				Required: true,
			},
			expected: &properties.ListProperty{
				PropertyDetails: property.PropertyDetails{
					Name:     "test",
					Required: true,
					Path:     "test",
				},
				ItemProperty: &properties.StringProperty{},
				Properties:   map[string]property.Property{},
			},
		},
		{
			name: "Convert set property type",
			property: InputTemplate{
				Type:     "set(string)",
				Name:     "test",
				Path:     "test",
				Required: true,
				ItemProperty: &InputTemplate{
					Type: "string",
				},
			},
			expected: &properties.SetProperty{
				PropertyDetails: property.PropertyDetails{
					Name:     "test",
					Required: true,
					Path:     "test",
				},
				ItemProperty: &properties.StringProperty{},
				Properties:   map[string]property.Property{},
			},
		},
		{
			name: "Convert key_value_list property type",
			property: InputTemplate{
				Type:     "key_value_list(string,string)",
				Name:     "test",
				Path:     "test",
				Required: true,
			},
			expected: &properties.KeyValueListProperty{
				PropertyDetails: property.PropertyDetails{
					Name:     "test",
					Required: true,
					Path:     "test",
				},
				KeyProperty:   &properties.StringProperty{PropertyDetails: property.PropertyDetails{Name: "Key"}},
				ValueProperty: &properties.StringProperty{PropertyDetails: property.PropertyDetails{Name: "Value"}},
			},
		},
		{
			name: "Convert key_value_list property type with custom key and value properties",
			property: InputTemplate{
				Type: "key_value_list(string,string)",
				Name: "test",
				Path: "test",
				KeyProperty: &InputTemplate{
					Type: "string",
					Name: "CustomKeyKey",
				},
				ValueProperty: &InputTemplate{
					Type: "string",
					Name: "CustomValueKey",
				},
			},
			expected: &properties.KeyValueListProperty{
				PropertyDetails: property.PropertyDetails{
					Name: "test",
					Path: "test",
				},
				KeyProperty:   &properties.StringProperty{PropertyDetails: property.PropertyDetails{Name: "CustomKeyKey"}},
				ValueProperty: &properties.StringProperty{PropertyDetails: property.PropertyDetails{Name: "CustomValueKey"}},
			},
		},
		{
			name: "Convert construct property type",
			property: InputTemplate{
				Type:     "construct",
				Name:     "test",
				Path:     "test",
				Required: true,
			},
			expected: &properties.ConstructProperty{
				PropertyDetails: property.PropertyDetails{
					Name:     "test",
					Required: true,
					Path:     "test",
				},
			},
		},
		{
			name: "Convert any property type",
			property: InputTemplate{
				Type:     "any",
				Name:     "test",
				Path:     "test",
				Required: true,
			},
			expected: &properties.AnyProperty{
				PropertyDetails: property.PropertyDetails{
					Name:     "test",
					Required: true,
					Path:     "test",
				},
			},
		},
		{
			name: "Convert path property type",
			property: InputTemplate{
				Type:     "path",
				Name:     "test",
				Path:     "test",
				Required: true,
			},
			expected: &properties.PathProperty{
				PropertyDetails: property.PropertyDetails{
					Name:     "test",
					Required: true,
					Path:     "test",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := test.property.Convert()
			require.NoError(t, err)
			assert.EqualValuesf(t, actual, test.expected, "expected %v, got %v", test.expected, actual)
		})
	}
}
