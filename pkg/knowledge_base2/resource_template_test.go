package knowledgebase2

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var testTemplate = ResourceTemplate{
	Properties: map[string]Property{
		"name": {
			Name: "name",
			Properties: map[string]Property{
				"nested": {
					Name: "nested",
				},
			},
		},
	},
}

func Test_GetProperty(t *testing.T) {
	tests := []struct {
		name     string
		template ResourceTemplate
		property string
		expected *Property
	}{
		{
			name:     "Get top level property",
			template: testTemplate,
			property: "name",
			expected: &Property{
				Name: "name",
			},
		},
		{
			name:     "Get nested property",
			template: testTemplate,
			property: "name.nested",
			expected: &Property{
				Name: "nested",
			},
		},
		{
			name:     "Get nested property with array index",
			template: testTemplate,
			property: "name[0].nested",
			expected: &Property{
				Name: "nested",
			},
		},
		{
			name:     "Get nested property with array index and array property",
			template: testTemplate,
			property: "name[0].nested[0]",
			expected: &Property{
				Name: "nested",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			actual := test.template.GetProperty(test.property)
			assert.NotNil(actual, "Expected property %s to exist", test.property)
			assert.Equal(actual.Name, test.expected.Name, "Expected property name %s to equal %s", actual.Name, test.expected.Name)
		})
	}
}

func Test_GetNamespacedProperty(t *testing.T) {
	tests := []struct {
		name     string
		template ResourceTemplate
		expected *Property
	}{
		{
			name:     "Get namespaced property",
			template: ResourceTemplate{Properties: map[string]Property{"name": {Name: "name", Namespace: true}}},
			expected: &Property{Name: "name", Namespace: true},
		},
		{
			name:     "Get namespaced property with nested properties only looks top level",
			template: ResourceTemplate{Properties: map[string]Property{"name": {Name: "name", Properties: map[string]Property{"nested": {Name: "nested", Namespace: true}}}}},
			expected: nil,
		},
		{
			name:     "Get non namespaced property",
			template: ResourceTemplate{Properties: map[string]Property{"name": {Name: "name"}}},
			expected: nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			actual := test.template.GetNamespacedProperty()
			if test.expected == nil {
				assert.Nil(actual, "Expected property to be nil")
				return
			}
			assert.Equal(actual, test.expected, "Expected property %s to equal %s", actual, test.expected)
		})
	}
}
