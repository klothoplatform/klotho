package constructs

import (
	"path/filepath"
	"testing"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/k2/model"
)

func TestNewConstruct(t *testing.T) {
	tests := []struct {
		name         string
		urn          string
		inputs       map[string]any
		expectedErr  bool
		expectedName string
	}{
		{
			name:         "Valid inputs",
			urn:          "urn:accountid:project:dev::construct/klotho.aws.Bucket:my-bucket",
			inputs:       map[string]any{"someKey": "someValue"},
			expectedErr:  false,
			expectedName: "my-bucket",
		},
		{
			name:        "Reserved Name key",
			urn:         "urn:accountid:project:dev::construct/klotho.aws.Bucket:my-bucket",
			inputs:      map[string]any{"Name": "invalid"},
			expectedErr: true,
		},
		{
			name:        "Invalid URN type",
			urn:         "urn:accountid:project:dev::resource/klotho.aws.Bucket:invalidType",
			inputs:      map[string]any{"someKey": "someValue"},
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			constructURN, err := model.ParseURN(tt.urn)
			if err != nil {
				t.Fatalf("Failed to parse URN: %v", err)
			}

			c, err := NewConstruct(*constructURN, tt.inputs)
			if tt.expectedErr {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if c.Inputs["Name"] != tt.expectedName {
				t.Errorf("Expected Name to be %v, got %v", tt.expectedName, c.Inputs["Name"])
			}
		})
	}
}

func TestGetInput(t *testing.T) {
	c := &Construct{
		Inputs: map[string]any{
			"key1": "value1",
			"key2": "value2",
		},
	}

	tests := []struct {
		name       string
		key        string
		expected   any
		shouldFind bool
	}{
		{
			name:       "Existing key",
			key:        "key1",
			expected:   "value1",
			shouldFind: true,
		},
		{
			name:       "Non-existing key",
			key:        "nonexistent",
			expected:   nil,
			shouldFind: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, found := c.GetInput(tt.key)
			if found != tt.shouldFind {
				t.Errorf("Expected found to be %v, got %v", tt.shouldFind, found)
			}
			if value != tt.expected {
				t.Errorf("Expected value to be %v, got %v", tt.expected, value)
			}
		})
	}
}

func TestDeclareOutput(t *testing.T) {
	c := &Construct{
		OutputDeclarations: make(map[string]OutputDeclaration),
	}

	outputDecl := OutputDeclaration{
		Name: "output1",
		Ref:  construct.PropertyRef{},
	}

	c.DeclareOutput("key1", outputDecl)

	if len(c.OutputDeclarations) != 1 {
		t.Errorf("Expected 1 output declaration, got %d", len(c.OutputDeclarations))
	}

	if c.OutputDeclarations["key1"] != outputDecl {
		t.Errorf("Expected output declaration to be %v, got %v", outputDecl, c.OutputDeclarations["key1"])
	}
}

