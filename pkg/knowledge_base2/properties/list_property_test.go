package properties

import (
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/enginetesting"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_ListProperty_Set(t *testing.T) {
	tests := []struct {
		name     string
		property *ListProperty
		resource *construct.Resource
		value    []any
	}{
		{
			name:     "list property",
			resource: &construct.Resource{Properties: make(map[string]any)},
			property: &ListProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
			value: []any{"test"},
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

func Test_ListProperty_Append(t *testing.T) {
	tests := []struct {
		name     string
		property *ListProperty
		resource *construct.Resource
		value    any
		expect   any
	}{
		{
			name:     "list property",
			resource: &construct.Resource{Properties: make(map[string]any)},
			property: &ListProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
			value:  "test",
			expect: []any{"test"},
		},
		{
			name: "existing property",
			resource: &construct.Resource{
				Properties: map[string]any{
					"test": []any{"first"},
				},
			},
			property: &ListProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
			value:  "test",
			expect: []any{"first", "test"},
		},
		{
			name: "appends list of values",
			resource: &construct.Resource{
				Properties: map[string]any{
					"test": []any{"first"},
				},
			},
			property: &ListProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
				ItemProperty: &StringProperty{},
			},
			value:  []any{"test"},
			expect: []any{"first", "test"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			err := tt.property.AppendProperty(tt.resource, tt.value)
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.expect, tt.resource.Properties[tt.property.Path])
		})
	}
}

func Test_ListProperty_Remove(t *testing.T) {
	tests := []struct {
		name     string
		property *ListProperty
		resource *construct.Resource
		value    any
		expect   any
	}{
		{
			name:     "list property",
			resource: &construct.Resource{Properties: make(map[string]any)},
			property: &ListProperty{
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
					"test": []any{"first", "test"},
				},
			},
			property: &ListProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
			},
			value:  "test",
			expect: []any{"first"},
		},
		{
			name: "removes list of values",
			resource: &construct.Resource{
				Properties: map[string]any{
					"test": []any{"first", "test"},
				},
			},
			property: &ListProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
				ItemProperty: &StringProperty{},
			},
			value:  []any{"test"},
			expect: []any{"first"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			err := tt.property.RemoveProperty(tt.resource, tt.value)
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.expect, tt.resource.Properties[tt.property.Path])
		})
	}
}

func Test_ListProperty_Clone(t *testing.T) {
	assert := assert.New(t)
	property := &ListProperty{
		PropertyDetails: knowledgebase.PropertyDetails{
			Path: "test",
		},
		ItemProperty: &StringProperty{},
	}
	clone := property.Clone()
	assert.Equal(property, clone)
	assert.NotSame(property, clone)
	clonedListP := clone.(*ListProperty)
	assert.NotSame(property.ItemProperty, clonedListP.ItemProperty)
}

func Test_ListProperty_Details(t *testing.T) {
	assert := assert.New(t)
	property := &ListProperty{
		PropertyDetails: knowledgebase.PropertyDetails{
			Path: "test",
		},
		ItemProperty: &StringProperty{},
	}
	details := property.Details()
	assert.Same(&property.PropertyDetails, details)
}

func Test_ListProperty_GetDefaultValue(t *testing.T) {
	tests := []struct {
		name     string
		property *ListProperty
		expect   any
	}{
		{
			name: "list property",
			property: &ListProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
				SharedPropertyFields: SharedPropertyFields{
					DefaultValue: []any{"test"},
				},
				ItemProperty: &StringProperty{},
			},
			expect: []any{"test"},
		},
		{
			name: "list property with template value",
			property: &ListProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
				ItemProperty: &StringProperty{},
				SharedPropertyFields: SharedPropertyFields{
					DefaultValue: []any{"{{ \"test\" }}"},
				},
			},
			expect: []any{"test"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			ctx := knowledgebase.DynamicValueContext{}
			data := knowledgebase.DynamicValueData{}
			result, err := tt.property.GetDefaultValue(ctx, data)
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.expect, result)
		})
	}
}

func Test_ListProperty_Parse(t *testing.T) {
	tests := []struct {
		name     string
		property *ListProperty
		value    any
		expect   any
	}{
		{
			name: "list property",
			property: &ListProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
				ItemProperty: &StringProperty{},
			},
			value:  []any{"test"},
			expect: []any{"test"},
		},
		{
			name: "list property with template value",
			property: &ListProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
				ItemProperty: &StringProperty{},
			},
			value:  []any{"{{ \"test\" }}"},
			expect: []any{"test"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			ctx := knowledgebase.DynamicValueContext{}
			data := knowledgebase.DynamicValueData{}
			result, err := tt.property.Parse(tt.value, ctx, data)
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.expect, result)
		})
	}
}

func Test_ListProperty_Validate(t *testing.T) {
	minLength := 1
	maxLength := 2
	tests := []struct {
		name          string
		property      *ListProperty
		testResources []*construct.Resource
		mockKBCalls   []mock.Call
		value         any
		wantErr       bool
	}{
		{
			name: "list property",
			property: &ListProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
				MinLength:    &minLength,
				MaxLength:    &maxLength,
				ItemProperty: &StringProperty{},
			},
			value: []any{"test"},
		},
		{
			name: "list property over max length",
			property: &ListProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
				MinLength:    &minLength,
				MaxLength:    &maxLength,
				ItemProperty: &StringProperty{},
			},
			value:   []any{"test", "test", "test"},
			wantErr: true,
		},
		{
			name: "list property under min length",
			property: &ListProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
				MinLength:    &minLength,
				MaxLength:    &maxLength,
				ItemProperty: &StringProperty{},
			},
			value:   []any{},
			wantErr: true,
		},
		{
			name: "list property checks item property validation",
			property: &ListProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
				MinLength:    &minLength,
				MaxLength:    &maxLength,
				ItemProperty: &StringProperty{AllowedValues: []string{"val"}},
			},
			value:   []any{"test", "test", "test"},
			wantErr: true,
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
			err := tt.property.Validate(resource, tt.value, ctx)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)
		})
	}
}

func Test_ListProperty_Contains(t *testing.T) {
	tests := []struct {
		name     string
		property *ListProperty
		value    any
		expect   bool
	}{
		{
			name: "list property",
			property: &ListProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
				ItemProperty: &StringProperty{},
			},
			value:  []any{"test"},
			expect: true,
		},
		{
			name: "list property does not conatin",
			property: &ListProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Path: "test",
				},
				ItemProperty: &StringProperty{},
			},
			value: []any{"tttt"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			result := tt.property.Contains(tt.value, "test")
			assert.Equal(tt.expect, result)
		})
	}
}

func Test_ListProperty_ZeroValue(t *testing.T) {
	assert := assert.New(t)
	property := &ListProperty{}
	assert.Nil(property.ZeroValue())
}

func Test_ListProperty_Type(t *testing.T) {
	assert := assert.New(t)
	property := &ListProperty{}
	assert.Equal("list", property.Type())
	property2 := &ListProperty{
		ItemProperty: &StringProperty{},
	}
	assert.Equal("list(string)", property2.Type())
}

func Test_ListProperty_SubProperties(t *testing.T) {
	assert := assert.New(t)
	property := &ListProperty{
		Properties: make(knowledgebase.Properties),
	}
	assert.NotNil(property.SubProperties())
}
