package properties

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/k2/constructs/template/property"
	"github.com/stretchr/testify/assert"
)

// Testing the SetProperty method for different cases
func Test_FloatProperty_SetProperty(t *testing.T) {
	tests := []struct {
		name      string
		property  *FloatProperty
		input     any
		wantError bool
	}{
		{
			name: "valid float64 value",
			property: &FloatProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			input:     float64(3.14),
			wantError: false,
		},
		{
			name: "valid float32 value",
			property: &FloatProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			input:     float32(3.14),
			wantError: false,
		},
		{
			name: "valid int value",
			property: &FloatProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			input:     42,
			wantError: false,
		},
		{
			name: "invalid string value",
			property: &FloatProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			input:     "invalid_float",
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
func Test_FloatProperty_ZeroValue(t *testing.T) {
	assert := assert.New(t)
	property := &FloatProperty{}
	assert.Equal(0.0, property.ZeroValue())
}

// Testing the Details method
func Test_FloatProperty_Details(t *testing.T) {
	assert := assert.New(t)
	property := &FloatProperty{}
	assert.Same(&property.PropertyDetails, property.Details())
}

// Testing the Clone method
func Test_FloatProperty_Clone(t *testing.T) {
	property := &FloatProperty{
		PropertyDetails: property.PropertyDetails{Path: "test"},
		MinValue:        new(float64),
		MaxValue:        new(float64),
	}
	clone := property.Clone()
	assert.Equal(t, property, clone)
}

// Testing the AppendProperty method with different cases
func Test_FloatProperty_AppendProperty(t *testing.T) {
	tests := []struct {
		name       string
		property   *FloatProperty
		properties construct.Properties
		input      any
		wantError  bool
	}{
		{
			name: "append float64 value",
			property: &FloatProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			properties: construct.Properties{},
			input:      float64(3.14),
			wantError:  false,
		},
		{
			name: "append int value",
			property: &FloatProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			properties: construct.Properties{},
			input:      42,
			wantError:  false,
		},
		{
			name: "append invalid value",
			property: &FloatProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			properties: construct.Properties{},
			input:      "invalid_float",
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
func Test_FloatProperty_RemoveProperty(t *testing.T) {
	tests := []struct {
		name       string
		property   *FloatProperty
		properties construct.Properties
		input      any
		wantError  bool
	}{
		{
			name: "remove existing float property",
			property: &FloatProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			properties: construct.Properties{"test": float64(3.14)},
			wantError:  false,
		},
		{
			name: "remove non-existent property",
			property: &FloatProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			properties: construct.Properties{},
			wantError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.property.RemoveProperty(tt.properties, nil)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Testing the Parse method
func Test_FloatProperty_Parse(t *testing.T) {
	tests := []struct {
		name      string
		property  *FloatProperty
		input     any
		expected  any
		wantError bool
	}{
		{
			name:      "parse string to float",
			property:  &FloatProperty{},
			input:     "3.14",
			expected:  float32(3.14),
			wantError: false,
		},
		{
			name:      "parse int to float",
			property:  &FloatProperty{},
			input:     42,
			expected:  float64(42),
			wantError: false,
		},
		{
			name:      "parse float64",
			property:  &FloatProperty{},
			input:     float64(3.14),
			expected:  float64(3.14),
			wantError: false,
		},
		{
			name:      "parse invalid string",
			property:  &FloatProperty{},
			input:     "invalid_float",
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
func Test_FloatProperty_Contains(t *testing.T) {
	assert := assert.New(t)
	property := &FloatProperty{}
	assert.False(property.Contains(1.0, 1.0))
}

// Testing the Type method
func Test_FloatProperty_Type(t *testing.T) {
	assert := assert.New(t)
	property := &FloatProperty{}
	assert.Equal("float", property.Type())
}

// Testing the SubProperties method
func Test_FloatProperty_SubProperties(t *testing.T) {
	assert := assert.New(t)
	property := &FloatProperty{}
	assert.Nil(property.SubProperties())
}
