package properties

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/k2/constructs/template/property"
	"github.com/stretchr/testify/assert"
)

func Test_KeyValueListProperty_SetProperty(t *testing.T) {
	tests := []struct {
		name      string
		property  *KeyValueListProperty
		input     any
		expected  []any
		wantError bool
	}{
		{
			name: "set property with map input",
			property: &KeyValueListProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
				KeyProperty:     &StringProperty{PropertyDetails: property.PropertyDetails{Name: "key"}},
				ValueProperty:   &StringProperty{PropertyDetails: property.PropertyDetails{Name: "value"}},
			},
			input: map[string]any{"key1": "value1", "key2": "value2"},
			expected: []any{
				map[string]any{"key": "key1", "value": "value1"},
				map[string]any{"key": "key2", "value": "value2"},
			},
			wantError: false,
		},
		{
			name: "set property with list input",
			property: &KeyValueListProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
				KeyProperty:     &StringProperty{PropertyDetails: property.PropertyDetails{Name: "key"}},
				ValueProperty:   &StringProperty{PropertyDetails: property.PropertyDetails{Name: "value"}},
			},
			input: []any{
				map[string]any{"key": "key1", "value": "value1"},
				map[string]any{"key": "key2", "value": "value2"},
			},
			expected: []any{
				map[string]any{"key": "key1", "value": "value1"},
				map[string]any{"key": "key2", "value": "value2"},
			},
			wantError: false,
		},
		{
			name: "set property with invalid input",
			property: &KeyValueListProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
				KeyProperty:     &StringProperty{PropertyDetails: property.PropertyDetails{Name: "key"}},
				ValueProperty:   &StringProperty{PropertyDetails: property.PropertyDetails{Name: "value"}},
			},
			input:     "invalid input",
			expected:  nil,
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
				result, _ := properties.GetProperty(tt.property.Path)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func Test_KeyValueListProperty_AppendProperty(t *testing.T) {
	tests := []struct {
		name      string
		property  *KeyValueListProperty
		initial   []any
		input     any
		expected  []any
		wantError bool
	}{
		{
			name: "append to existing list",
			property: &KeyValueListProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
				KeyProperty:     &StringProperty{PropertyDetails: property.PropertyDetails{Name: "key"}},
				ValueProperty:   &StringProperty{PropertyDetails: property.PropertyDetails{Name: "value"}},
			},
			initial: []any{
				map[string]any{"key": "key1", "value": "value1"},
			},
			input: map[string]any{"key2": "value2"},
			expected: []any{
				map[string]any{"key": "key1", "value": "value1"},
				map[string]any{"key": "key2", "value": "value2"},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			properties := construct.Properties{}
			properties.SetProperty(tt.property.Path, tt.initial)
			err := tt.property.AppendProperty(properties, tt.input)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				result, _ := properties.GetProperty(tt.property.Path)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func Test_KeyValueListProperty_RemoveProperty(t *testing.T) {
	tests := []struct {
		name      string
		property  *KeyValueListProperty
		initial   []any
		input     any
		expected  []any
		wantError bool
	}{
		{
			name: "remove existing key-value pair",
			property: &KeyValueListProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
				KeyProperty:     &StringProperty{PropertyDetails: property.PropertyDetails{Name: "key"}},
				ValueProperty:   &StringProperty{PropertyDetails: property.PropertyDetails{Name: "value"}},
			},
			initial: []any{
				map[string]any{"key": "key1", "value": "value1"},
				map[string]any{"key": "key2", "value": "value2"},
			},
			input: map[string]any{"key1": "value1"},
			expected: []any{
				map[string]any{"key": "key2", "value": "value2"},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			properties := construct.Properties{}
			properties.SetProperty(tt.property.Path, tt.initial)
			err := tt.property.RemoveProperty(properties, tt.input)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				result, _ := properties.GetProperty(tt.property.Path)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func Test_KeyValueListProperty_GetDefaultValue(t *testing.T) {
	tests := []struct {
		name          string
		property      *KeyValueListProperty
		expectedValue any
		wantError     bool
	}{
		{
			name: "return default value",
			property: &KeyValueListProperty{
				PropertyDetails: property.PropertyDetails{
					Path: "test",
				},
				SharedPropertyFields: SharedPropertyFields{
					DefaultValue: map[string]any{"defaultKey": "defaultValue"},
				},
				KeyProperty:   &StringProperty{PropertyDetails: property.PropertyDetails{Name: "key"}},
				ValueProperty: &StringProperty{PropertyDetails: property.PropertyDetails{Name: "value"}},
			},
			expectedValue: []any{map[string]any{"key": "defaultKey", "value": "defaultValue"}},
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

func Test_KeyValueListProperty_Parse(t *testing.T) {
	tests := []struct {
		name          string
		property      *KeyValueListProperty
		input         any
		expectedValue any
		wantError     bool
	}{
		{
			name: "parse valid input",
			property: &KeyValueListProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
				KeyProperty:     &StringProperty{PropertyDetails: property.PropertyDetails{Name: "key"}},
				ValueProperty:   &IntProperty{PropertyDetails: property.PropertyDetails{Name: "value"}},
			},
			input: []any{
				map[string]any{"key": "key1", "value": "42"},
				map[string]any{"key": "key2", "value": "24"},
			},
			expectedValue: []any{
				map[string]any{"key": "key1", "value": 42},
				map[string]any{"key": "key2", "value": 24},
			},
			wantError: false,
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

func Test_KeyValueListProperty_Contains(t *testing.T) {
	tests := []struct {
		name     string
		property *KeyValueListProperty
		value    any
		contains any
		expected bool
	}{
		{
			name: "contains key-value pair",
			property: &KeyValueListProperty{
				KeyProperty:   &StringProperty{PropertyDetails: property.PropertyDetails{Name: "key"}},
				ValueProperty: &StringProperty{PropertyDetails: property.PropertyDetails{Name: "value"}},
			},
			value: []any{
				map[string]any{"key": "key1", "value": "value1"},
				map[string]any{"key": "key2", "value": "value2"},
			},
			contains: map[string]any{"key1": "value1"},
			expected: true,
		},
		{
			name: "does not contain key-value pair",
			property: &KeyValueListProperty{
				KeyProperty:   &StringProperty{PropertyDetails: property.PropertyDetails{Name: "key"}},
				ValueProperty: &StringProperty{PropertyDetails: property.PropertyDetails{Name: "value"}},
			},
			value: []any{
				map[string]any{"key": "key1", "value": "value1"},
				map[string]any{"key": "key2", "value": "value2"},
			},
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

func Test_KeyValueListProperty_Type(t *testing.T) {
	property := &KeyValueListProperty{
		KeyProperty:   &StringProperty{},
		ValueProperty: &IntProperty{},
	}
	assert.Equal(t, "keyvaluelist(string,int)", property.Type())
}

func Test_KeyValueListProperty_Validate(t *testing.T) {
	tests := []struct {
		name      string
		property  *KeyValueListProperty
		value     any
		wantError bool
	}{
		{
			name: "valid input",
			property: &KeyValueListProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
				KeyProperty:     &StringProperty{PropertyDetails: property.PropertyDetails{Name: "key"}},
				ValueProperty:   &IntProperty{PropertyDetails: property.PropertyDetails{Name: "value"}},
			},
			value: []any{
				map[string]any{"key": "key1", "value": 42},
				map[string]any{"key": "key2", "value": 24},
			},
			wantError: false,
		},
		{
			name: "invalid key",
			property: &KeyValueListProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
				KeyProperty:     &StringProperty{PropertyDetails: property.PropertyDetails{Name: "key"}},
				ValueProperty:   &IntProperty{PropertyDetails: property.PropertyDetails{Name: "value"}},
			},
			value: []any{
				map[string]any{"key": 42, "value": 42},
			},
			wantError: true,
		},
		{
			name: "invalid value",
			property: &KeyValueListProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
				KeyProperty:     &StringProperty{PropertyDetails: property.PropertyDetails{Name: "key"}},
				ValueProperty:   &IntProperty{PropertyDetails: property.PropertyDetails{Name: "value"}},
			},
			value: []any{
				map[string]any{"key": "key1", "value": "not an int"},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			properties := construct.Properties{}
			err := tt.property.Validate(properties, tt.value)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_KeyValueListProperty_Clone(t *testing.T) {
	original := &KeyValueListProperty{
		PropertyDetails: property.PropertyDetails{Path: "test"},
		KeyProperty:     &StringProperty{PropertyDetails: property.PropertyDetails{Name: "key"}},
		ValueProperty:   &IntProperty{PropertyDetails: property.PropertyDetails{Name: "value"}},
		MinLength:       ptr(1),
		MaxLength:       ptr(10),
	}

	clone := original.Clone().(*KeyValueListProperty)

	assert.Equal(t, original.PropertyDetails, clone.PropertyDetails)
	assert.Equal(t, original.KeyProperty.Type(), clone.KeyProperty.Type())
	assert.Equal(t, original.ValueProperty.Type(), clone.ValueProperty.Type())
	assert.Equal(t, *original.MinLength, *clone.MinLength)
	assert.Equal(t, *original.MaxLength, *clone.MaxLength)
	assert.NotSame(t, original.KeyProperty, clone.KeyProperty)
	assert.NotSame(t, original.ValueProperty, clone.ValueProperty)
}

func Test_KeyValueListProperty_SubProperties(t *testing.T) {
	property := &KeyValueListProperty{}
	assert.Nil(t, property.SubProperties())
}
