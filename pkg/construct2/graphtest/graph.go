package graphtest

import (
	"errors"
	"fmt"
	"testing"

	"github.com/dominikbraun/graph"
	"github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/stretchr/testify/assert"
)

func AssertGraphEqual(t *testing.T, expect, actual construct2.Graph, message string, args ...any) {
	assert := assert.New(t)
	must := func(v any, err error) any {
		if err != nil {
			t.Fatal(err)
		}
		return v
	}

	msg := func(subMessage string) []any {
		if message == "" {
			return []any{subMessage}
		}
		return append([]any{message + ": " + subMessage}, args...)
	}

	assert.Equal(must(expect.Order()), must(actual.Order()), msg("order (# of nodes) mismatch")...)
	assert.Equal(must(expect.Size()), must(actual.Size()), msg("size (# of edges) mismatch")...)

	// Use the string representation to compare the graphs so that the diffs are nicer
	eStr := must(construct2.String(expect))
	aStr := must(construct2.String(actual))
	assert.Equal(eStr, aStr, msg("graph mismatch")...)
}

func AssertGraphContains(t *testing.T, expect, actual construct2.Graph) {
	assert := assert.New(t)
	must := func(v any, err error) any {
		if err != nil {
			t.Fatal(err)
		}
		return v
	}

	expectVs := must(construct2.TopologicalSort(expect)).([]construct2.ResourceId)
	for _, expectV := range expectVs {
		_, err := actual.Vertex(expectV)
		assert.NoError(err)
	}

	expectEs := must(expect.Edges()).([]construct2.Edge)
	for _, expectE := range expectEs {
		_, err := actual.Edge(expectE.Source, expectE.Target)
		assert.NoError(err)
	}
}

func stringToGraphElement(e string) (any, error) {
	var id construct2.ResourceId
	idErr := id.UnmarshalText([]byte(e))
	if idErr == nil {
		return id, nil
	}
	var edge construct2.SimpleEdge
	edgeErr := edge.UnmarshalText([]byte(e))
	if edgeErr == nil {
		return edge, nil
	}

	var path construct2.Path
	pathErr := path.UnmarshalText([]byte(e))
	if pathErr == nil {
		return path, nil
	}

	return nil, errors.Join(idErr, edgeErr, pathErr)
}

// MakeGraph is a utility function for creating a graph from a list of elements which can be of types:
// - ResourceId : adds an empty resource with the given ID
// - Resource, *Resource : adds the given resource
// - Edge : adds the given edge
// - Path : adds all the edges in the path
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
	addIfMissing := func(res *construct2.Resource) {
		if _, err := g.Vertex(res.ID); errors.Is(err, graph.ErrVertexNotFound) {
			must(g.AddVertex(res))
		} else if err != nil {
			t.Fatal(fmt.Errorf("could check vertex %s: %w", res.ID, err))
		}
	}
	failed := false
	for i, e := range elements {
		if estr, ok := e.(string); ok {
			var err error
			e, err = stringToGraphElement(estr)
			if err != nil {
				t.Errorf("invalid element[%d] %q (type %[2]T) Parse errors: %v", i, e, err)
				failed = true
			}
		}
		switch e := e.(type) {
		case construct2.ResourceId:
			addIfMissing(&construct2.Resource{ID: e})

		case construct2.Resource:
			must(g.AddVertex(&e))

		case *construct2.Resource:
			must(g.AddVertex(e))

		case construct2.Edge:
			addIfMissing(&construct2.Resource{ID: e.Source})
			addIfMissing(&construct2.Resource{ID: e.Target})
			must(g.AddEdge(e.Source, e.Target))

		case construct2.ResourceEdge:
			addIfMissing(e.Source)
			addIfMissing(e.Target)
			must(g.AddEdge(e.Source.ID, e.Target.ID))

		case construct2.SimpleEdge:
			addIfMissing(&construct2.Resource{ID: e.Source})
			addIfMissing(&construct2.Resource{ID: e.Target})
			must(g.AddEdge(e.Source, e.Target))

		case construct2.Path:
			for i, id := range e {
				addIfMissing(&construct2.Resource{ID: id})
				if i > 0 {
					must(g.AddEdge(e[i-1], id))
				}
			}
		default:
			t.Errorf("invalid element[%d] of type %T", i, e)
			failed = true
		}
	}
	if failed {
		// Fail now because if the graph didn't parse correctly, then the rest of the test is likely to fail
		t.FailNow()
	}

	return g
}
