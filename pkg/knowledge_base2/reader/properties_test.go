package reader

import (
	"testing"

	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/klothoplatform/klotho/pkg/knowledge_base2/properties"
	"github.com/stretchr/testify/assert"
)

func Test_ConvertProperty(t *testing.T) {
	tests := []struct {
		name     string
		property Property
		expected knowledgebase.Property
	}{
		{
			name: "Get string property type",
			property: Property{
				Type:          "string",
				Name:          "test",
				Path:          "test",
				Required:      true,
				AllowedValues: []string{"test1", "test2"},
				SanitizeTmpl:  "test",
				// Name string `json:"name" yaml:"name"`
				// // Type defines the type of the property
				// Type string `json:"type" yaml:"type"`

				// Namespace bool `json:"namespace" yaml:"namespace"`

				// DefaultValue any `json:"default_value" yaml:"default_value"`

				// Required bool `json:"required" yaml:"required"`

				// ConfigurationDisabled bool `json:"configuration_disabled" yaml:"configuration_disabled"`

				// DeployTime bool `json:"deploy_time" yaml:"deploy_time"`

				// OperationalRule *knowledgebase.PropertyRule `json:"operational_rule" yaml:"operational_rule"`

				// Properties map[string]*Property `json:"properties" yaml:"properties"`

				// MinLength *int `yaml:"min_length"`
				// MaxLength *int `yaml:"max_length"`

				// LowerBound *float64 `yaml:"lower_bound"`
				// UpperBound *float64 `yaml:"upper_bound"`

				// AllowedTypes construct.ResourceList `yaml:"allowed_types"`

				// SanitizeTmpl  *knowledgebase.SanitizeTmpl `yaml:"sanitize"`
				// AllowedValues []string                    `yaml:"allowed_values"`

				// KeyProperty   knowledgebase.Property `yaml:"key_property"`
				// ValueProperty knowledgebase.Property `yaml:"value_property"`

				// ItemProperty knowledgebase.Property `yaml:"item_property"`

				// Path string `json:"-" yaml:"-"`
			},
			expected: &properties.StringProperty{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			actual, err := test.property.Convert()
			if assert.NoError(err, "Expected no error, but got: %v", err) {
				return
			}
			assert.Equal(actual, test.expected, "expected %v, got %v", test.expected, actual)
		})
	}
}
