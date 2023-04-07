package coretesting

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/graph"
	"github.com/stretchr/testify/assert"
)

type (
	StringDep = graph.Edge[string]

	ResourcesExpectation struct {
		Nodes []string
		Deps  []StringDep

		// AssertSubset assert the dag contains all the `.Nodes` and `.Deps`. If false,
		// checks full equality.
		AssertSubset bool
	}
)

func (expect ResourcesExpectation) Assert(t *testing.T, dag *core.ResourceGraph) {
	got := ResoucesFromDAG(dag)

	if expect.AssertSubset {
		assert.Subset(t, got.Nodes, expect.Nodes)
		assert.Subset(t, got.Deps, expect.Deps)
	} else {
		assert.ElementsMatch(t, expect.Nodes, got.Nodes)
		assert.ElementsMatch(t, expect.Deps, got.Deps)
	}
}

func ResoucesFromDAG(dag *core.ResourceGraph) ResourcesExpectation {
	var nodes []string
	for _, r := range dag.ListResources() {
		nodes = append(nodes, r.Id())
	}
	var deps []StringDep
	for _, e := range dag.ListDependencies() {
		deps = append(deps, StringDep{Source: e.Source.Id(), Destination: e.Destination.Id()})
	}

	return ResourcesExpectation{
		Nodes: nodes,
		Deps:  deps,
	}
}

// GoString is useful in combination with `ResoucesFromDAG` to generate or update unit tests. Make sure to read over
// the results before using to make sure it is correct.
// For example:
//
//	fmt.Print(coretesting.ResoucesFromDAG(dag).GoString())
func (expect ResourcesExpectation) GoString() string {
	buf := new(strings.Builder)
	buf.WriteString("coretesting.ResourcesExpectation{\n")

	nodes := make([]string, len(expect.Nodes))
	copy(nodes, expect.Nodes)
	sort.Strings(nodes)
	buf.WriteString("	Nodes: []string{\n")
	for _, n := range nodes {
		fmt.Fprintf(buf, "		%s,\n", strconv.Quote(n))
	}
	buf.WriteString("	},\n")

	edges := make([]StringDep, len(expect.Deps))
	copy(edges, expect.Deps)
	sort.SliceStable(edges, func(i, j int) bool {
		a, b := edges[i], edges[j]
		if a.Source == b.Source {
			return a.Destination < b.Destination
		}
		return a.Source < b.Source
	})
	buf.WriteString("	Deps: []coretesting.StringDep{\n")
	for _, e := range edges {
		fmt.Fprintf(buf, "		{Source: %s, Destination: %s},\n", strconv.Quote(e.Source), strconv.Quote(e.Destination))
	}
	buf.WriteString("	},\n")

	buf.WriteString("}\n")

	return buf.String()
}