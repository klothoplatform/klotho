package properties

import (
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/k2/constructs/template/property"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_BoolProppertySetProperty(t *testing.T) {
	tests := []struct {
		name      string
		property  *BoolProperty
		input     any
		wantError bool
	}{
		{
			name: "valid bool value",
			property: &BoolProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			input:     true,
			wantError: false,
		},
		{
			name: "invalid value type",
			property: &BoolProperty{
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

func Test_BoolPropertyZeroValue(t *testing.T) {
	assert := assert.New(t)
	property := &BoolProperty{}
	assert.Equal(false, property.ZeroValue())
}

func Test_BoolPropertyDetails(t *testing.T) {
	assert := assert.New(t)
	property := &BoolProperty{}
	assert.Same(&property.PropertyDetails, property.Details())
}

func Test_BoolPropertyContains(t *testing.T) {
	assert := assert.New(t)
	property := &BoolProperty{}
	assert.False(property.Contains(nil, nil))
}

func Test_BoolPropertyType(t *testing.T) {
	assert := assert.New(t)
	property := &BoolProperty{}
	assert.Equal("bool", property.Type())
}

func Test_BoolPropertyValidate(t *testing.T) {
	tests := []struct {
		name       string
		property   *BoolProperty
		properties construct.Properties
		value      any
		wantErr    bool
	}{
		{
			name: "bool property",
			property: &BoolProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			value:   true,
			wantErr: false,
		},
		{
			name: "invalid value",
			property: &BoolProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			value:   1,
			wantErr: true,
		},
		{
			name: "nil value for required property",
			property: &BoolProperty{
				PropertyDetails: property.PropertyDetails{Path: "test", Required: true},
			},
			value:   nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.property.Validate(tt.properties, tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_BoolPropertySubProperties(t *testing.T) {
	assert := assert.New(t)
	property := &BoolProperty{}
	assert.Nil(property.SubProperties())
}
