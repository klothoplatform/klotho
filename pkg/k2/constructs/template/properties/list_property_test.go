package properties

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/k2/constructs/template/property"
	"github.com/stretchr/testify/assert"
)

// Testing the SetProperty method for different cases
func Test_ListProperty_SetProperty(t *testing.T) {
	tests := []struct {
		name      string
		property  *ListProperty
		input     any
		wantError bool
	}{
		{
			name: "valid list value",
			property: &ListProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			input:     []any{"item1", "item2"},
			wantError: false,
		},
		{
			name: "invalid map value",
			property: &ListProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			input:     map[string]any{"key": "value"},
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
func Test_ListProperty_ZeroValue(t *testing.T) {
	assert := assert.New(t)
	property := &ListProperty{}
	assert.Nil(property.ZeroValue())
}

// Testing the Details method
func Test_ListProperty_Details(t *testing.T) {
	assert := assert.New(t)
	property := &ListProperty{}
	assert.Same(&property.PropertyDetails, property.Details())
}

// Testing the Clone method
func Test_ListProperty_Clone(t *testing.T) {
	property := &ListProperty{
		PropertyDetails: property.PropertyDetails{Path: "test"},
		ItemProperty:    &StringProperty{},
	}
	clone := property.Clone()
	assert.Equal(t, property, clone)
}

// Testing the AppendProperty method with different cases
func Test_ListProperty_AppendProperty(t *testing.T) {
	tests := []struct {
		name       string
		property   *ListProperty
		properties construct.Properties
		input      any
		wantError  bool
		expect     any
	}{
		{
			name: "append valid list value",
			property: &ListProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			properties: construct.Properties{},
			input:      []any{"item1", "item2"},
			expect:     []any{"item1", "item2"},
		},
		{
			name: "append string value",
			property: &ListProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			properties: construct.Properties{},
			input:      "item1",
			expect:     []any{"item1"},
		},
		{
			name: "append to existing list",
			property: &ListProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			properties: construct.Properties{"test": []any{"item1"}},
			input:      "item2",
			expect:     []any{"item1", "item2"},
		},
		{
			// This test documents existing non-ideal behavior
			name: "append allows invalid values",
			property: &ListProperty{
				ItemProperty:    &StringProperty{},
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			properties: construct.Properties{},
			input:      map[string]any{"key": "value"},
			expect:     []any{map[string]any{"key": "value"}},
		},
	}

	for _, tt := range tests {
		assert := assert.New(t)
		t.Run(tt.name, func(t *testing.T) {
			err := tt.property.AppendProperty(tt.properties, tt.input)
			if tt.wantError {
				assert.Error(err)
				return
			}
			assert.NoError(err)
			assert.Equal(tt.expect, tt.properties[tt.property.Path])
		})
	}
}

// Testing the RemoveProperty method
func Test_ListProperty_RemoveProperty(t *testing.T) {
	tests := []struct {
		name       string
		property   *ListProperty
		properties construct.Properties
		input      any
		wantError  bool
	}{
		{
			name: "remove existing list property",
			property: &ListProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			properties: construct.Properties{"test": []any{"item1", "item2"}},
			input:      "item1",
			wantError:  false,
		},
		{
			name: "remove non-existent property",
			property: &ListProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			properties: construct.Properties{},
			input:      "item1",
			wantError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.property.RemoveProperty(tt.properties, tt.input)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Testing the Parse method
func Test_ListProperty_Parse(t *testing.T) {
	tests := []struct {
		name      string
		property  *ListProperty
		input     any
		expected  any
		wantError bool
	}{
		{
			name: "parse string to list",
			property: &ListProperty{
				ItemProperty: &StringProperty{},
			},
			input:     "[\"item1\", \"item2\"]",
			expected:  []any{"item1", "item2"},
			wantError: false,
		},
		{
			name: "parse valid list",
			property: &ListProperty{
				ItemProperty: &StringProperty{},
			},
			input:     []any{"item1", "item2"},
			expected:  []any{"item1", "item2"},
			wantError: false,
		},
		{
			name: "parse invalid map",
			property: &ListProperty{
				ItemProperty: &StringProperty{},
			},
			input:     map[string]any{"key": "value"},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.property.Parse(tt.input, DefaultExecutionContext{}, nil)
			if tt.wantError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Testing the Contains method
func Test_ListProperty_Contains(t *testing.T) {
	tests := []struct {
		name     string
		property *ListProperty
		value    any
		expected bool
	}{
		{
			name: "list contains value",
			property: &ListProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
				ItemProperty:    &StringProperty{},
			},
			value:    []any{"test"},
			expected: true,
		},
		{
			name: "list does not contain value",
			property: &ListProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
				ItemProperty:    &StringProperty{},
			},
			value:    []any{"other"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.property.Contains(tt.value, "test")
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Testing the Type method
func Test_ListProperty_Type(t *testing.T) {
	assert := assert.New(t)
	property := &ListProperty{}
	assert.Equal("list", property.Type())
	property2 := &ListProperty{
		ItemProperty: &StringProperty{},
	}
	assert.Equal("list(string)", property2.Type())
}

// Testing the SubProperties method
func Test_ListProperty_SubProperties(t *testing.T) {
	assert := assert.New(t)
	property := &ListProperty{
		Properties: make(property.PropertyMap),
	}
	assert.NotNil(property.SubProperties())
}
