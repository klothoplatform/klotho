package knowledgebase2

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

var testTemplate = ResourceTemplate{
	Properties: map[string]*Property{
		"name": {
			Name: "name",
			Type: "list",
			Properties: map[string]*Property{
				"nested": {
					Name: "nested",
					Type: "string",
				},
			},
		},
	},
}

func Test_PathSatisfactionRouteUnmarshalYaml(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		expected PathSatisfactionRoute
	}{
		{
			name: "as struct with prop ref",
			yaml: `classification: network
property_reference: Subnet#AvailabilityZone`,
			expected: PathSatisfactionRoute{
				Classification:    "network",
				PropertyReference: "Subnet#AvailabilityZone",
			},
		},
		{
			name: "as struct with validity",
			yaml: `classification: network
property_reference: Subnet#AvailabilityZone
validity: downstream`,
			expected: PathSatisfactionRoute{
				Classification:    "network",
				PropertyReference: "Subnet#AvailabilityZone",
				Validity:          "downstream",
			},
		},
		{
			name: "as string",
			yaml: `network#Subnet#AvailabilityZone`,
			expected: PathSatisfactionRoute{
				Classification:    "network",
				PropertyReference: "Subnet#AvailabilityZone",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			p := PathSatisfactionRoute{}
			node := &yaml.Node{}
			err := yaml.Unmarshal([]byte(test.yaml), node)
			assert.NoError(err, "Expected no error")
			if node.Content[0].Kind == yaml.ScalarNode {
				err = p.UnmarshalYAML(node.Content[0])
			} else {
				err = p.UnmarshalYAML(node)
			}
			assert.NoError(err, "Expected no error")
			assert.Equal(p, test.expected, "Expected unmarshalled yaml to equal expected")
		})
	}
}

func Test_PropertiesUnmarshalYaml(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		expected Properties
	}{
		{
			name: "propagate path and name",
			yaml: `
name:
  type: string
  properties:
    nested:
      type: string`,
			expected: map[string]*Property{
				"name": {
					Name: "name",
					Path: "name",
					Type: "string",
					Properties: map[string]*Property{
						"nested": {
							Name: "nested",
							Path: "name.nested",
							Type: "string",
						},
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			p := Properties{}
			node := &yaml.Node{}
			err := yaml.Unmarshal([]byte(test.yaml), node)
			assert.NoError(err, "Expected no error")
			err = p.UnmarshalYAML(node)
			assert.NoError(err, "Expected no error")
			assert.Equal(p, test.expected, "Expected unmarshalled yaml to equal expected")
		})
	}
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
				Type: "list",
			},
		},
		{
			name:     "Get nested property",
			template: testTemplate,
			property: "name.nested",
			expected: &Property{
				Name: "nested",
				Type: "string",
			},
		},
		{
			name:     "Get nested property with array index",
			template: testTemplate,
			property: "name[0].nested",
			expected: &Property{
				Name: "nested",
				Type: "string",
			},
		},
		{
			name:     "Get nested property with array index and array property",
			template: testTemplate,
			property: "name[0].nested[0]",
			expected: &Property{
				Name: "nested",
				Type: "string",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			actual := test.template.GetProperty(test.property)
			assert.NotNil(actual, "Expected property %s to exist", test.property)
			assert.Equal(actual.Name, test.expected.Name, "Expected property name %s to equal %s", actual.Name, test.expected.Name)
			assert.Equal(actual.Type, test.expected.Type, "Expected property Type %s to equal %s", actual.Type, test.expected.Type)
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
			template: ResourceTemplate{Properties: map[string]*Property{"name": {Name: "name", Namespace: true}}},
			expected: &Property{Name: "name", Namespace: true},
		},
		{
			name:     "Get namespaced property with nested properties only looks top level",
			template: ResourceTemplate{Properties: map[string]*Property{"name": {Name: "name", Properties: map[string]*Property{"nested": {Name: "nested", Namespace: true}}}}},
			expected: nil,
		},
		{
			name:     "Get non namespaced property",
			template: ResourceTemplate{Properties: map[string]*Property{"name": {Name: "name"}}},
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
