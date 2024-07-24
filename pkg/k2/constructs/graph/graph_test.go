package graph

import (
	"testing"

	"github.com/dominikbraun/graph"
	"github.com/klothoplatform/klotho/pkg/k2/model"
)

func TestNewGraph(t *testing.T) {
	g := NewGraph()
	if g == nil {
		t.Error("Expected new graph to be created, got nil")
	}
}

func TestNewAcyclicGraph_SimpleAcyclicGraph(t *testing.T) {
	g := NewAcyclicGraph()

	node1, err := model.ParseURN("urn:pulumi:stack::project::resource1")
	if err != nil {
		t.Fatalf("Failed to parse URN: %v", err)
	}
	node2, err := model.ParseURN("urn:pulumi:stack::project::resource2")
	if err != nil {
		t.Fatalf("Failed to parse URN: %v", err)
	}

	err = g.AddVertex(*node1)
	if err != nil && err != graph.ErrVertexAlreadyExists {
		t.Fatalf("Failed to add vertex %v: %v", node1, err)
	}
	err = g.AddVertex(*node2)
	if err != nil && err != graph.ErrVertexAlreadyExists {
		t.Fatalf("Failed to add vertex %v: %v", node2, err)
	}

	err = g.AddEdge(*node1, *node2)
	if err != nil {
		t.Errorf("Unexpected error adding edge %v -> %v: %v", node1, node2, err)
	}
}

func TestNewAcyclicGraph_SimpleCyclicGraph(t *testing.T) {
	g := NewAcyclicGraph()

	node1, err := model.ParseURN("urn:pulumi:stack::project::resource1")
	if err != nil {
		t.Fatalf("Failed to parse URN: %v", err)
	}
	node2, err := model.ParseURN("urn:pulumi:stack::project::resource2")
	if err != nil {
		t.Fatalf("Failed to parse URN: %v", err)
	}

	err = g.AddVertex(*node1)
	if err != nil && err != graph.ErrVertexAlreadyExists {
		t.Fatalf("Failed to add vertex %v: %v", node1, err)
	}
	err = g.AddVertex(*node2)
	if err != nil && err != graph.ErrVertexAlreadyExists {
		t.Fatalf("Failed to add vertex %v: %v", node2, err)
	}

	err = g.AddEdge(*node1, *node2)
	if err != nil {
		t.Fatalf("Failed to add edge %v -> %v: %v", node1, node2, err)
	}

	err = g.AddEdge(*node2, *node1)
	if err == nil {
		t.Errorf("Expected error when adding cyclic edge %v -> %v, got nil", node2, node1)
	} else {
		t.Logf("Correctly detected cycle when adding edge %v -> %v: %v", node2, node1, err)
	}
}

func TestResolveDeploymentGroups_SimpleGraph(t *testing.T) {
	g := NewGraph()

	node1, err := model.ParseURN("urn:pulumi:stack::project::resource1")
	if err != nil {
		t.Fatalf("Failed to parse URN: %v", err)
	}
	node2, err := model.ParseURN("urn:pulumi:stack::project::resource2")
	if err != nil {
		t.Fatalf("Failed to parse URN: %v", err)
	}

	err = g.AddVertex(*node1)
	if err != nil {
		t.Fatalf("Failed to add vertex: %v", err)
	}
	err = g.AddVertex(*node2)
	if err != nil {
		t.Fatalf("Failed to add vertex: %v", err)
	}

	err = g.AddEdge(*node1, *node2)
	if err != nil {
		t.Fatalf("Failed to add edge: %v", err)
	}

	expected := [][]string{{"urn:pulumi:stack::project::resource2"}, {"urn:pulumi:stack::project::resource1"}}
	groups, err := ResolveDeploymentGroups(g)
	if err != nil {
		t.Fatalf("Failed to resolve deployment groups: %v", err)
	}

	if !compareGroupsURN(groups, expected) {
		t.Errorf("Expected groups %v, but got %v", expected, groups)
	}
}

