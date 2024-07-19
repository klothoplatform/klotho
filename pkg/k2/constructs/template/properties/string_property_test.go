package properties

import (
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/k2/constructs/template/property"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_StringPropertySetProperty(t *testing.T) {
	tests := []struct {
		name      string
		property  *StringProperty
		input     any
		wantError bool
	}{
		{
			name: "valid string value",
			property: &StringProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			input:     "valid_string",
			wantError: false,
		},
		{
			name: "invalid value type",
			property: &StringProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			input:     123,
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

func Test_StringPropertyZeroValue(t *testing.T) {
	assert := assert.New(t)
	property := &StringProperty{}
	assert.Equal("", property.ZeroValue())
}

func Test_StringPropertyDetails(t *testing.T) {
	assert := assert.New(t)
	property := &StringProperty{}
	assert.Same(&property.PropertyDetails, property.Details())
}

func Test_StringPropertyContains(t *testing.T) {
	tests := []struct {
		name     string
		property *StringProperty
		value    any
		contains any
		expected bool
	}{
		{
			name: "string contains value",
			property: &StringProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			value:    "hello world",
			contains: "hello",
			expected: true,
		},
		{
			name: "string does not contain value",
			property: &StringProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			value:    "hello world",
			contains: "goodbye",
			expected: false,
		},
		{
			name: "non-string value",
			property: &StringProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			value:    123,
			contains: "1",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.property.Contains(tt.value, tt.contains)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func Test_StringPropertyType(t *testing.T) {
	assert := assert.New(t)
	property := &StringProperty{}
	assert.Equal("string", property.Type())
}

func Test_StringPropertyValidate(t *testing.T) {
	tests := []struct {
		name             string
		property         *StringProperty
		value            any
		wantErr          bool
		sanitizeTemplate string
		allowedValues    []string
	}{
		{
			name: "valid string in allowed values",
			property: &StringProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
				AllowedValues:   []string{"allowed_value"},
			},
			value:   "allowed_value",
			wantErr: false,
		},
		{
			name: "string not in allowed values",
			property: &StringProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
				AllowedValues:   []string{"allowed_value"},
			},
			value:   "disallowed_value",
			wantErr: true,
		},
		{
			name: "invalid value type",
			property: &StringProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			value:   123,
			wantErr: true,
		},
		{
			name: "sanitized string value",
			property: &StringProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			sanitizeTemplate: "{{ . | upper }}",
			value:            "TEST",
			wantErr:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			properties := construct.Properties{}
			if tt.sanitizeTemplate != "" {
				tmpl, err := property.NewSanitizationTmpl(tt.name, tt.sanitizeTemplate)
				if !assert.NoError(t, err) {
					return
				}
				tt.property.SanitizeTmpl = tmpl
			}
			err := tt.property.Validate(properties, tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_StringPropertySubProperties(t *testing.T) {
	assert := assert.New(t)
	property := &StringProperty{}
	assert.Nil(property.SubProperties())
}
