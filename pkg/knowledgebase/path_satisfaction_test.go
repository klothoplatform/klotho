package knowledgebase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

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
