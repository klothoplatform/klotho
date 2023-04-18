package graph

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStableTopologicalOrder(t *testing.T) {
	assert := assert.New(t)

	g := NewDirected[dummyVertex]()
	g.AddVerticesAndEdge("a", "c")
	g.AddVerticesAndEdge("b", "c")

	archetype, err := g.VertexIdsInTopologicalOrder()
	if !assert.NoError(err) {
		return
	}
	archetypeStr := strings.Join(archetype, "")
	assert.True(archetypeStr == "abc" || archetypeStr == "bac")

	for i := 0; i < 10000; i++ {
		check, err := g.VertexIdsInTopologicalOrder()
		if !assert.NoError(err) {
			return
		}
		checkStr := strings.Join(check, "")
		if !assert.Equal(archetypeStr, checkStr, `failed on attempt #%d`, i+1) {
			return
		}
	}

}

type dummyVertex string

func (d dummyVertex) Id() string {
	return string(d)
}
