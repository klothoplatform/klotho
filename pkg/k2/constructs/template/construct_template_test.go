package template

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/k2/constructs/template/properties"
	"github.com/klothoplatform/klotho/pkg/k2/constructs/template/property"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestUnmarshalConstructTemplateId(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected property.ConstructType
		wantErr  bool
	}{
		{
			name:     "Valid ConstructTemplateId",
			input:    "package.name",
			expected: property.ConstructType{Package: "package", Name: "name"},
			wantErr:  false,
		},
		{
			name:    "Invalid ConstructTemplateId",
			input:   "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ctId property.ConstructType
			err := yaml.Unmarshal([]byte(tt.input), &ctId)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, ctId)
			}
		})
	}
}

func TestParseConstructType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected property.ConstructType
		wantErr  bool
	}{
		{
			name:     "Valid ConstructTemplateId",
			input:    "package.name",
			expected: property.ConstructType{Package: "package", Name: "name"},
			wantErr:  false,
		},
		{
			name:    "Invalid ConstructTemplateId",
			input:   "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctId, err := property.ParseConstructType(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, ctId)
			}
		})
	}
}

func TestConstructType_String(t *testing.T) {
	ctId := property.ConstructType{Package: "package", Name: "name"}
	expected := "package.name"
	assert.Equal(t, expected, ctId.String())
}

func TestEdgeTemplate_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected EdgeTemplate
		wantErr  bool
	}{
		{
			name: "Valid EdgeTemplate",
			input: `
from: from-resource
to: to-resource
data: {}`,
			expected: EdgeTemplate{
				From: ResourceRef{ResourceKey: "from-resource", Type: ResourceRefTypeTemplate},
				To:   ResourceRef{ResourceKey: "to-resource", Type: ResourceRefTypeTemplate},
				Data: construct.EdgeData{},
			},
			wantErr: false,
		},
		{
			name:    "Invalid EdgeTemplate",
			input:   `invalid`, // This should trigger a YAML unmarshaling error
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var et EdgeTemplate
			err := yaml.Unmarshal([]byte(tt.input), &et)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, et)
			}
		})
	}
}

func TestConstructTemplate_UnmarshalYAML(t *testing.T) {
	input := `
id: package.name
version: "1.0"
description: "A test template"
resources:
  res1:
    type: "type1"
    name: "name1"
    namespace: "namespace1"
    properties:
      prop1: "value1"
  res2:
    type: "type2"
    name: "name2"
    namespace: "namespace2"
    properties:
      prop2: "value2"
edges:
  - from: res1
    to: res2
    data: {}
inputs:
  input1:
    name: "input1"
    type: "string"
    description: "An input"
    default_value: "default"
outputs:
  output1:
    name: "output1"
    description: "An output"
    value: "value1"
input_rules:
  - if: "condition"
    then:
      resources:
        res3:
          type: "type3"
          name: "name3"
          namespace: "namespace3"
          properties:
            prop3: "value3"
    else:
      resources:
        res4:
          type: "type4"
          name: "name4"
          namespace: "namespace4"
          properties:
            prop4: "value4"
`

	expected := ConstructTemplate{
		Id:          property.ConstructType{Package: "package", Name: "name"},
		Version:     "1.0",
		Description: "A test template",
		Resources: map[string]ResourceTemplate{
			"res1": {Type: "type1", Name: "name1", Namespace: "namespace1", Properties: map[string]any{"prop1": "value1"}},
			"res2": {Type: "type2", Name: "name2", Namespace: "namespace2", Properties: map[string]any{"prop2": "value2"}},
		},
		Edges: []EdgeTemplate{
			{
				From: ResourceRef{ResourceKey: "res1", Type: ResourceRefTypeTemplate},
				To:   ResourceRef{ResourceKey: "res2", Type: ResourceRefTypeTemplate},
				Data: construct.EdgeData{},
			},
		},
		Inputs: NewProperties(property.PropertyMap{
			"input1": &properties.StringProperty{
				PropertyDetails: property.PropertyDetails{
					Name:        "input1",
					Description: "An input",
					Path:        "input1",
				},
				SharedPropertyFields: properties.SharedPropertyFields{DefaultValue: "default"},
			},
		}),
		Outputs: map[string]OutputTemplate{
			"output1": {Name: "output1", Description: "An output", Value: "value1"},
		},
		InputRules: []InputRuleTemplate{
			{
				If: "condition",
				Then: &ConditionalExpressionTemplate{
					Resources: map[string]ResourceTemplate{
						"res3": {Type: "type3", Name: "name3", Namespace: "namespace3", Properties: map[string]any{"prop3": "value3"}},
					},
					resourceOrder: []string{"res3"},
				},
				Else: &ConditionalExpressionTemplate{
					Resources: map[string]ResourceTemplate{
						"res4": {Type: "type4", Name: "name4", Namespace: "namespace4", Properties: map[string]any{"prop4": "value4"}},
					},
					resourceOrder: []string{"res4"},
				},
			},
		},
		resourceOrder: []string{"res1", "res2"},
	}

	var ct ConstructTemplate
	err := yaml.Unmarshal([]byte(input), &ct)
	assert.NoError(t, err)
	assert.Equal(t, expected, ct)
}

