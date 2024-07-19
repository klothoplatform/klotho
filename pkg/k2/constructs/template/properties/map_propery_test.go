package properties

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/k2/constructs/template/property"
	"github.com/stretchr/testify/assert"
)

func Test_MapProperty_SetProperty(t *testing.T) {
	tests := []struct {
		name      string
		property  *MapProperty
		input     any
		wantError bool
	}{
		{
			name: "valid map value",
			property: &MapProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			input:     map[string]any{"key1": "value1", "key2": "value2"},
			wantError: false,
		},
		{
			name: "invalid value type",
			property: &MapProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			input:     "invalid_value",
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

func Test_MapProperty_ZeroValue(t *testing.T) {
	assert := assert.New(t)
	property := &MapProperty{}
	assert.Equal(nil, property.ZeroValue())
}

func Test_MapProperty_Details(t *testing.T) {
	assert := assert.New(t)
	property := &MapProperty{}
	assert.Same(&property.PropertyDetails, property.Details())
}

func Test_MapProperty_Clone(t *testing.T) {
	property := &MapProperty{
		PropertyDetails: property.PropertyDetails{Path: "test"},
		MinLength:       new(int),
		MaxLength:       new(int),
	}
	clone := property.Clone()
	assert.Equal(t, property, clone)
}

func Test_MapProperty_Validate(t *testing.T) {
	tests := []struct {
		name      string
		property  *MapProperty
		value     any
		wantErr   bool
		minLength int
		maxLength int
	}{
		{
			name: "valid map length within range",
			property: &MapProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
				MaxLength:       ptr(2),
			},
			value:   map[string]any{"key1": "value1"},
			wantErr: false,
		},
		{
			name: "map length too short",
			property: &MapProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
				MinLength:       ptr(1),
			},
			value:   map[string]any{},
			wantErr: true,
		},
		{
			name: "map length too long",
			property: &MapProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
				MaxLength:       ptr(1),
			},
			value:   map[string]any{"key1": "value1", "key2": "value2", "key3": "value3"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			properties := construct.Properties{}
			err := tt.property.Validate(properties, tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_MapProperty_SubProperties(t *testing.T) {
	assert := assert.New(t)
	property := &MapProperty{}
	assert.Nil(property.SubProperties())
}

func Test_MapProperty_AppendProperty(t *testing.T) {
	tests := []struct {
		name      string
		property  *MapProperty
		initial   map[string]any
		input     any
		expected  map[string]any
		wantError bool
	}{
		{
			name: "append to existing map",
			property: &MapProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			initial:   map[string]any{"existing": "value"},
			input:     map[string]any{"new": "appended"},
			expected:  map[string]any{"existing": "value", "new": "appended"},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			properties := construct.Properties{}
			properties.SetProperty("test", tt.initial)
			err := tt.property.AppendProperty(properties, tt.input)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				result, _ := properties.GetProperty("test")
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func Test_MapProperty_RemoveProperty(t *testing.T) {
	tests := []struct {
		name      string
		property  *MapProperty
		initial   map[string]any
		input     any
		expected  map[string]any
		wantError bool
	}{
		{
			name: "remove existing key",
			property: &MapProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			initial:   map[string]any{"key1": "value1", "key2": "value2"},
			input:     map[string]any{"key1": "value1"},
			expected:  map[string]any{"key2": "value2"},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			properties := construct.Properties{}
			properties.SetProperty("test", tt.initial)
			err := tt.property.RemoveProperty(properties, tt.input)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				result, _ := properties.GetProperty("test")
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func Test_MapProperty_GetDefaultValue(t *testing.T) {
	tests := []struct {
		name          string
		property      *MapProperty
		expectedValue any
		wantError     bool
	}{
		{
			name: "return default value",
			property: &MapProperty{
				Properties: map[string]property.Property{
					"default": &StringProperty{
						PropertyDetails: property.PropertyDetails{Path: ".default", Name: "default"},
					}},
				SharedPropertyFields: SharedPropertyFields{DefaultValue: map[string]any{"default": "value"}},
			},
			expectedValue: map[string]any{"default": "value"},
			wantError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := DefaultExecutionContext{}
			result, err := tt.property.GetDefaultValue(ctx, nil)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedValue, result)
			}
		})
	}
}

func Test_MapProperty_Parse(t *testing.T) {
	tests := []struct {
		name          string
		property      *MapProperty
		input         any
		expectedValue any
		wantError     bool
	}{
		{
			name: "parse with sub-properties",
			property: &MapProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
				Properties: property.PropertyMap{
					"subKey": &StringProperty{PropertyDetails: property.PropertyDetails{Path: "subKey"}},
				},
			},
			input:         map[string]any{"subKey": "value"},
			expectedValue: map[string]any{"subKey": "value"},
			wantError:     false,
		},
		{
			name: "parse with key and value properties",
			property: &MapProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
				KeyProperty:     &StringProperty{},
				ValueProperty:   &IntProperty{},
			},
			input:         map[string]any{"key": "42"},
			expectedValue: map[string]any{"key": 42},
			wantError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := DefaultExecutionContext{}
			result, err := tt.property.Parse(tt.input, ctx, nil)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedValue, result)
			}
		})
	}
}

func Test_MapProperty_Contains(t *testing.T) {
	tests := []struct {
		name     string
		property *MapProperty
		value    any
		contains any
		expected bool
	}{
		{
			name:     "contains key-value pair",
			property: &MapProperty{},
			value:    map[string]any{"key1": "value1", "key2": "value2"},
			contains: map[string]any{"key1": "value1"},
			expected: true,
		},
		{
			name:     "does not contain key-value pair",
			property: &MapProperty{},
			value:    map[string]any{"key1": "value1", "key2": "value2"},
			contains: map[string]any{"key3": "value3"},
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

func Test_MapProperty_Type(t *testing.T) {
	tests := []struct {
		name     string
		property *MapProperty
		expected string
	}{
		{
			name: "type with key and value properties",
			property: &MapProperty{
				KeyProperty:   &StringProperty{},
				ValueProperty: &IntProperty{},
			},
			expected: "map(string,int)",
		},
		{
			name:     "type without key and value properties",
			property: &MapProperty{},
			expected: "map",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.property.Type()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func Test_MapProperty_Key(t *testing.T) {
	keyProp := &StringProperty{}
	prop := &MapProperty{KeyProperty: keyProp}
	assert.Same(t, keyProp, prop.Key())
}

func Test_MapProperty_Value(t *testing.T) {
	valueProp := &IntProperty{}
	prop := &MapProperty{ValueProperty: valueProp}
	assert.Same(t, valueProp, prop.Value())
}

func ptr[T any](v T) *T {
	return &v
}
