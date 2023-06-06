package graph

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEmptyGraph(t *testing.T) {
	assert := assert.New(t)
	d := NewDirected(DummyVertex.Id)
	assert.Empty(d.Roots())
}

func TestSimpleGraph(t *testing.T) {
	// A ┬─➤ B
	//   └─➤ C
	a, b, c := DummyVertex("a"), DummyVertex("b"), DummyVertex("c")
	d := NewDirected(DummyVertex.Id)
	d.AddVertex(a)
	d.AddVertex(b)
	d.AddVertex(c)
	d.AddEdge(a.Id(), b.Id(), nil)
	d.AddEdge(a.Id(), c.Id(), nil)

	test(t, "roots", func(assert *assert.Assertions) {
		assert.Equal([]DummyVertex{a}, d.Roots())
	})
	test(t, "outgoing nodes", func(assert *assert.Assertions) {
		assert.ElementsMatch([]DummyVertex{b, c}, d.OutgoingVertices(a))
	})
	test(t, "outgoing edges", func(assert *assert.Assertions) {
		assert.ElementsMatch(
			[]Edge[DummyVertex]{
				{
					Source:      a,
					Destination: b,
				},
				{
					Source:      a,
					Destination: c,
				},
			},
			d.OutgoingEdges(a))
	})
}

func TestCycleToSelf(t *testing.T) {
	assert := assert.New(t)
	d := NewDirected(DummyVertex.Id)
	v := DummyVertex("dummy")
	d.AddVertex(v)
	d.AddEdge(v.Id(), v.Id(), nil)
	assert.Equal(
		[]Edge[DummyVertex]{
			{
				Source:      v,
				Destination: v,
			},
		},
		d.OutgoingEdges(v))
}

func TestCycle(t *testing.T) {
	assert := assert.New(t)
	d := NewDirected(DummyVertex.Id)
	v1 := DummyVertex("hello")
	v2 := DummyVertex("world")
	d.AddVertex(v1)
	d.AddVertex(v2)
	d.AddEdge(v1.Id(), v2.Id(), nil)
	d.AddEdge(v2.Id(), v1.Id(), nil)
	assert.Equal(
		[]Edge[DummyVertex]{
			{
				Source:      v1,
				Destination: v2,
			},
		},
		d.OutgoingEdges(v1))
	assert.Equal(
		[]Edge[DummyVertex]{
			{
				Source:      v2,
				Destination: v1,
			},
		},
		d.OutgoingEdges(v2))
}

func TestNegativeCases(t *testing.T) {
	test(t, "duplicate vertex", func(assert *assert.Assertions) {
		d := NewDirected(DummyVertex.Id)
		v := DummyVertex("dummy")
		d.AddVertex(v)
		d.AddVertex(v)
		assert.Equal([]DummyVertex{v}, d.Roots())
	})
	test(t, "duplicate edge", func(assert *assert.Assertions) {
		d := NewDirected(DummyVertex.Id)
		v1 := DummyVertex("hello")
		v2 := DummyVertex("world")
		d.AddVertex(v1)
		d.AddVertex(v2)
		d.AddEdge(v1.Id(), v2.Id(), nil)
		d.AddEdge(v1.Id(), v2.Id(), nil)
		assert.Equal(
			[]Edge[DummyVertex]{
				{
					Source:      v1,
					Destination: v2,
				},
			},
			d.OutgoingEdges(v1))
	})
}

type DummyVertex string

func (v DummyVertex) Id() string {
	return string(v)
}

func test(t *testing.T, name string, f func(assert *assert.Assertions)) {
	t.Run(name, func(t *testing.T) {
		assert := assert.New(t)
		f(assert)
	})
}
