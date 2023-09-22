package graphtest

import (
	"errors"
	"testing"

	"github.com/dominikbraun/graph"
	"github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/stretchr/testify/assert"
)

func AssertGraphEqual(t *testing.T, expect, actual construct2.Graph) {
	assert := assert.New(t)
	must := func(v any, err error) any {
		if err != nil {
			t.Fatal(err)
		}
		return v
	}

	assert.Equal(must(expect.Order()), must(actual.Order()), "order (# of nodes) mismatch")
	assert.Equal(must(expect.Size()), must(actual.Size()), "size (# of edges) mismatch")

	// Use the string representation to compare the graphs so that the diffs are nicer
	eStr := must(construct2.String(expect))
	aStr := must(construct2.String(actual))
	assert.Equal(eStr, aStr)
}

// MakeGraph is a utility function for creating a graph from a list of elements which can be of types:
// - ResourceId : adds an empty resource with the given ID
// - Resource, *Resource : adds the given resource
// - Edge : adds the given edge
// - string : parses the string as either a ResourceId or an Edge and add it as above
//
// The input graph is so it can be either via NewGraph or NewAcyclicGraph.
// Users are encouraged to wrap this function for the specific test function for ease of use, such as:
//
//	makeGraph := func(elements ...any) Graph {
//		return MakeGraph(t, NewGraph(), elements...)
//	}
func MakeGraph(t *testing.T, g construct2.Graph, elements ...any) construct2.Graph {
	must := func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, e := range elements {
		switch e := e.(type) {
		case construct2.ResourceId:
			must(g.AddVertex(&construct2.Resource{ID: e}))

		case construct2.Resource:
			must(g.AddVertex(&e))

		case *construct2.Resource:
			must(g.AddVertex(e))

		case construct2.Edge:
			must(g.AddEdge(e.Source, e.Target))

		case string:
			var id construct2.ResourceId
			idErr := id.UnmarshalText([]byte(e))
			if idErr == nil {
				must(g.AddVertex(&construct2.Resource{ID: id}))
				continue
			}
			var edge construct2.IoEdge
			edgeErr := edge.UnmarshalText([]byte(e))
			if edgeErr == nil {
				if _, getErr := g.Vertex(edge.Source); errors.Is(getErr, graph.ErrVertexNotFound) {
					must(g.AddVertex(&construct2.Resource{ID: edge.Source}))
				}
				if _, getErr := g.Vertex(edge.Target); errors.Is(getErr, graph.ErrVertexNotFound) {
					must(g.AddVertex(&construct2.Resource{ID: edge.Target}))
				}
				must(g.AddEdge(edge.Source, edge.Target))
				continue
			}

			t.Fatalf("invalid element %q (type %[1]T) Parse errors: %v", e, errors.Join(idErr, edgeErr))
		default:
		}
	}

	return g
}
