package properties

import (
	"fmt"
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine/enginetesting"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/klothoplatform/klotho/pkg/set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_SetSetProperty(t *testing.T) {
	tests := []struct {
		name     string
		property *SetProperty
		resource *construct.Resource
		value    any
	}{
		{
			name:     "set property",
			resource: &construct.Resource{Properties: make(map[string]any)},
			property: &SetProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
			value: set.HashedSet[string, any]{
				Hasher: func(s any) string {
					return s.(string)
				},
				M: map[string]any{
					"test1": "test1",
					"test2": "test2",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			err := tt.property.SetProperty(tt.resource, tt.value)
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.value.(set.HashedSet[string, any]).M, tt.resource.Properties[tt.property.Path].(set.HashedSet[string, any]).M)
		})
	}
}

func Test_AppendSetProperty(t *testing.T) {
	tests := []struct {
		name     string
		property *SetProperty
		resource *construct.Resource
		value    any
		expected set.HashedSet[string, any]
	}{
		{
			name:     "set property",
			resource: &construct.Resource{Properties: make(map[string]any)},
			property: &SetProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
			value: set.HashedSet[string, any]{
				Hasher: func(s any) string {
					return s.(string)
				},
				M: map[string]any{
					"test1": "test1",
					"test2": "test2",
				},
			},
			expected: set.HashedSet[string, any]{
				Hasher: func(s any) string {
					return s.(string)
				},
				M: map[string]any{
					"test1": "test1",
					"test2": "test2",
				},
			},
		},
		{
			name: "existing property",
			resource: &construct.Resource{
				Properties: map[string]any{
					"test": set.HashedSet[string, any]{
						Hasher: func(s any) string {
							return s.(string)
						},
						M: map[string]any{
							"test1": "test1",
						},
					},
				},
			},
			property: &SetProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
			value: set.HashedSet[string, any]{
				Hasher: func(s any) string {
					return s.(string)
				},
				M: map[string]any{
					"test2": "test2",
				},
			},
			expected: set.HashedSet[string, any]{
				Hasher: func(s any) string {
					return s.(string)
				},
				M: map[string]any{
					"test1": "test1",
					"test2": "test2",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			err := tt.property.AppendProperty(tt.resource, tt.value)
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.expected.M, tt.resource.Properties[tt.property.Path].(set.HashedSet[string, any]).M)
		})
	}
}

func Test_RemoveSetProperty(t *testing.T) {
	tests := []struct {
		name     string
		property *SetProperty
		resource *construct.Resource
		value    any
		expected set.HashedSet[string, any]
	}{
		{
			name: "existing property",
			resource: &construct.Resource{
				Properties: map[string]any{
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
			},
			property: &SetProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
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
			err := tt.property.RemoveProperty(tt.resource, tt.value)
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.expected.M, tt.resource.Properties[tt.property.Path].(set.HashedSet[string, any]).M)
		})
	}
}

func Test_SetDetails(t *testing.T) {
	assert := assert.New(t)
	property := &SetProperty{
		PropertyDetails: knowledgebase.PropertyDetails{
			Path: "test",
		},
	}
	assert.Equal(property.Details(), &property.PropertyDetails)
}

func Test_SetClone(t *testing.T) {
	assert := assert.New(t)
	property := &SetProperty{
		PropertyDetails: knowledgebase.PropertyDetails{
			Path: "test",
		},
	}
	clone := property.Clone()
	assert.Equal(clone, property)
}

