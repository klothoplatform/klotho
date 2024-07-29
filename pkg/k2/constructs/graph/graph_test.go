package graph

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/stretchr/testify/assert"
)

func TestNewGraph(t *testing.T) {
	g := NewGraph()
	assert.NotNil(t, g, "Expected new graph to be created, got nil")
}

func TestNewAcyclicGraph_SimpleAcyclicGraph(t *testing.T) {
	g := NewAcyclicGraph()

	node1, err := model.ParseURN("urn:pulumi:stack::project::resource1")
	assert.NoError(t, err, "Failed to parse URN")
	node2, err := model.ParseURN("urn:pulumi:stack::project::resource2")
	assert.NoError(t, err, "Failed to parse URN")

	err = g.AddVertex(*node1)
	assert.NoError(t, err, "Failed to add vertex %v", node1)
	err = g.AddVertex(*node2)
	assert.NoError(t, err, "Failed to add vertex %v", node2)

	err = g.AddEdge(*node1, *node2)
	assert.NoError(t, err, "Unexpected error adding edge %v -> %v", node1, node2)
}

func TestNewAcyclicGraph_SimpleCyclicGraph(t *testing.T) {
	g := NewAcyclicGraph()

	node1, err := model.ParseURN("urn:pulumi:stack::project::resource1")
	assert.NoError(t, err, "Failed to parse URN")
	node2, err := model.ParseURN("urn:pulumi:stack::project::resource2")
	assert.NoError(t, err, "Failed to parse URN")

	err = g.AddVertex(*node1)
	assert.NoError(t, err, "Failed to add vertex %v", node1)
	err = g.AddVertex(*node2)
	assert.NoError(t, err, "Failed to add vertex %v", node2)

	err = g.AddEdge(*node1, *node2)
	assert.NoError(t, err, "Failed to add edge %v -> %v", node1, node2)

	err = g.AddEdge(*node2, *node1)
	assert.Error(t, err, "Expected error when adding cyclic edge %v -> %v", node2, node1)
}

func TestResolveDeploymentGroups_SimpleGraph(t *testing.T) {
	g := NewGraph()

	node1, err := model.ParseURN("urn:pulumi:stack::project::resource1")
	assert.NoError(t, err, "Failed to parse URN")
	node2, err := model.ParseURN("urn:pulumi:stack::project::resource2")
	assert.NoError(t, err, "Failed to parse URN")

	err = g.AddVertex(*node1)
	assert.NoError(t, err, "Failed to add vertex")
	err = g.AddVertex(*node2)
	assert.NoError(t, err, "Failed to add vertex")

	err = g.AddEdge(*node1, *node2)
	assert.NoError(t, err, "Failed to add edge")

	expected := [][]string{{"urn:pulumi:stack::project::resource2"}, {"urn:pulumi:stack::project::resource1"}}
	groups, err := ResolveDeploymentGroups(g)
	assert.NoError(t, err, "Failed to resolve deployment groups")

	assert.True(t, compareGroupsURN(groups, expected), "Expected groups %v, but got %v", expected, groups)
}

func TestResolveDeploymentGroups_ComplexGraph(t *testing.T) {
	g := NewGraph()

	node1, err := model.ParseURN("urn:pulumi:stack::project::resource1")
	assert.NoError(t, err, "Failed to parse URN")
	node2, err := model.ParseURN("urn:pulumi:stack::project::resource2")
	assert.NoError(t, err, "Failed to parse URN")
	node3, err := model.ParseURN("urn:pulumi:stack::project::resource3")
	assert.NoError(t, err, "Failed to parse URN")

	err = g.AddVertex(*node1)
	assert.NoError(t, err, "Failed to add vertex")
	err = g.AddVertex(*node2)
	assert.NoError(t, err, "Failed to add vertex")
	err = g.AddVertex(*node3)
	assert.NoError(t, err, "Failed to add vertex")

	err = g.AddEdge(*node1, *node2)
	assert.NoError(t, err, "Failed to add edge")
	err = g.AddEdge(*node2, *node3)
	assert.NoError(t, err, "Failed to add edge")

	expected := [][]string{{"urn:pulumi:stack::project::resource3"}, {"urn:pulumi:stack::project::resource2"}, {"urn:pulumi:stack::project::resource1"}}
	groups, err := ResolveDeploymentGroups(g)
	assert.NoError(t, err, "Failed to resolve deployment groups")

	assert.True(t, compareGroupsURN(groups, expected), "Expected groups %v, but got %v", expected, groups)
}

