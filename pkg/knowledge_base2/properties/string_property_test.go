package properties

import (
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/stretchr/testify/assert"
)

func Test_SetStringProperty(t *testing.T) {
	tests := []struct {
		name     string
		property *StringProperty
		resource *construct.Resource
		value    string
	}{
		{
			name:     "string property",
			resource: &construct.Resource{Properties: make(map[string]any)},
			property: &StringProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
			value: "test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			err := tt.property.SetProperty(tt.resource, tt.value)
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.value, tt.resource.Properties[tt.property.Path])
		})
	}
}

func Test_AppendStringProperty(t *testing.T) {
	tests := []struct {
		name     string
		property *StringProperty
		resource *construct.Resource
		value    string
	}{
		{
			name:     "string property",
			resource: &construct.Resource{Properties: make(map[string]any)},
			property: &StringProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
			value: "test",
		},
		{
			name: "existing property",
			resource: &construct.Resource{
				Properties: map[string]any{
					"test": "test",
				},
			},
			property: &StringProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
			value: "test2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			err := tt.property.AppendProperty(tt.resource, tt.value)
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.value, tt.resource.Properties[tt.property.Path])
		})
	}
}

func Test_RemoveStringProperty(t *testing.T) {
	tests := []struct {
		name     string
		property *StringProperty
		resource *construct.Resource
		value    string
	}{
		{
			name:     "string property",
			resource: &construct.Resource{Properties: make(map[string]any)},
			property: &StringProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
			value: "test",
		},
		{
			name: "existing property",
			resource: &construct.Resource{
				Properties: map[string]any{
					"test": "test",
				},
			},
			property: &StringProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
			value: "test2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			err := tt.property.RemoveProperty(tt.resource, tt.value)
			if !assert.NoError(err) {
				return
			}
			assert.Empty(tt.resource.Properties)
		})
	}
}

func Test_StringDetails(t *testing.T) {
	tests := []struct {
		name     string
		property *StringProperty
	}{
		{
			name: "string property",
			property: &StringProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			details := tt.property.Details()
			assert.Equal(tt.property.Path, details.Path)
		})
	}
}

func Test_StringClone(t *testing.T) {
	tests := []struct {
		name     string
		property *StringProperty
	}{
		{
			name: "string property",
			property: &StringProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			clone := tt.property.Clone()
			assert.Equal(tt.property.Path, clone.Details().Path)
		})
	}
}

func Test_StringGetDefaultValue(t *testing.T) {
	tests := []struct {
		name     string
		property *StringProperty
		ctx      knowledgebase.DynamicValueContext
		data     knowledgebase.DynamicValueData
		value    any
	}{
		{
			name: "string property",
			property: &StringProperty{
				SharedPropertyFields: SharedPropertyFields{
					DefaultValue: "test",
				},
			},
			value: "test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			val, err := tt.property.GetDefaultValue(tt.ctx, tt.data)
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.value, val)
		})
	}
}

func Test_StringParse(t *testing.T) {
	tests := []struct {
		name     string
		property *StringProperty
		ctx      knowledgebase.DynamicValueContext
		data     knowledgebase.DynamicValueData
		value    any
		expected string
		wantErr  bool
	}{
		{
			name: "string property",
			property: &StringProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
			value:    "test",
			expected: "test",
		},
		{
			name: "string property as template",
			property: &StringProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
			value:    "{{ \"test\" }}",
			expected: "test",
		},
		{
			name: "non string throws error",
			property: &StringProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
			value:   1,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			ctx := knowledgebase.DynamicValueContext{}
			actual, err := tt.property.Parse(tt.value, ctx, knowledgebase.DynamicValueData{})
			if tt.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			assert.Equal(actual, tt.expected, "expected %s, got %s", tt.expected, actual)
		})
	}
}

func Test_StringZeroValue(t *testing.T) {
	assert := assert.New(t)
	property := &StringProperty{}
	assert.Equal(property.ZeroValue(), "")
}

func Test_StringType(t *testing.T) {
	assert := assert.New(t)
	property := &StringProperty{}
	assert.Equal(property.Type(), "string")
}

func Test_StringContain(t *testing.T) {
	tests := []struct {
		name     string
		property *StringProperty
		value    any
		contains any
		expected bool
	}{
		{
			name: "string property",
			property: &StringProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
			value:    "test",
			contains: "test",
			expected: true,
		},
		{
			name: "contains is substring",
			property: &StringProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
			value:    "test2",
			contains: "test",
			expected: true,
		},
		{
			name: "val is substring",
			property: &StringProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
			value:    "test",
			contains: "test2",
			expected: false,
		},
		{
			name: "non string throws error",
			property: &StringProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
			value:    1,
			contains: 1,
			expected: false,
		},
		{
			name: "non string throws error",
			property: &StringProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
			value:    "test",
			contains: 1,
			expected: false,
		},
		{
			name: "non string throws error",
			property: &StringProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
			value:    1,
			contains: "test",
			expected: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			actual := tt.property.Contains(tt.value, tt.contains)
			assert.Equal(actual, tt.expected, "expected %v, got %v", tt.expected, actual)
		})
	}
}

func Test_StringValidate(t *testing.T) {
	tests := []struct {
		name             string
		property         *StringProperty
		sanitizeTemplate string
		value            any
		expected         bool
	}{
		{
			name: "string property",
			property: &StringProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
			value:    "test",
			expected: true,
		},
		{
			name: "string property with allowed values",
			property: &StringProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
				AllowedValues: []string{"test", "test2"},
			},
			value:    "test",
			expected: true,
		},
		{
			name: "string is sanitized",
			property: &StringProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
			sanitizeTemplate: "{{ . | upper }}",
			value:            "TEST",
			expected:         true,
		},
		{
			name: "string not in allowed values",
			property: &StringProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
				AllowedValues: []string{"test2"},
			},
			value:    "test",
			expected: false,
		},
		{
			name: "string not sanitized",
			property: &StringProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
			sanitizeTemplate: "{{ . | upper }}",
			value:            "test",
			expected:         false,
		},
		{
			name: "non string throws error",
			property: &StringProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
			value:    1,
			expected: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			if tt.sanitizeTemplate != "" {
				tmpl, err := knowledgebase.NewSanitizationTmpl(tt.name, tt.sanitizeTemplate)
				if !assert.NoError(err) {
					return
				}
				tt.property.SanitizeTmpl = tmpl
			}
			actual := tt.property.Validate(tt.value, construct.Properties{})
			if tt.expected {
				assert.NoError(actual)
			} else {
				assert.Error(actual)
			}
		})
	}
}

func Test_StringSubProperties(t *testing.T) {
	assert := assert.New(t)
	property := &StringProperty{}
	assert.Nil(property.SubProperties())
}
