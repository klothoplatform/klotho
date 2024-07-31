package kbtesting

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/dominikbraun/graph"
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/knowledgebase"
)

func StringToGraphElement(e string) (any, error) {
	var errs []error
	if !strings.Contains(e, "->") {
		parts := strings.Split(e, ":")
		if len(parts) != 2 {
			errs = append(errs, fmt.Errorf("invalid resource ID %q", e))
		} else {
			return &knowledgebase.ResourceTemplate{
				QualifiedTypeName: e,
			}, nil
		}
	}

	var path construct.Path
	pathErr := path.Parse(e)
	if len(path) > 1 {
		ets := make([]*knowledgebase.EdgeTemplate, len(path)-1)
		for i, id := range path {
			if id.Provider == "" || id.Type == "" {
				return nil, fmt.Errorf("missing provider or type in path element %d", i)
			}
			if i == 0 {
				continue
			}
			ets[i-1] = &knowledgebase.EdgeTemplate{
				Source: path[i-1],
				Target: id,
			}
		}
		return ets, nil
	} else if pathErr == nil {
		pathErr = fmt.Errorf("path must have at least two elements (got %d)", len(path))
	}
	errs = append(errs, pathErr)

	return nil, errors.Join(errs...)
}

// AddElement is a utility function for adding an element to a graph. See [MakeGraph] for more information on supported
// element types. Returns whether adding the element failed.
func AddElement(t *testing.T, g knowledgebase.Graph, e any) (failed bool) {
	must := func(err error) {
		if err != nil {
			t.Error(err)
			failed = true
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

	addIfMissing := func(res *knowledgebase.ResourceTemplate) {
		err := g.AddVertex(res)
		if err != nil && !errors.Is(err, graph.ErrVertexAlreadyExists) {
			t.Errorf("could add vertex %s: %v", res.QualifiedTypeName, err)
			failed = true
		}
	}

	addEdge := func(e *knowledgebase.EdgeTemplate) {
		must(g.AddEdge(e.Source.QualifiedTypeName(), e.Target.QualifiedTypeName(), graph.EdgeData(e)))
	}

	switch e := e.(type) {
	case knowledgebase.ResourceTemplate:
		addIfMissing(&e)

	case *knowledgebase.ResourceTemplate:
		addIfMissing(e)

	case knowledgebase.EdgeTemplate:
		addIfMissing(&knowledgebase.ResourceTemplate{QualifiedTypeName: e.Source.QualifiedTypeName()})
		addIfMissing(&knowledgebase.ResourceTemplate{QualifiedTypeName: e.Target.QualifiedTypeName()})
		addEdge(&e)

	case *knowledgebase.EdgeTemplate:
		addIfMissing(&knowledgebase.ResourceTemplate{QualifiedTypeName: e.Source.QualifiedTypeName()})
		addIfMissing(&knowledgebase.ResourceTemplate{QualifiedTypeName: e.Target.QualifiedTypeName()})
		addEdge(e)

	case []*knowledgebase.EdgeTemplate:
		for _, edge := range e {
			addIfMissing(&knowledgebase.ResourceTemplate{QualifiedTypeName: edge.Source.QualifiedTypeName()})
			addIfMissing(&knowledgebase.ResourceTemplate{QualifiedTypeName: edge.Target.QualifiedTypeName()})
			addEdge(edge)
		}
	default:
		t.Errorf("invalid element of type %T", e)
		return true
	}

	return
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
func MakeKB(t *testing.T, elements ...any) *knowledgebase.KnowledgeBase {
	kb := knowledgebase.NewKB()
	failed := false
	for i, e := range elements {
		elemFailed := AddElement(t, kb.Graph(), e)
		if elemFailed {
			t.Errorf("failed to add element[%d] (%v) to graph", i, e)
			failed = true
		}
	}
	if failed {
		// Fail now because if the graph didn't parse correctly, then the rest of the test is likely to fail
		t.FailNow()
	}

	return kb
}
