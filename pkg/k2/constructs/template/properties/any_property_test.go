package properties

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/k2/constructs/template/property"
	"github.com/stretchr/testify/assert"
)

// Testing the SetProperty method for different cases
func Test_AnyProperty_SetProperty(t *testing.T) {
	tests := []struct {
		name      string
		property  *AnyProperty
		input     any
		wantError bool
	}{
		{
			name: "valid string value",
			property: &AnyProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			input:     "valid_string",
			wantError: false,
		},
		{
			name: "valid map value",
			property: &AnyProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			input:     map[string]any{"key1": "value1", "key2": "value2"},
			wantError: false,
		},
		{
			name: "valid list value",
			property: &AnyProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			input:     []any{"item1", "item2"},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			properties := construct.Properties{}
			err := tt.property.SetProperty(properties, tt.input)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Testing the ZeroValue method
func Test_AnyProperty_ZeroValue(t *testing.T) {
	assert := assert.New(t)
	property := &AnyProperty{}
	assert.Nil(property.ZeroValue())
}

// Testing the Details method
func Test_AnyProperty_Details(t *testing.T) {
	assert := assert.New(t)
	property := &AnyProperty{}
	assert.Same(&property.PropertyDetails, property.Details())
}

// Testing the Clone method
func Test_AnyProperty_Clone(t *testing.T) {
	property := &AnyProperty{
		PropertyDetails: property.PropertyDetails{Path: "test"},
	}
	clone := property.Clone()
	assert.Equal(t, property, clone)
}

// Testing the AppendProperty method with different cases
func Test_AnyProperty_AppendProperty(t *testing.T) {
	tests := []struct {
		name       string
		property   *AnyProperty
		properties construct.Properties
		input      any
		expected   any
		wantError  bool
	}{
		{
			name: "append to empty property",
			property: &AnyProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			properties: construct.Properties{},
			input:      map[string]any{"key1": "value1"},
			wantError:  true,
		},
		{
			name: "append to existing map property",
			property: &AnyProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			properties: construct.Properties{
				"test": map[string]any{"key1": "value1"},
			},
			input:     map[string]any{"key2": "value2"},
			expected:  map[string]any{"key1": "value1", "key2": "value2"},
			wantError: false,
		},
		{
			name: "append invalid type to map property",
			property: &AnyProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			properties: construct.Properties{
				"test": map[string]any{"key1": "value1"},
			},
			input:     "invalid_value",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.property.AppendProperty(tt.properties, tt.input)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, tt.properties[tt.property.Path])
			}
		})
	}
}

// Testing the RemoveProperty method
func Test_AnyProperty_RemoveProperty(t *testing.T) {
	tests := []struct {
		name       string
		property   *AnyProperty
		properties construct.Properties
		input      any
		expected   any
		wantError  bool
	}{
		{
			name: "remove existing map entry",
			property: &AnyProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			properties: construct.Properties{
				"test": map[string]any{"key1": "value1", "key2": "value2"},
			},
			input:     "key1",
			expected:  map[string]any{"key2": "value2"},
			wantError: true, // DS - not sure if this is correct or an existing bug
		},
		{
			name: "remove existing list entry",
			property: &AnyProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			properties: construct.Properties{
				"test": []any{"item1", "item2"},
			},
			input:     "item1",
			expected:  []any{"item2"},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.property.RemoveProperty(tt.properties, tt.input)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, tt.properties[tt.property.Path])
			}
		})
	}
}

// Testing the Parse method
func Test_AnyProperty_Parse(t *testing.T) {
	tests := []struct {
		name      string
		property  *AnyProperty
		input     any
		expected  any
		wantError bool
	}{
		{
			name:      "parse string template",
			property:  &AnyProperty{},
			input:     "{{ toUpper \"VALUE\" }}",
			expected:  "VALUE",
			wantError: false,
		},
		{
			name:      "parse map value",
			property:  &AnyProperty{},
			input:     map[string]any{"key1": "value1", "key2": "value2"},
			expected:  map[string]any{"key1": "value1", "key2": "value2"},
			wantError: false,
		},
		{
			name:      "parse list value",
			property:  &AnyProperty{},
			input:     []any{"item1", "item2"},
			expected:  []any{"item1", "item2"},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.property.Parse(tt.input, DefaultExecutionContext{}, nil)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