func TestResolveDeploymentGroups_ComplexGraph(t *testing.T) {
	g := NewGraph()

	node1, err := model.ParseURN("urn:pulumi:stack::project::resource1")
	if err != nil {
		t.Fatalf("Failed to parse URN: %v", err)
	}
	node2, err := model.ParseURN("urn:pulumi:stack::project::resource2")
	if err != nil {
		t.Fatalf("Failed to parse URN: %v", err)
	}
	node3, err := model.ParseURN("urn:pulumi:stack::project::resource3")
	if err != nil {
		t.Fatalf("Failed to parse URN: %v", err)
	}

	err = g.AddVertex(*node1)
	if err != nil {
		t.Fatalf("Failed to add vertex: %v", err)
	}
	err = g.AddVertex(*node2)
	if err != nil {
		t.Fatalf("Failed to add vertex: %v", err)
	}
	err = g.AddVertex(*node3)
	if err != nil {
		t.Fatalf("Failed to add vertex: %v", err)
	}

	err = g.AddEdge(*node1, *node2)
	if err != nil {
		t.Fatalf("Failed to add edge: %v", err)
	}
	err = g.AddEdge(*node2, *node3)
	if err != nil {
		t.Fatalf("Failed to add edge: %v", err)
	}

	expected := [][]string{{"urn:pulumi:stack::project::resource3"}, {"urn:pulumi:stack::project::resource2"}, {"urn:pulumi:stack::project::resource1"}}
	groups, err := ResolveDeploymentGroups(g)
	if err != nil {
		t.Fatalf("Failed to resolve deployment groups: %v", err)
	}

	if !compareGroupsURN(groups, expected) {
		t.Errorf("Expected groups %v, but got %v", expected, groups)
	}
}

func TestHasEdges_NodeHasEdgesWithinGroup(t *testing.T) {
	g := NewGraph()

	node1, _ := model.ParseURN("urn:pulumi:stack::project::resource1")
	node2, _ := model.ParseURN("urn:pulumi:stack::project::resource2")
	node3, _ := model.ParseURN("urn:pulumi:stack::project::resource3")

	_ = g.AddVertex(*node1)
	_ = g.AddVertex(*node2)
	_ = g.AddVertex(*node3)

	_ = g.AddEdge(*node1, *node2) // node1 -> node2
	_ = g.AddEdge(*node2, *node3) // node2 -> node3

	group := []model.URN{*node2, *node3}

	if !hasEdges(*node1, group, g) {
		t.Errorf("Expected node1 to have edges with nodes in the group")
	}
}

func TestHasEdges_NodeDoesNotHaveEdgesWithinGroup(t *testing.T) {
	g := NewGraph()

	node1, _ := model.ParseURN("urn:pulumi:stack::project::resource1")
	node2, _ := model.ParseURN("urn:pulumi:stack::project::resource2")
	node3, _ := model.ParseURN("urn:pulumi:stack::project::resource3")

	_ = g.AddVertex(*node1)
	_ = g.AddVertex(*node2)
	_ = g.AddVertex(*node3)

	_ = g.AddEdge(*node1, *node2) // node1 -> node2
	_ = g.AddEdge(*node3, *node1) // node3 -> node1

	group := []model.URN{*node2}

	if hasEdges(*node3, group, g) {
		t.Errorf("Expected node3 not to have edges with node2")
	}
}

func TestHasEdges_GroupHasEdgesBackToNode(t *testing.T) {
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

	group := []model.URN{*node1}

	if !hasEdges(*node3, group, g) {
		t.Errorf("Expected node3 to have edges with node1")
	}
}

func TestHasEdges_NodeAndGroupHasEdgesBothWays(t *testing.T) {
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

	group := []model.URN{*node2, *node3}

	if !hasEdges(*node1, group, g) {
		t.Errorf("Expected node1 to have edges with nodes in the group")
	}
}

func TestHasEdges_NodeInGroupHasEdgesBackToNode(t *testing.T) {
	g := NewGraph()

	node1, _ := model.ParseURN("urn:pulumi:stack::project::resource1")
	node2, _ := model.ParseURN("urn:pulumi:stack::project::resource2")
	node3, _ := model.ParseURN("urn:pulumi:stack::project::resource3")

	_ = g.AddVertex(*node1)
	_ = g.AddVertex(*node2)
	_ = g.AddVertex(*node3)

	_ = g.AddEdge(*node1, *node2) // node1 -> node2
	_ = g.AddEdge(*node3, *node1) // node3 -> node1

	group := []model.URN{*node3}

	if !hasEdges(*node1, group, g) {
		t.Errorf("Expected node1 to have edges with nodes in the group")
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
