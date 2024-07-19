package constructs

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/k2/constructs/template"
	properties2 "github.com/klothoplatform/klotho/pkg/k2/constructs/template/properties"
	"github.com/klothoplatform/klotho/pkg/k2/constructs/template/property"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/stretchr/testify/assert"
)

func TestGetInput(t *testing.T) {
	c := &Construct{
		Inputs: map[string]any{
			"key1": "value1",
			"key2": "value2",
		},
	}

	tests := []struct {
		name     string
		key      string
		expected any
		wantErr  bool
	}{
		{
			name:     "Existing key",
			key:      "key1",
			expected: "value1",
		},
		{
			name:     "Non-existing key",
			key:      "nonexistent",
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := c.GetInputValue(tt.key)
			if tt.wantErr {
				require.Error(t, err)
			}
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
	mockTemplate, err := parseConstructTemplate(`
resources:
  res1:
    type: type1
    name: name1
    namespace: namespace1
    properties:
      prop1: value1
  res2:  
    type: type2
    name: name2
    namespace: namespace2
    properties:
      prop2: value2
`)
	require.NoError(t, err)

	c := &Construct{
		ConstructTemplate: *mockTemplate,
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

func parseConstructTemplate(yamlStr string) (*template.ConstructTemplate, error) {
	mockTemplate := &template.ConstructTemplate{}
	err := yaml.Unmarshal([]byte(yamlStr), mockTemplate)
	if err != nil {
		return nil, err
	}
	return mockTemplate, nil
}

func TestConstructMethods(t *testing.T) {
	// Common setup
	edgeTemplates := []template.EdgeTemplate{{}, {}}
	inputRules := []template.InputRuleTemplate{{}, {}}
	outputTemplates := map[string]template.OutputTemplate{"output1": {}, "output2": {}}
	inputTemplates := template.NewProperties(map[string]property.Property{"input1": &properties2.StringProperty{}, "input2": &properties2.StringProperty{}})
	initialGraph := construct.NewGraph()
	resourceTemplates := map[string]template.ResourceTemplate{"resource1": {}, "resource2": {}}
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
		From: template.ResourceRef{
			ConstructURN: *fromURN,
			ResourceKey:  "from-resource",
			Property:     "from-property",
			Type:         template.ResourceRefTypeIaC,
		},
		To: template.ResourceRef{
			ConstructURN: *toURN,
			ResourceKey:  "to-resource",
			Property:     "to-property",
			Type:         template.ResourceRefTypeIaC,
		},
		Data: construct.EdgeData{},
	})

	mockTemplate := template.ConstructTemplate{
		Edges:      edgeTemplates,
		InputRules: inputRules,
		Outputs:    outputTemplates,
		Inputs:     inputTemplates,
		Resources:  resourceTemplates,
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

		ref := template.ResourceRef{
			ConstructURN: *refURN,
			ResourceKey:  "resource-key",
			Property:     "property",
			Type:         template.ResourceRefTypeIaC,
		}
		expected := "resource-key#property"
		assert.Equal(t, expected, ref.String())
	})
}