func TestHasEdges(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() (*Graph, model.URN, []model.URN)
		expected bool
	}{
		{
			name: "NodeHasEdgesWithinGroup",
			setup: func() (*Graph, model.URN, []model.URN) {
				g := NewGraph()
				node1, _ := model.ParseURN("urn:pulumi:stack::project::resource1")
				node2, _ := model.ParseURN("urn:pulumi:stack::project::resource2")
				node3, _ := model.ParseURN("urn:pulumi:stack::project::resource3")

				_ = g.AddVertex(*node1)
				_ = g.AddVertex(*node2)
				_ = g.AddVertex(*node3)

				_ = g.AddEdge(*node1, *node2) // node1 -> node2
				_ = g.AddEdge(*node2, *node3) // node2 -> node3

				return &g, *node1, []model.URN{*node2, *node3}
			},
			expected: true,
		},
		{
			name: "NodeDoesNotHaveEdgesWithinGroup",
			setup: func() (*Graph, model.URN, []model.URN) {
				g := NewGraph()
				node1, _ := model.ParseURN("urn:pulumi:stack::project::resource1")
				node2, _ := model.ParseURN("urn:pulumi:stack::project::resource2")
				node3, _ := model.ParseURN("urn:pulumi:stack::project::resource3")

				_ = g.AddVertex(*node1)
				_ = g.AddVertex(*node2)
				_ = g.AddVertex(*node3)

				_ = g.AddEdge(*node1, *node2) // node1 -> node2
				_ = g.AddEdge(*node3, *node1) // node3 -> node1

				return &g, *node3, []model.URN{*node2}
			},
			expected: false,
		},
		{
			name: "GroupHasEdgesBackToNode",
			setup: func() (*Graph, model.URN, []model.URN) {
				g := NewGraph()
				node1, _ := model.ParseURN("urn:pulumi:stack::project::resource1")
				node2, _ := model.ParseURN("urn:pulumi:stack::project::resource2")
				node3, _ := model.ParseURN("urn:pulumi:stack::project::resource3")

				_ = g.AddVertex(*node1)
				_ = g.AddVertex(*node2)
				_ = g.AddVertex(*node3)

				_ = g.AddEdge(*node1, *node2) // node1 -> node2
				_ = g.AddEdge(*node2, *node3) // node2 -> node3
				_ = g.AddEdge(*node3, *node1) // node3 -> node1

				return &g, *node3, []model.URN{*node1}
			},
			expected: true,
		},
		{
			name: "NodeAndGroupHasEdgesBothWays",
			setup: func() (*Graph, model.URN, []model.URN) {
				g := NewGraph()
				node1, _ := model.ParseURN("urn:pulumi:stack::project::resource1")
				node2, _ := model.ParseURN("urn:pulumi:stack::project::resource2")
				node3, _ := model.ParseURN("urn:pulumi:stack::project::resource3")

				_ = g.AddVertex(*node1)
				_ = g.AddVertex(*node2)
				_ = g.AddVertex(*node3)

				_ = g.AddEdge(*node1, *node2) // node1 -> node2
				_ = g.AddEdge(*node2, *node3) // node2 -> node3
				_ = g.AddEdge(*node3, *node1) // node3 -> node1

				return &g, *node1, []model.URN{*node2, *node3}
			},
			expected: true,
		},
		{
			name: "NodeInGroupHasEdgesBackToNode",
			setup: func() (*Graph, model.URN, []model.URN) {
				g := NewGraph()
				node1, _ := model.ParseURN("urn:pulumi:stack::project::resource1")
				node2, _ := model.ParseURN("urn:pulumi:stack::project::resource2")
				node3, _ := model.ParseURN("urn:pulumi:stack::project::resource3")

				_ = g.AddVertex(*node1)
				_ = g.AddVertex(*node2)
				_ = g.AddVertex(*node3)

				_ = g.AddEdge(*node1, *node2) // node1 -> node2
				_ = g.AddEdge(*node3, *node1) // node3 -> node1

				return &g, *node1, []model.URN{*node3}
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, node, group := tt.setup()
			assert.Equal(t, tt.expected, hasEdges(node, group, *g))
		})
	}
}

// compareGroupsURN is a helper function to compare groups of URNs in string form
func compareGroupsURN(a [][]model.URN, b [][]string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if len(a[i]) != len(b[i]) || !compareURNsString(a[i], b[i]) {
			return false
		}
	}
	return true
}

// compareURNsString is a helper function to compare slices of URNs and strings
func compareURNsString(a []model.URN, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].String() != b[i] {
			return false
		}
	}
	return true
}