func TestOrderedBindings(t *testing.T) {
	b1 := &Binding{Priority: 2}
	b2 := &Binding{Priority: 1}
	b3 := &Binding{Priority: 2}

	c := &Construct{
		Bindings: []*Binding{b1, b2, b3},
	}

	ordered := c.OrderedBindings()

	if len(ordered) != 3 {
		t.Fatalf("Expected 3 bindings, got %d", len(ordered))
	}

	if ordered[0] != b2 {
		t.Errorf("Expected first binding to be %v, got %v", b2, ordered[0])
	}

	if ordered[1] != b1 && ordered[1] != b3 {
		t.Errorf("Expected second binding to be either %v or %v, got %v", b1, b3, ordered[1])
	}

	if ordered[2] != b1 && ordered[2] != b3 {
		t.Errorf("Expected third binding to be either %v or %v, got %v", b1, b3, ordered[2])
	}
}
func TestGetTemplateResourcesIterator(t *testing.T) {
	mockResources := map[string]ResourceTemplate{
		"res1": {Type: "type1", Name: "name1", Namespace: "namespace1", Properties: map[string]any{"prop1": "value1"}},
		"res2": {Type: "type2", Name: "name2", Namespace: "namespace2", Properties: map[string]any{"prop2": "value2"}},
	}
	mockTemplate := ConstructTemplate{
		Resources:     mockResources,
		resourceOrder: []string{"res1", "res2"},
	}

	c := &Construct{
		ConstructTemplate: mockTemplate,
	}

	iter := c.GetTemplateResourcesIterator()

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

func TestPopulateDefaultInputValues(t *testing.T) {
	tests := []struct {
		name      string
		inputs    map[string]any
		templates map[string]InputTemplate
		expected  map[string]any
	}{
		{
			name:   "Populate default path value",
			inputs: map[string]any{},
			templates: map[string]InputTemplate{
				"pathInput": {
					Default: "default/path",
					Type:    "path",
				},
			},
			expected: map[string]any{
				"pathInput": getAbsolutePath(t, "default/path"),
			},
		},
		{
			name:   "Populate non-path default value",
			inputs: map[string]any{},
			templates: map[string]InputTemplate{
				"simpleInput": {
					Default: "default-value",
				},
			},
			expected: map[string]any{
				"simpleInput": "default-value",
			},
		},
		{
			name: "Existing value should not be overwritten",
			inputs: map[string]any{
				"simpleInput": "existing-value",
			},
			templates: map[string]InputTemplate{
				"simpleInput": {
					Default: "default-value",
				},
			},
			expected: map[string]any{
				"simpleInput": "existing-value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			populateDefaultInputValues(tt.inputs, tt.templates)
			for key, expectedValue := range tt.expected {
				if val, ok := tt.inputs[key]; !ok || val != expectedValue {
					t.Errorf("Expected %v for key %s, got %v", expectedValue, key, val)
				}
			}
		})
	}
}

// getAbsolutePath converts a relative path to an absolute path and fails the test if an error occurs
func getAbsolutePath(t *testing.T, path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("Failed to get absolute path for %s: %v", path, err)
	}
	return absPath
}

