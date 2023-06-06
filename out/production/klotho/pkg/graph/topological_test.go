package graph

import (
	"strings"
	"testing"

	"github.com/dominikbraun/graph"
	"github.com/stretchr/testify/assert"
)

func TestStableTopologicalOrder(t *testing.T) {
	assert := assert.New(t)

	g := NewDirected(DummyVertex.Id)
	g.AddVerticesAndEdge("a", "c")
	g.AddVerticesAndEdge("b", "c")

	for i := 0; i < 10000; i++ {
		check, err := g.VertexIdsInTopologicalOrder()
		if !assert.NoError(err) {
			return
		}

		// "bac" would also be valid, but the ordering we use happens to be "abc", and that's consistent across runs
		expected := "abc"
		if !assert.Equal(expected, strings.Join(check, ""), `failed on attempt #%d`, i+1) {
			return
		}
	}
}

func TestKvIteratorStable(t *testing.T) {
	assert := assert.New(t)

	g := NewDirected(DummyVertex.Id)
	g.AddVerticesAndEdge("a", "c")
	g.AddVerticesAndEdge("b", "c")
	predecessorsMap, err := g.underlying.PredecessorMap()
	if !assert.NoError(err) {
		return
	}

	var verticesList string
	stringIterator.forEach(predecessorsMap, func(v string, _m map[string]graph.Edge[string]) {
		verticesList += v
	})
	assert.Equal("abc", verticesList)
}
