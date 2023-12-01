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
