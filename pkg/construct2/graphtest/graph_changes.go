package graphtest

import (
	"fmt"
	"testing"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/stretchr/testify/assert"
)

type GraphChanges struct {
	construct.Graph

	Added   []construct.ResourceId
	Removed []construct.ResourceId

	AddedEdges   []construct.Edge
	RemovedEdges []construct.Edge
}

func RecordChanges(inner construct.Graph) *GraphChanges {
	return &GraphChanges{
		Graph: inner,
	}
}

func (c *GraphChanges) AddVertex(value *construct.Resource, options ...func(*graph.VertexProperties)) error {
	err := c.Graph.AddVertex(value, options...)
	if err == nil {
		c.Added = append(c.Added, value.ID)
	}
	return err
}

func (c *GraphChanges) AddVerticesFrom(g construct.Graph) error {
	adj, err := g.AdjacencyMap()
	if err != nil {
		return err
	}
	err = c.Graph.AddVerticesFrom(g)
	if err == nil {
		for v := range adj {
			c.Added = append(c.Added, v)
		}
	}
	return err
}

func (c *GraphChanges) RemoveVertex(hash construct.ResourceId) error {
	err := c.Graph.RemoveVertex(hash)
	if err == nil {
		c.Removed = append(c.Removed, hash)
	}
	return err
}

func (c *GraphChanges) AddEdge(
	sourceHash, targetHash construct.ResourceId,
	options ...func(*graph.EdgeProperties),
) error {
	err := c.Graph.AddEdge(sourceHash, targetHash, options...)
	if err == nil {
		c.AddedEdges = append(c.AddedEdges, construct.Edge{Source: sourceHash, Target: targetHash})
	}
	return err
}

func (c *GraphChanges) AddEdgesFrom(g construct.Graph) error {
	edges, err := g.Edges()
	if err != nil {
		return err
	}
	err = c.Graph.AddEdgesFrom(g)
	if err == nil {
		c.AddedEdges = append(c.AddedEdges, edges...)
	}
	return err
}

func (c *GraphChanges) RemoveEdge(source, target construct.ResourceId) error {
	err := c.Graph.RemoveEdge(source, target)
	if err == nil {
		c.RemovedEdges = append(c.RemovedEdges, construct.Edge{Source: source, Target: target})
	}
	return err
}

func (expected *GraphChanges) AssertEqual(t *testing.T, actual *GraphChanges) {
	// the following two helpers make the diffs nicer to read, instead of printing the whole structs
	ids := func(s []construct.ResourceId) []string {
		out := make([]string, len(s))
		for i, id := range s {
			out[i] = id.String()
		}
		return out
	}
	edges := func(s []construct.Edge) []string {
		out := make([]string, len(s))
		for i, e := range s {
			out[i] = fmt.Sprintf("%s -> %s", e.Source, e.Target)
		}
		return out
	}
	assert.ElementsMatch(t, ids(expected.Added), ids(actual.Added), "added vertices")
	assert.ElementsMatch(t, ids(expected.Removed), ids(actual.Removed), "removed vertices")
	assert.ElementsMatch(t, edges(expected.AddedEdges), edges(actual.AddedEdges), "added edges")
	assert.ElementsMatch(t, edges(expected.RemovedEdges), edges(actual.RemovedEdges), "removed edges")
}
