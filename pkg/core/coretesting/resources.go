package coretesting

import (
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
	var res []string
	for _, r := range dag.ListResources() {
		res = append(res, r.Id())
	}
	if expect.AssertSubset {
		assert.Subset(t, res, expect.Nodes)
	} else {
		assert.ElementsMatch(t, expect.Nodes, res)
	}

	var dep []StringDep
	for _, e := range dag.ListDependencies() {
		dep = append(dep, StringDep{Source: e.Source.Id(), Destination: e.Destination.Id()})
	}

	if expect.AssertSubset {
		assert.Subset(t, dep, expect.Deps)
	} else {
		assert.ElementsMatch(t, expect.Deps, dep)
	}
}