func TestBindingTemplate_ResourcesIterator(t *testing.T) {
	mockTemplate := BindingTemplate{
		Resources: map[string]ResourceTemplate{
			"res1": {Type: "type1", Name: "name1", Namespace: "namespace1", Properties: map[string]any{"prop1": "value1"}},
			"res2": {Type: "type2", Name: "name2", Namespace: "namespace2", Properties: map[string]any{"prop2": "value2"}},
		},
		resourceOrder: []string{"res1", "res2"},
	}

	iter := mockTemplate.ResourcesIterator()

	keys := []string{"res1", "res2"}
	i := 0

	for key, value, ok := iter.Next(); ok; key, value, ok = iter.Next() {
		assert.Equal(t, keys[i], key)
		assert.NotEmpty(t, value.Type)
		assert.NotEmpty(t, value.Name)
		assert.NotEmpty(t, value.Namespace)
		assert.NotEmpty(t, value.Properties)
		i++
	}
}

func TestIterator_ForEach(t *testing.T) {
	source := map[string]ResourceTemplate{
		"res1": {Type: "type1", Name: "name1", Namespace: "namespace1", Properties: map[string]any{"prop1": "value1"}},
		"res2": {Type: "type2", Name: "name2", Namespace: "namespace2", Properties: map[string]any{"prop2": "value2"}},
	}

	order := []string{"res1", "res2"}

	iter := Iterator[string, ResourceTemplate]{
		source: source,
		order:  order,
	}

	var keys []string
	var values []ResourceTemplate

	iter.ForEach(func(key string, value ResourceTemplate) error {
		keys = append(keys, key)
		values = append(values, value)
		return nil
	})

	assert.Equal(t, len(order), len(keys))
	for i, key := range keys {
		assert.Equal(t, order[i], key)
	}

	for i, value := range values {
		expectedValue := source[order[i]]
		assert.True(t, compareResourceTemplates(value, expectedValue))
	}
}

func compareResourceTemplates(a, b ResourceTemplate) bool {
	if a.Type != b.Type || a.Name != b.Name || a.Namespace != b.Namespace {
		return false
	}
	if len(a.Properties) != len(b.Properties) {
		return false
	}
	for k, v := range a.Properties {
		if bv, ok := b.Properties[k]; !ok || v != bv {
			return false
		}
	}
	return true
}

func TestBindingTemplate_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected BindingTemplate
		wantErr  bool
	}{
		{
			name: "Valid BindingTemplate",
			input: `
resources:
  res1:
    type: "type1"
    name: "name1"
    namespace: "namespace1"
    properties:
      prop1: "value1"
  res2:
    type: "type2"
    name: "name2"
    namespace: "namespace2"
    properties:
      prop2: "value2"
edges:
  - from: res1
    to: res2
    data: {}`,
			expected: BindingTemplate{
				Resources: map[string]ResourceTemplate{
					"res1": {Type: "type1", Name: "name1", Namespace: "namespace1", Properties: map[string]any{"prop1": "value1"}},
					"res2": {Type: "type2", Name: "name2", Namespace: "namespace2", Properties: map[string]any{"prop2": "value2"}},
				},
				Edges: []EdgeTemplate{
					{
						From: ResourceRef{ResourceKey: "res1", Type: ResourceRefTypeTemplate},
						To:   ResourceRef{ResourceKey: "res2", Type: ResourceRefTypeTemplate},
						Data: construct.EdgeData{},
					},
				},
				resourceOrder: []string{"res1", "res2"},
			},
			wantErr: false,
		},
		{
			name:    "Invalid BindingTemplate",
			input:   `invalid`, // This should trigger a YAML unmarshaling error
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bt BindingTemplate
			err := yaml.Unmarshal([]byte(tt.input), &bt)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !compareBindingTemplates(bt, tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, bt)
			}
		})
	}
}

func compareBindingTemplates(a, b BindingTemplate) bool {
	if !compareResourceTemplates(a.Resources["res1"], b.Resources["res1"]) {
		return false
	}
	if !compareResourceTemplates(a.Resources["res2"], b.Resources["res2"]) {
		return false
	}
	if len(a.Edges) != len(b.Edges) {
		return false
	}
	for i, edge := range a.Edges {
		if edge.From != b.Edges[i].From || edge.To != b.Edges[i].To || edge.Data != b.Edges[i].Data {
			return false
		}
	}
	return true
}