func Test_SetGetDefaultValue(t *testing.T) {
	tests := []struct {
		name     string
		property *SetProperty
		ctx      knowledgebase.DynamicValueContext
		data     knowledgebase.DynamicValueData
		expected set.HashedSet[string, any]
		wantErr  bool
	}{
		{
			name: "set property",
			property: &SetProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
				SharedPropertyFields: SharedPropertyFields{
					DefaultValue: []any{"test1", "test2"},
				},
				ItemProperty: &StringProperty{},
			},
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
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
				SharedPropertyFields: SharedPropertyFields{
					DefaultValue: []any{"{{ \"test1\" }}", "{{ \"test2\" }}"},
				},
				ItemProperty: &StringProperty{},
			},
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
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
				SharedPropertyFields: SharedPropertyFields{
					DefaultValue: "test",
				},
				ItemProperty: &StringProperty{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			ctx := knowledgebase.DynamicValueContext{}
			actual, err := tt.property.GetDefaultValue(ctx, knowledgebase.DynamicValueData{})
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
				PropertyDetails: knowledgebase.PropertyDetails{
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
				PropertyDetails: knowledgebase.PropertyDetails{
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
				PropertyDetails: knowledgebase.PropertyDetails{
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
			ctx := knowledgebase.DynamicValueContext{}
			actual, err := tt.property.Parse(tt.value, ctx, knowledgebase.DynamicValueData{})
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

func Test_SetZeroValue(t *testing.T) {
	assert := assert.New(t)
	property := &SetProperty{}
	assert.Nil(property.ZeroValue())
}

func Test_SetType(t *testing.T) {
	assert := assert.New(t)
	property := &SetProperty{}
	assert.Equal(property.Type(), "set")
	property = &SetProperty{
		ItemProperty: &StringProperty{},
	}
	assert.Equal(property.Type(), "set(string)")
}

func Test_SetContain(t *testing.T) {
	tests := []struct {
		name     string
		property *SetProperty
		value    set.HashedSet[string, any]
		contains any
		expected bool
	}{
		{
			name: "set property",
			property: &SetProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
			value: set.HashedSet[string, any]{
				Hasher: func(s any) string {
					return fmt.Sprintf("%v", s)
				},
				M: map[string]any{
					"test1": "test1",
					"test2": "test2",
				},
			},
			contains: "test1",
			expected: true,
		},
		{
			name: "contains is in set",
			property: &SetProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
			value: set.HashedSet[string, any]{
				Hasher: func(s any) string {
					return fmt.Sprintf("%v", s)
				},
				M: map[string]any{
					"test1": "test1",
					"test2": "test2",
				},
			},
			contains: "test3",
			expected: false,
		},
		{
			name: "non set throws error",
			property: &SetProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
			value: set.HashedSet[string, any]{
				Hasher: func(s any) string {
					return fmt.Sprintf("%v", s)
				},
				M: map[string]any{
					"test1": "test1",
					"test2": "test2",
				},
			},
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

func Test_SetValidate(t *testing.T) {
	minLength := 1
	maxLength := 2
	tests := []struct {
		name          string
		property      *SetProperty
		testResources []*construct.Resource
		mockKBCalls   []mock.Call
		value         any
		expected      bool
	}{
		{
			name: "set property",
			property: &SetProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
				MinLength:    &minLength,
				MaxLength:    &maxLength,
				ItemProperty: &StringProperty{},
			},
			value: set.HashedSet[string, any]{
				Hasher: func(s any) string {
					return fmt.Sprintf("%v", s)
				},
				M: map[string]any{
					"test1": "test1",
					"test2": "test2",
				},
			},
			expected: true,
		},

		{
			name: "set property under min length",
			property: &SetProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
				MinLength:    &minLength,
				MaxLength:    &maxLength,
				ItemProperty: &StringProperty{},
			},
			value: set.HashedSet[string, any]{
				Hasher: func(s any) string {
					return fmt.Sprintf("%v", s)
				},
				M: map[string]any{},
			},
			expected: false,
		},

		{
			name: "set property over max length",
			property: &SetProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
				MinLength:    &minLength,
				MaxLength:    &maxLength,
				ItemProperty: &StringProperty{},
			},
			value: set.HashedSet[string, any]{
				Hasher: func(s any) string {
					return fmt.Sprintf("%v", s)
				},
				M: map[string]any{
					"test1": "test1",
					"test2": "test2",
					"test3": "test3",
				},
			},
			expected: false,
		},
		{
			name: "set checks item property",
			property: &SetProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
				ItemProperty: &StringProperty{
					AllowedValues: []string{"test2"},
				},
			},
			value: set.HashedSet[string, any]{
				Hasher: func(s any) string {
					return fmt.Sprintf("%v", s)
				},
				M: map[string]any{
					"test1": "test1",
					"test2": "test2",
				},
			},
			expected: false,
		},
		{
			name: "non set throws error",
			property: &SetProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
				ItemProperty: &StringProperty{},
			},
			value:    "test",
			expected: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			resource := &construct.Resource{}
			graph := construct.NewGraph()
			for _, r := range tt.testResources {
				graph.AddVertex(r)
			}
			mockKB := &enginetesting.MockKB{}
			for _, call := range tt.mockKBCalls {
				mockKB.On(call.Method, call.Arguments...).Return(call.ReturnArguments...)
			}
			ctx := knowledgebase.DynamicValueContext{
				Graph:         graph,
				KnowledgeBase: mockKB,
			}
			actual := tt.property.Validate(resource, tt.value, ctx)
			if tt.expected {
				assert.NoError(actual)
			} else {
				assert.Error(actual)
			}
		})
	}
}

func Test_SetSubProperties(t *testing.T) {
	assert := assert.New(t)
	property := &SetProperty{
		Properties: knowledgebase.Properties{
			"test": &StringProperty{},
		},
	}
	assert.Len(property.SubProperties(), 1)
}
