package graphtest

import (
	"errors"
	"fmt"
	"testing"

	"github.com/dominikbraun/graph"
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/stretchr/testify/assert"
)

func AssertGraphEqual(t *testing.T, expect, actual construct.Graph, message string, args ...any) {
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
	eStr := must(construct.String(expect))
	aStr := must(construct.String(actual))
	assert.Equal(eStr, aStr, msg("graph mismatch")...)
}

func AssertGraphContains(t *testing.T, expect, actual construct.Graph) {
	assert := assert.New(t)
	must := func(v any, err error) any {
		if err != nil {
			t.Fatal(err)
		}
		return v
	}

	expectVs := must(construct.TopologicalSort(expect)).([]construct.ResourceId)
	for _, expectV := range expectVs {
		_, err := actual.Vertex(expectV)
		assert.NoError(err)
	}

	expectEs := must(expect.Edges()).([]construct.Edge)
	for _, expectE := range expectEs {
		_, err := actual.Edge(expectE.Source, expectE.Target)
		assert.NoError(err)
	}
}

func StringToGraphElement(e string) (any, error) {
	var id construct.ResourceId
	idErr := id.Parse(e)
	if id.Validate() == nil {
		return id, nil
	}

	var path construct.Path
	pathErr := path.Parse(e)
	if len(path) > 0 {
		return path, nil
	}

	return nil, errors.Join(idErr, pathErr)
}

// AddElement is a utility function for adding an element to a graph. See [MakeGraph] for more information on supported
// element types. Returns whether adding the element failed.
func AddElement(t *testing.T, g construct.Graph, e any) (failed bool) {
	must := func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}
	if estr, ok := e.(string); ok {
		var err error
		e, err = StringToGraphElement(estr)
		if err != nil {
			t.Errorf("invalid element %q (type %[1]T) Parse errors: %v", e, err)
			return true
		}
	}

	addIfMissing := func(res *construct.Resource) {
		if _, err := g.Vertex(res.ID); errors.Is(err, graph.ErrVertexNotFound) {
			must(g.AddVertex(res))
		} else if err != nil {
			t.Fatal(fmt.Errorf("could check vertex %s: %w", res.ID, err))
		}
	}

	switch e := e.(type) {
	case construct.ResourceId:
		addIfMissing(&construct.Resource{ID: e})

	case construct.Resource:
		must(g.AddVertex(&e))

	case *construct.Resource:
		must(g.AddVertex(e))

	case construct.Edge:
		addIfMissing(&construct.Resource{ID: e.Source})
		addIfMissing(&construct.Resource{ID: e.Target})
		must(g.AddEdge(e.Source, e.Target))

	case construct.ResourceEdge:
		addIfMissing(e.Source)
		addIfMissing(e.Target)
		must(g.AddEdge(e.Source.ID, e.Target.ID))

	case construct.SimpleEdge:
		addIfMissing(&construct.Resource{ID: e.Source})
		addIfMissing(&construct.Resource{ID: e.Target})
		must(g.AddEdge(e.Source, e.Target))

	case construct.Path:
		for i, id := range e {
			addIfMissing(&construct.Resource{ID: id})
			if i > 0 {
				must(g.AddEdge(e[i-1], id))
			}
		}
	default:
		t.Errorf("invalid element of type %T", e)
		return true
	}
	return false
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
func MakeGraph(t *testing.T, g construct.Graph, elements ...any) construct.Graph {
	failed := false
	for i, e := range elements {
		elemFailed := AddElement(t, g, e)
		if elemFailed {
			t.Errorf("failed to add element[%d] (%v) to graph", i, e)
			failed = true
		}
	}
	if failed {
		// Fail now because if the graph didn't parse correctly, then the rest of the test is likely to fail
		t.FailNow()
	}

	return g
}
