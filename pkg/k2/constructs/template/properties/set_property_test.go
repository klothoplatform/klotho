package properties

import (
	"fmt"
	"testing"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/k2/constructs/template/property"
	"github.com/klothoplatform/klotho/pkg/knowledgebase"
	"github.com/klothoplatform/klotho/pkg/set"
	"github.com/stretchr/testify/assert"
)

func Test_SetProperty_SetProperty(t *testing.T) {
	tests := []struct {
		name      string
		property  *SetProperty
		input     any
		wantError bool
	}{
		{
			name: "valid set value",
			property: &SetProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			input: set.HashedSet[string, any]{
				Hasher: func(s any) string {
					return fmt.Sprintf("%v", s)
				},
				M: map[string]any{
					"item1": "item1",
					"item2": "item2",
				},
			},
			wantError: false,
		},
		{
			name: "invalid list value",
			property: &SetProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			input:     []any{"item1", "item2"},
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

func Test_SetProperty_ZeroValue(t *testing.T) {
	assert := assert.New(t)
	property := &SetProperty{}
	assert.Nil(property.ZeroValue())
}

// Testing the Details method
func Test_SetProperty_Details(t *testing.T) {
	assert := assert.New(t)
	property := &SetProperty{}
	assert.Same(&property.PropertyDetails, property.Details())
}

func Test_SetProperty_Clone(t *testing.T) {
	property := &SetProperty{
		PropertyDetails: property.PropertyDetails{Path: "test"},
		ItemProperty:    &StringProperty{},
	}
	clone := property.Clone()
	assert.Equal(t, property, clone)
}

func Test_SetProperty_AppendProperty(t *testing.T) {
	tests := []struct {
		name       string
		property   *SetProperty
		properties construct.Properties
		input      any
		wantError  bool
	}{
		{
			name: "append valid set value",
			property: &SetProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			properties: construct.Properties{},
			input: set.HashedSet[string, any]{
				Hasher: func(s any) string {
					return fmt.Sprintf("%v", s)
				},
				M: map[string]any{
					"item1": "item1",
					"item2": "item2",
				},
			},
			wantError: false,
		},
		{
			name: "append list value",
			property: &SetProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			properties: construct.Properties{},
			input:      []any{"item1", "item2"},
			wantError:  false,
		},
		{
			// This test documents a bug in the AppendProperty method
			name: "append invalid map value",
			property: &SetProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
			},
			properties: construct.Properties{},
			input:      map[string]any{"key": "value"},
			wantError:  false, // This should be true if the bug is fixed
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

func Test_RemoveSetProperty(t *testing.T) {
	tests := []struct {
		name       string
		property   *SetProperty
		properties construct.Properties
		value      any
		expected   set.HashedSet[string, any]
	}{
		{
			name: "existing property",
			properties: map[string]any{
				"test": set.HashedSet[string, any]{
					Hasher: func(s any) string {
						return fmt.Sprintf("%v", s)
					},
					M: map[string]any{
						"test2": "test2",
						"test1": "test1",
					},
				},
			},
			property: &SetProperty{
				PropertyDetails: property.PropertyDetails{
					Path: "test",
				},
			},
			value: set.HashedSet[string, any]{
				Hasher: func(s any) string {
					return fmt.Sprintf("%v", s)
				},
				M: map[string]any{
					"test2": "test2",
				},
			},
			expected: set.HashedSet[string, any]{
				Hasher: func(s any) string {
					return fmt.Sprintf("%v", s)
				},
				M: map[string]any{
					"test1": "test1",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			err := tt.property.RemoveProperty(tt.properties, tt.value)
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.expected.M, tt.properties[tt.property.Path].(set.HashedSet[string, any]).M)
		})
	}
}

func Test_SetParse(t *testing.T) {
	tests := []struct {
		name     string
		property *SetProperty
		ctx      knowledgebase.DynamicValueContext
		data     knowledgebase.DynamicValueData
		value    any
		expected set.HashedSet[string, any]
		wantErr  bool
	}{
		{
			name: "set property",
			property: &SetProperty{
				PropertyDetails: property.PropertyDetails{
					Path: "test",
				},
				ItemProperty: &StringProperty{},
			},
			value: []any{"test1", "test2"},
			expected: set.HashedSet[string, any]{
				Hasher: func(s any) string {
					return fmt.Sprintf("%v", s)
				},
				M: map[string]any{
					"test1": "test1",
					"test2": "test2",
				},
			},
		},
		{
			name: "set property as template",
			property: &SetProperty{
				PropertyDetails: property.PropertyDetails{
					Path: "test",
				},
				ItemProperty: &StringProperty{},
			},
			value: []any{"{{ \"test1\" }}", "{{ \"test2\" }}"},
			expected: set.HashedSet[string, any]{
				Hasher: func(s any) string {
					return fmt.Sprintf("%v", s)
				},
				M: map[string]any{
					"test1": "test1",
					"test2": "test2",
				},
			},
		},
		{
			name: "non set throws error",
			property: &SetProperty{
				PropertyDetails: property.PropertyDetails{
					Path: "test",
				},
			},
			value:   "test",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			ctx := DefaultExecutionContext{}
			actual, err := tt.property.Parse(tt.value, ctx, nil)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			assert.Equal(actual.(set.HashedSet[string, any]).M, tt.expected.M, "expected %v, got %v", tt.expected, actual)
		})
	}
}
func Test_SetProperty_Contains(t *testing.T) {
	tests := []struct {
		name     string
		property *SetProperty
		value    any
		expected bool
	}{
		{
			name: "set contains value",
			property: &SetProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
				ItemProperty:    &StringProperty{},
			},
			value: set.HashedSet[string, any]{
				Hasher: func(s any) string {
					return fmt.Sprintf("%v", s)
				},
				M: map[string]any{
					"test": "test",
				},
			},
			expected: true,
		},
		{
			name: "set does not contain value",
			property: &SetProperty{
				PropertyDetails: property.PropertyDetails{Path: "test"},
				ItemProperty:    &StringProperty{},
			},
			value: set.HashedSet[string, any]{
				Hasher: func(s any) string {
					return fmt.Sprintf("%v", s)
				},
				M: map[string]any{
					"other": "other",
				},
			},
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

func Test_SetProperty_Type(t *testing.T) {
	assert := assert.New(t)
	property := &SetProperty{}
	assert.Equal("set", property.Type())
	property2 := &SetProperty{
		ItemProperty: &StringProperty{},
	}
	assert.Equal("set(string)", property2.Type())
}
