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

		// OnlyContains only check whether the dag contains the `.Nodes` and `.Deps`. If false,
		// checks full equality.
		OnlyContains bool
	}
)

func (expect ResourcesExpectation) Assert(t *testing.T, dag *core.ResourceGraph) {
	var res []string
	for _, r := range dag.ListResources() {
		res = append(res, r.Id())
	}
	if expect.OnlyContains {
		for _, r := range expect.Nodes {
			assert.Contains(t, res, r)
		}
	} else {
		assert.ElementsMatch(t, expect.Nodes, res)
	}

	var dep []StringDep
	for _, e := range dag.ListDependencies() {
		dep = append(dep, StringDep{Source: e.Source.Id(), Destination: e.Destination.Id()})
	}

	if expect.OnlyContains {
		for _, d := range expect.Deps {
			assert.Contains(t, dep, d)
		}
	} else {
		assert.ElementsMatch(t, expect.Deps, dep)
	}
}
