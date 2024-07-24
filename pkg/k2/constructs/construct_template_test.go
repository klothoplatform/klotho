package constructs

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"gopkg.in/yaml.v3"
)

func TestUnmarshalConstructTemplateId(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected ConstructTemplateId
		wantErr  bool
	}{
		{
			name:     "Valid ConstructTemplateId",
			input:    "package.name",
			expected: ConstructTemplateId{Package: "package", Name: "name"},
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
			var ctId ConstructTemplateId
			err := yaml.Unmarshal([]byte(tt.input), &ctId)
			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !compareConstructTemplateId(ctId, tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, ctId)
			}
		})
	}
}

func compareConstructTemplateId(a, b ConstructTemplateId) bool {
	return a.Package == b.Package && a.Name == b.Name
}

func TestParseConstructTemplateId(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected ConstructTemplateId
		wantErr  bool
	}{
		{
			name:     "Valid ConstructTemplateId",
			input:    "package.name",
			expected: ConstructTemplateId{Package: "package", Name: "name"},
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
			ctId, err := ParseConstructTemplateId(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseConstructTemplateId() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !compareConstructTemplateId(ctId, tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, ctId)
			}
		})
	}
}

func TestConstructTemplateId_String(t *testing.T) {
	ctId := ConstructTemplateId{Package: "package", Name: "name"}
	expected := "package.name"
	if ctId.String() != expected {
		t.Errorf("Expected %v, got %v", expected, ctId.String())
	}
}

func TestConstructTemplateId_FromURN(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected ConstructTemplateId
		wantErr  bool
	}{
		{
			name:     "Valid URN",
			input:    "urn:accountid:project:dev::construct/package.name",
			expected: ConstructTemplateId{Package: "package", Name: "name"},
			wantErr:  false,
		},
		{
			name:    "Invalid URN type",
			input:   "urn:accountid:project:dev::other/package.name",
			wantErr: true,
		},
		{
			name:    "Invalid URN format",
			input:   "urn:accountid:project:dev::construct/invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ctId ConstructTemplateId
			urn, err := model.ParseURN(tt.input)
			if err != nil {
				t.Fatalf("Failed to parse URN: %v", err)
			}
			err = ctId.FromURN(*urn)
			if (err != nil) != tt.wantErr {
				t.Errorf("FromURN() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && ctId != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, ctId)
			}
		})
	}
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
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !compareEdgeTemplates(et, tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, et)
			}
		})
	}
}

func compareEdgeTemplates(a, b EdgeTemplate) bool {
	return a.From == b.From && a.To == b.To && a.Data == b.Data
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
    default: "default"
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
		Id:          ConstructTemplateId{Package: "package", Name: "name"},
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
		Inputs: map[string]InputTemplate{
			"input1": {Name: "input1", Type: "string", Description: "An input", Default: "default"},
		},
		Outputs: map[string]OutputTemplate{
			"output1": {Name: "output1", Description: "An output", Value: "value1"},
		},
		InputRules: []InputRuleTemplate{
			{
				If: "condition",
				Then: ConditionalExpressionTemplate{
					Resources: map[string]ResourceTemplate{
						"res3": {Type: "type3", Name: "name3", Namespace: "namespace3", Properties: map[string]any{"prop3": "value3"}},
					},
				},
				Else: ConditionalExpressionTemplate{
					Resources: map[string]ResourceTemplate{
						"res4": {Type: "type4", Name: "name4", Namespace: "namespace4", Properties: map[string]any{"prop4": "value4"}},
					},
				},
			},
		},
		resourceOrder: []string{"res1", "res2"},
	}

	var ct ConstructTemplate
	err := yaml.Unmarshal([]byte(input), &ct)
	if err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	if ct.Id != expected.Id ||
		ct.Version != expected.Version ||
		ct.Description != expected.Description ||
		len(ct.Resources) != len(expected.Resources) ||
		len(ct.Edges) != len(expected.Edges) ||
		len(ct.Inputs) != len(expected.Inputs) ||
		len(ct.Outputs) != len(expected.Outputs) ||
		len(ct.InputRules) != len(expected.InputRules) ||
		len(ct.resourceOrder) != len(expected.resourceOrder) {
		t.Errorf("Expected %+v, got %+v", expected, ct)
	}
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
		if key != keys[i] {
			t.Errorf("Expected key '%s', got '%s'", keys[i], key)
		}
		if value.Type == "" || value.Name == "" || value.Namespace == "" || value.Properties == nil {
			t.Errorf("Expected non-empty fields in ResourceTemplate for key '%s'", key)
		}
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

	if len(keys) != len(order) {
		t.Errorf("Expected keys length %d, got %d", len(order), len(keys))
	}
	for i, key := range keys {
		if key != order[i] {
			t.Errorf("Expected key '%s', got '%s'", order[i], key)
		}
	}

	for i, value := range values {
		expectedValue := source[order[i]]
		if !compareResourceTemplates(value, expectedValue) {
			t.Errorf("Expected %+v, got %+v", expectedValue, value)
		}
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
