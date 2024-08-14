package constructs

import (
	"path/filepath"
	"testing"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/stretchr/testify/assert"
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
			assert.NoError(t, err)

			c, err := NewConstruct(*constructURN, tt.inputs)
			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedName, c.Inputs["Name"])
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
			assert.Equal(t, tt.shouldFind, found)
			assert.Equal(t, tt.expected, value)
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

	assert.Len(t, c.OutputDeclarations, 1)
	assert.Equal(t, outputDecl, c.OutputDeclarations["key1"])
}

func TestOrderedBindings(t *testing.T) {
	b1 := &Binding{Priority: 2}
	b2 := &Binding{Priority: 1}
	b3 := &Binding{Priority: 2}

	c := &Construct{
		Bindings: []*Binding{b1, b2, b3},
	}

	ordered := c.OrderedBindings()

	assert.Len(t, ordered, 3)
	assert.Equal(t, b2, ordered[0])
	assert.Contains(t, []*Binding{b1, b3}, ordered[1])
	assert.Contains(t, []*Binding{b1, b3}, ordered[2])
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
		assert.Equal(t, keys[i], key)
		assert.NotEmpty(t, value.Type)
		assert.NotEmpty(t, value.Name)
		assert.NotEmpty(t, value.Namespace)
		assert.NotNil(t, value.Properties)
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
				assert.Equal(t, expectedValue, tt.inputs[key])
			}
		})
	}
}

func getAbsolutePath(t *testing.T, path string) string {
	absPath, err := filepath.Abs(path)
	assert.NoError(t, err)
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
	assert.NoError(t, err)

	toURN, err := model.ParseURN("urn:accountid:project:dev::construct/klotho.aws.Bucket:to-bucket")
	assert.NoError(t, err)

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
	assert.NoError(t, err)

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
		assert.Len(t, edges, len(edgeTemplates))
	})

	t.Run("GetEdges and SetEdges", func(t *testing.T) {
		c.SetEdges(edges)
		retrievedEdges := c.GetEdges()
		assert.Len(t, retrievedEdges, len(edges))
	})

	t.Run("GetInputRules", func(t *testing.T) {
		rules := c.GetInputRules()
		assert.Len(t, rules, len(inputRules))
	})

	t.Run("GetTemplateOutputs", func(t *testing.T) {
		outputs := c.GetTemplateOutputs()
		assert.Len(t, outputs, len(outputTemplates))
	})

	t.Run("GetPropertySource", func(t *testing.T) {
		ps := c.GetPropertySource()
		assert.NotNil(t, ps)
	})

	t.Run("GetResource and SetResource", func(t *testing.T) {
		resource := &Resource{}
		c.SetResource("resource1", resource)
		retrievedResource, ok := c.GetResource("resource1")
		assert.True(t, ok)
		assert.Equal(t, resource, retrievedResource)
	})

	t.Run("GetResources", func(t *testing.T) {
		retrievedResources := c.GetResources()
		assert.Len(t, retrievedResources, len(resources))
	})

	t.Run("GetInitialGraph", func(t *testing.T) {
		graph := c.GetInitialGraph()
		assert.Equal(t, initialGraph, graph)
	})

	t.Run("GetTemplateInputs", func(t *testing.T) {
		inputs := c.GetTemplateInputs()
		assert.Len(t, inputs, len(inputTemplates))
	})

	t.Run("GetURN", func(t *testing.T) {
		retrievedURN := c.GetURN()
		assert.Equal(t, *urn, retrievedURN)
	})

	t.Run("Edge.PrettyPrint", func(t *testing.T) {
		expected := "from-resource#from-property -> to-resource#to-property"
		assert.Equal(t, expected, edges[0].PrettyPrint())
	})

	t.Run("Edge.String", func(t *testing.T) {
		expected := "from-resource#from-property -> to-resource#to-property :: {}"
		assert.Equal(t, expected, edges[0].String())
	})

	t.Run("ResourceRef.String", func(t *testing.T) {
		refURN, err := model.ParseURN("urn:accountid:project:dev::construct/klotho.aws.Bucket:resource")
		assert.NoError(t, err)

		ref := ResourceRef{
			ConstructURN: *refURN,
			ResourceKey:  "resource-key",
			Property:     "property",
			Type:         ResourceRefTypeIaC,
		}
		expected := "resource-key#property"
		assert.Equal(t, expected, ref.String())
	})
}