func TestConstructMethods(t *testing.T) {
	// Common setup
	edgeTemplates := []EdgeTemplate{{}, {}}
	inputRules := []InputRuleTemplate{{}, {}}
	outputTemplates := map[string]OutputTemplate{"output1": {}, "output2": {}}
	inputTemplates := map[string]InputTemplate{"input1": {}, "input2": {}}
	initialGraph := construct.NewGraph()
	resourceTemplates := map[string]ResourceTemplate{"resource1": {}, "resource2": {}}
	resources := map[string]*Resource{"resource1": {}, "resource2": {}}
	inputs := map[string]any{"input1": "value1"}
	meta := map[string]any{"meta1": "value1"}
	edges := []*Edge{}

	// Parse URNs for edges
	fromURN, err := model.ParseURN("urn:accountid:project:dev::construct/klotho.aws.Bucket:from-bucket")
	if err != nil {
		t.Fatalf("Failed to parse URN: %v", err)
	}
	toURN, err := model.ParseURN("urn:accountid:project:dev::construct/klotho.aws.Bucket:to-bucket")
	if err != nil {
		t.Fatalf("Failed to parse URN: %v", err)
	}

	edges = append(edges, &Edge{
		From: ResourceRef{
			ConstructURN: *fromURN,
			ResourceKey:  "from-resource",
			Property:     "from-property",
			Type:         ResourceRefTypeIaC,
		},
		To: ResourceRef{
			ConstructURN: *toURN,
			ResourceKey:  "to-resource",
			Property:     "to-property",
			Type:         ResourceRefTypeIaC,
		},
		Data: construct.EdgeData{},
	})

	mockTemplate := ConstructTemplate{
		Edges:         edgeTemplates,
		InputRules:    inputRules,
		Outputs:       outputTemplates,
		Inputs:        inputTemplates,
		Resources:     resourceTemplates,
		resourceOrder: []string{"resource1", "resource2"},
	}

	urn, err := model.ParseURN("urn:accountid:project:dev::construct/klotho.aws.Bucket:my-bucket")
	if err != nil {
		t.Fatalf("Failed to parse URN: %v", err)
	}

	c := &Construct{
		ConstructTemplate: mockTemplate,
		InitialGraph:      initialGraph,
		Resources:         resources,
		Inputs:            inputs,
		Meta:              meta,
		Edges:             edges,
		URN:               *urn,
	}

	t.Run("GetTemplateEdges", func(t *testing.T) {
		edges := c.GetTemplateEdges()
		if len(edges) != len(edgeTemplates) {
			t.Errorf("Expected %d edges, got %d", len(edgeTemplates), len(edges))
		}
	})

	t.Run("GetEdges and SetEdges", func(t *testing.T) {
		c.SetEdges(edges)
		retrievedEdges := c.GetEdges()
		if len(retrievedEdges) != len(edges) {
			t.Errorf("Expected %d edges, got %d", len(edges), len(retrievedEdges))
		}
	})

	t.Run("GetInputRules", func(t *testing.T) {
		rules := c.GetInputRules()
		if len(rules) != len(inputRules) {
			t.Errorf("Expected %d input rules, got %d", len(inputRules), len(rules))
		}
	})

	t.Run("GetTemplateOutputs", func(t *testing.T) {
		outputs := c.GetTemplateOutputs()
		if len(outputs) != len(outputTemplates) {
			t.Errorf("Expected %d outputs, got %d", len(outputTemplates), len(outputs))
		}
	})

	t.Run("GetPropertySource", func(t *testing.T) {
		ps := c.GetPropertySource()
		if ps == nil {
			t.Errorf("Expected PropertySource, got nil")
		}
	})

	t.Run("GetResource and SetResource", func(t *testing.T) {
		resource := &Resource{}
		c.SetResource("resource1", resource)
		retrievedResource, ok := c.GetResource("resource1")
		if !ok {
			t.Errorf("Expected resource to be found, got not found")
		}
		if retrievedResource != resource {
			t.Errorf("Expected %v, got %v", resource, retrievedResource)
		}
	})

	t.Run("GetResources", func(t *testing.T) {
		retrievedResources := c.GetResources()
		if len(retrievedResources) != len(resources) {
			t.Errorf("Expected %d resources, got %d", len(resources), len(retrievedResources))
		}
	})

	t.Run("GetInitialGraph", func(t *testing.T) {
		graph := c.GetInitialGraph()
		if graph != initialGraph {
			t.Errorf("Expected initial graph %v, got %v", initialGraph, graph)
		}
	})

	t.Run("GetTemplateInputs", func(t *testing.T) {
		inputs := c.GetTemplateInputs()
		if len(inputs) != len(inputTemplates) {
			t.Errorf("Expected %d inputs, got %d", len(inputTemplates), len(inputs))
		}
	})

	t.Run("GetURN", func(t *testing.T) {
		retrievedURN := c.GetURN()
		if retrievedURN != *urn {
			t.Errorf("Expected URN %v, got %v", *urn, retrievedURN)
		}
	})

	t.Run("Edge.PrettyPrint", func(t *testing.T) {
		expected := "from-resource#from-property -> to-resource#to-property"
		if edges[0].PrettyPrint() != expected {
			t.Errorf("Expected PrettyPrint to be %s, got %s", expected, edges[0].PrettyPrint())
		}
	})

	t.Run("Edge.String", func(t *testing.T) {
		expected := "from-resource#from-property -> to-resource#to-property :: {}"
		if edges[0].String() != expected {
			t.Errorf("Expected String to be %s, got %s", expected, edges[0].String())
		}
	})

	t.Run("ResourceRef.String", func(t *testing.T) {
		refURN, err := model.ParseURN("urn:accountid:project:dev::construct/klotho.aws.Bucket:resource")
		if err != nil {
			t.Fatalf("Failed to parse URN: %v", err)
		}

		ref := ResourceRef{
			ConstructURN: *refURN,
			ResourceKey:  "resource-key",
			Property:     "property",
			Type:         ResourceRefTypeIaC,
		}
		expected := "resource-key#property"
		if ref.String() != expected {
			t.Errorf("Expected String to be %s, got %s", expected, ref.String())
		}
	})
}
