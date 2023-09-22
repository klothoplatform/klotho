package construct2

import (
	"errors"
	"testing"

	"github.com/dominikbraun/graph"
	"github.com/stretchr/testify/assert"
)

func AssertGraphEqual(t *testing.T, expect, actual Graph) {
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
	eStr := must(String(expect))
	aStr := must(String(actual))
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
func MakeGraph(t *testing.T, g Graph, elements ...any) Graph {
	must := func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, e := range elements {
		switch e := e.(type) {
		case ResourceId:
			must(g.AddVertex(&Resource{ID: e}))

		case Resource:
			must(g.AddVertex(&e))

		case *Resource:
			must(g.AddVertex(e))

		case Edge:
			must(g.AddEdge(e.Source, e.Target))

		case string:
			var id ResourceId
			idErr := id.UnmarshalText([]byte(e))
			if idErr == nil {
				must(g.AddVertex(&Resource{ID: id}))
				continue
			}
			var edge ioEdge
			edgeErr := edge.UnmarshalText([]byte(e))
			if edgeErr == nil {
				if _, getErr := g.Vertex(edge.Source); errors.Is(getErr, graph.ErrVertexNotFound) {
					must(g.AddVertex(&Resource{ID: edge.Source}))
				}
				if _, getErr := g.Vertex(edge.Target); errors.Is(getErr, graph.ErrVertexNotFound) {
					must(g.AddVertex(&Resource{ID: edge.Target}))
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

func TestHash(t *testing.T) {
	tests := []struct {
		name    string
		args    Graph
		want    []byte
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
		})
	}
}
