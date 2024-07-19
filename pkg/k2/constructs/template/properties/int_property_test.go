package properties

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/k2/constructs/template/property"
	"github.com/stretchr/testify/assert"
)

// Testing the SetProperty method for different cases
func Test_IntProperty_SetProperty(t *testing.T) {
	tests := []struct {
		name      string
		property  *IntProperty
		input     any
		wantError bool
	}{
		{
			name: "valid int value",
			property: &IntProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			input:     42,
			wantError: false,
		},
		{
			name: "invalid float value",
			property: &IntProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			input:     float32(42.0),
			wantError: true,
		},
		{
			name: "invalid string value",
			property: &IntProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			input:     "invalid_int",
			wantError: true,
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
func Test_IntProperty_ZeroValue(t *testing.T) {
	assert := assert.New(t)
	property := &IntProperty{}
	assert.Equal(0, property.ZeroValue())
}

// Testing the Details method
func Test_IntProperty_Details(t *testing.T) {
	assert := assert.New(t)
	property := &IntProperty{}
	assert.Same(&property.PropertyDetails, property.Details())
}

// Testing the Clone method
func Test_IntProperty_Clone(t *testing.T) {
	property := &IntProperty{
		PropertyDetails: property.PropertyDetails{Path: "test"},
		MinValue:        new(int),
		MaxValue:        new(int),
	}
	clone := property.Clone()
	assert.Equal(t, property, clone)
}

// Testing the AppendProperty method with different cases
func Test_IntProperty_AppendProperty(t *testing.T) {
	tests := []struct {
		name       string
		property   *IntProperty
		properties construct.Properties
		input      any
		wantError  bool
	}{
		{
			name: "append int value",
			property: &IntProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			properties: construct.Properties{},
			input:      42,
			wantError:  false,
		},
		{
			name: "append invalid float value",
			property: &IntProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			properties: construct.Properties{},
			input:      float32(42.0),
			wantError:  true,
		},
		{
			name: "append invalid string value",
			property: &IntProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			properties: construct.Properties{},
			input:      "invalid_int",
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.property.AppendProperty(tt.properties, tt.input)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Testing the RemoveProperty method
func Test_IntProperty_RemoveProperty(t *testing.T) {
	tests := []struct {
		name       string
		property   *IntProperty
		properties construct.Properties
		wantError  bool
	}{
		{
			name: "remove existing int property",
			property: &IntProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			properties: construct.Properties{"test": 42},
		},
		{
			name: "remove non-existent property",
			property: &IntProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			properties: construct.Properties{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			err := test.property.RemoveProperty(test.properties, nil)
			if test.wantError {
				assert.Error(err)
				return
			} else {
				assert.NoError(err)
			}
			assert.NotContains(test.properties, test.property.Path)
		})
	}
}

// Testing the Parse method
func Test_IntProperty_Parse(t *testing.T) {
	tests := []struct {
		name      string
		property  *IntProperty
		input     any
		expected  any
		wantError bool
	}{
		{
			name:      "parse string to int",
			property:  &IntProperty{},
			input:     "42",
			expected:  42,
			wantError: false,
		},
		{
			name:      "parse int",
			property:  &IntProperty{},
			input:     42,
			expected:  42,
			wantError: false,
		},
		{
			name:      "parse float",
			property:  &IntProperty{},
			input:     float32(42.0),
			expected:  42,
			wantError: false,
		},
		{
			name:      "parse invalid string",
			property:  &IntProperty{},
			input:     "invalid_int",
			expected:  nil,
			wantError: true,
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

// Testing the Contains method
func Test_IntProperty_Contains(t *testing.T) {
	assert := assert.New(t)
	property := &IntProperty{}
	assert.False(property.Contains(1, 1))
}

// Testing the Type method
func Test_IntProperty_Type(t *testing.T) {
	assert := assert.New(t)
	property := &IntProperty{}
	assert.Equal("int", property.Type())
}

// Testing the SubProperties method
func Test_IntProperty_SubProperties(t *testing.T) {
	assert := assert.New(t)
	property := &IntProperty{}
	assert.Nil(property.SubProperties())
}
