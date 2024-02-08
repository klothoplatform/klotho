package properties

import (
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct"
	knowledgebase2 "github.com/klothoplatform/klotho/pkg/knowledgebase"
	"github.com/stretchr/testify/assert"
)

func Test_SetResourceProperty(t *testing.T) {
	tests := []struct {
		name     string
		property *ResourceProperty
		resource *construct.Resource
		value    construct.ResourceId
	}{
		{
			name:     "resource property",
			resource: &construct.Resource{Properties: make(map[string]any)},
			property: &ResourceProperty{
				PropertyDetails: knowledgebase2.PropertyDetails{
					Path: "test",
				},
			},
			value: construct.ResourceId{Provider: "test"},
		},
		{
			name:     "existing resource property",
			resource: &construct.Resource{Properties: map[string]any{"test": construct.ResourceId{Provider: "first"}}},
			property: &ResourceProperty{
				PropertyDetails: knowledgebase2.PropertyDetails{
					Path: "test",
				},
			},
			value: construct.ResourceId{Provider: "test"},
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

func Test_AppendResourceProperty(t *testing.T) {
	tests := []struct {
		name     string
		property *ResourceProperty
		resource *construct.Resource
		value    construct.ResourceId
	}{
		{
			name:     "resource property",
			resource: &construct.Resource{Properties: make(map[string]any)},
			property: &ResourceProperty{
				PropertyDetails: knowledgebase2.PropertyDetails{
					Path: "test",
				},
			},
			value: construct.ResourceId{Provider: "test"},
		},
		{
			name:     "existing resource property",
			resource: &construct.Resource{Properties: map[string]any{"test": construct.ResourceId{Provider: "first"}}},
			property: &ResourceProperty{
				PropertyDetails: knowledgebase2.PropertyDetails{
					Path: "test",
				},
			},
			value: construct.ResourceId{Provider: "test"},
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

func Test_RemoveResourceProperty(t *testing.T) {
	tests := []struct {
		name     string
		property *ResourceProperty
		resource *construct.Resource
		value    construct.ResourceId
		wantErr  bool
	}{
		{
			name:     "resource property",
			resource: &construct.Resource{Properties: make(map[string]any)},
			property: &ResourceProperty{
				PropertyDetails: knowledgebase2.PropertyDetails{
					Path: "test",
				},
			},
			value: construct.ResourceId{Provider: "test"},
		},
		{
			name:     "existing property",
			resource: &construct.Resource{Properties: map[string]any{"test": construct.ResourceId{Provider: "first"}}},

			property: &ResourceProperty{
				PropertyDetails: knowledgebase2.PropertyDetails{
					Path: "test",
				},
			},
			value:   construct.ResourceId{Provider: "test"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			err := tt.property.RemoveProperty(tt.resource, tt.value)

			if tt.wantErr {
				assert.Error(err)
				assert.NotNil(tt.resource.Properties[tt.property.Path])
			} else {
				if !assert.NoError(err) {
					return
				}
				assert.Nil(tt.resource.Properties[tt.property.Path])
			}
		})
	}
}

func Test_ResourceProperty_GetDefaultValue(t *testing.T) {
	tests := []struct {
		name     string
		property *ResourceProperty
		ctx      knowledgebase2.DynamicValueContext
		data     knowledgebase2.DynamicValueData
		want     construct.ResourceId
		wantErr  bool
	}{
		{
			name: "default value",
			property: &ResourceProperty{
				SharedPropertyFields: SharedPropertyFields{
					DefaultValue: "mock:resource:r1",
				},
			},
			want: construct.ResourceId{Provider: "mock", Type: "resource", Name: "r1"},
		},
		{
			name: "default value with template",
			property: &ResourceProperty{
				SharedPropertyFields: SharedPropertyFields{
					DefaultValue: "{{ \"mock:resource:r1\" }}",
				},
			},
			want: construct.ResourceId{Provider: "mock", Type: "resource", Name: "r1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			actual, err := tt.property.GetDefaultValue(tt.ctx, tt.data)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(tt.want, actual)
		})
	}
}

func Test_ResourceProperty_Parse(t *testing.T) {
	tests := []struct {
		name     string
		property *ResourceProperty
		ctx      knowledgebase2.DynamicValueContext
		data     knowledgebase2.DynamicValueData
		value    any
		want     construct.ResourceId
		wantErr  bool
	}{
		{
			name: "resource property",
			property: &ResourceProperty{
				PropertyDetails: knowledgebase2.PropertyDetails{
					Path: "test",
				},
			},
			value: construct.ResourceId{Provider: "mock", Type: "resource", Name: "r1"},
			want:  construct.ResourceId{Provider: "mock", Type: "resource", Name: "r1"},
		},
		{
			name: "resource property from string",
			property: &ResourceProperty{
				PropertyDetails: knowledgebase2.PropertyDetails{
					Path: "test",
				},
			},
			value: "mock:resource:r1",
			want:  construct.ResourceId{Provider: "mock", Type: "resource", Name: "r1"},
		},
		{
			name: "resource property as template",
			property: &ResourceProperty{
				PropertyDetails: knowledgebase2.PropertyDetails{
					Path: "test",
				},
			},
			value: "{{ \"mock:resource:r1\" }}",
			want:  construct.ResourceId{Provider: "mock", Type: "resource", Name: "r1"},
		},
		{
			name: "non resource throws error",
			property: &ResourceProperty{
				PropertyDetails: knowledgebase2.PropertyDetails{
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
			actual, err := tt.property.Parse(tt.value, tt.ctx, tt.data)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.want, actual)
		})
	}
}

func Test_ResourceProperty_Clone(t *testing.T) {
	assert := assert.New(t)
	property := &ResourceProperty{
		PropertyDetails: knowledgebase2.PropertyDetails{
			Path: "test",
		},
	}
	actual := property.Clone()
	assert.Equal(property, actual)
}

func Test_ResourceProperty_ZeroValue(t *testing.T) {
	assert := assert.New(t)
	property := &ResourceProperty{}
	assert.Equal(property.ZeroValue(), construct.ResourceId{})
}

func Test_ResourceProperty_Details(t *testing.T) {
	assert := assert.New(t)
	property := &ResourceProperty{
		PropertyDetails: knowledgebase2.PropertyDetails{
			Path: "test",
		},
	}
	assert.Equal(property.Details(), &property.PropertyDetails)
}

func Test_ResourcePropertyContains(t *testing.T) {
	tests := []struct {
		name     string
		property *ResourceProperty
		value    construct.ResourceId
		contains construct.ResourceId
		want     bool
	}{
		{
			name: "resource property",
			property: &ResourceProperty{
				PropertyDetails: knowledgebase2.PropertyDetails{
					Path: "test",
				},
			},
			value:    construct.ResourceId{Provider: "test"},
			contains: construct.ResourceId{Provider: "test"},
			want:     true,
		},
		{
			name: "resource is different",
			property: &ResourceProperty{
				PropertyDetails: knowledgebase2.PropertyDetails{
					Path: "test",
				},
			},
			value:    construct.ResourceId{Provider: "test"},
			contains: construct.ResourceId{Provider: "test2"},
			want:     false,
		},
		{
			name: "resource property different namespace",
			property: &ResourceProperty{
				PropertyDetails: knowledgebase2.PropertyDetails{
					Path: "test",
				},
			},
			value:    construct.ResourceId{Provider: "test"},
			contains: construct.ResourceId{Provider: "test2", Namespace: "test2"},
			want:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.property.Contains(tt.value, tt.contains)
			if actual != tt.want {
				t.Errorf("Contains() = %v, want %v", actual, tt.want)
			}
		})
	}
}

func Test_ResourcePropertySubProperties(t *testing.T) {
	assert := assert.New(t)
	property := &ResourceProperty{
		PropertyDetails: knowledgebase2.PropertyDetails{
			Path: "test",
		},
	}
	assert.Nil(property.SubProperties())
}
