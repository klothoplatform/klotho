package construct

import (
	"testing"

	orig_graph "github.com/dominikbraun/graph"
	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/graph"
	"github.com/stretchr/testify/assert"
)

type TestConstruct struct {
	Name string
}

func (tc *TestConstruct) Id() ResourceId {
	return ResourceId{
		Provider: "test",
		Type:     "test",
		Name:     tc.Name,
	}
}

var emptyProperties = orig_graph.EdgeProperties{Attributes: make(map[string]string)}

func Test_AddConstruct(t *testing.T) {
	assert := assert.New(t)
	g := graph.NewDirected(construct2Hash, false)
	constructGraph := ConstructGraph{
		underlying: g,
	}
	gw := &TestConstruct{Name: "test"}
	constructGraph.AddConstruct(gw)
	construct := g.GetVertex("test:test:test")
	storedGw, ok := construct.(*TestConstruct)
	if !assert.True(ok) {
		return
	}
	assert.Equal(storedGw, gw)
}

func Test_AddDependency(t *testing.T) {
	assert := assert.New(t)
	g := graph.NewDirected(construct2Hash, false)
	constructGraph := ConstructGraph{
		underlying: g,
	}
	kv := &TestConstruct{Name: "testkv"}
	eu := &TestConstruct{Name: "testeu"}
	g.AddVertex(kv)
	g.AddVertex(eu)
	constructGraph.AddDependency(eu.Id(), kv.Id())
	edge := g.GetEdge(eu.Id().String(), kv.Id().String())
	if !assert.NotNil(edge) {
		return
	}
	assert.Equal(edge.Source, eu)
	assert.Equal(edge.Destination, kv)
}

func Test_GetConstruct(t *testing.T) {
	assert := assert.New(t)
	g := graph.NewDirected(construct2Hash, false)
	constructGraph := ConstructGraph{
		underlying: g,
	}
	gw := &TestConstruct{Name: "test"}
	g.AddVertex(gw)
	construct := constructGraph.GetConstruct(gw.Id())
	storedGw, ok := construct.(*TestConstruct)
	if !assert.True(ok) {
		return
	}
	assert.Equal(storedGw, gw)
	nilConstruct := constructGraph.GetConstruct(ResourceId{
		Provider: AbstractConstructProvider,
		Type:     annotation.ExposeCapability,
		Name:     "fake",
	})
	assert.Nil(nilConstruct)
}

func Test_ListConstructs(t *testing.T) {
	assert := assert.New(t)
	g := graph.NewDirected(construct2Hash, false)
	constructGraph := ConstructGraph{
		underlying: g,
	}
	kv := &TestConstruct{Name: "testkv"}
	eu := &TestConstruct{Name: "testeu"}
	g.AddVertex(kv)
	g.AddVertex(eu)
	constructs := ListConstructs[BaseConstruct](&constructGraph)
	expect := []BaseConstruct{kv, eu}
	assert.ElementsMatch(expect, constructs)
}

func Test_ListDependencies(t *testing.T) {
	assert := assert.New(t)
	g := graph.NewDirected(construct2Hash, false)
	constructGraph := ConstructGraph{
		underlying: g,
	}
	kv := &TestConstruct{Name: "testkv"}
	eu := &TestConstruct{Name: "testeu"}
	g.AddVertex(kv)
	g.AddVertex(eu)
	constructs := ListConstructs[BaseConstruct](&constructGraph)
	expect := []BaseConstruct{kv, eu}
	assert.ElementsMatch(expect, constructs)
}

func Test_GetDownstreamDependencies(t *testing.T) {
	tests := []struct {
		name      string
		construct BaseConstruct
		deps      []BaseConstruct
		want      []graph.Edge[BaseConstruct]
	}{
		{
			name:      "single dependency",
			construct: &TestConstruct{Name: "testkv"},
			deps: []BaseConstruct{
				&TestConstruct{Name: "testeu"},
			},
			want: []graph.Edge[BaseConstruct]{
				{
					Source:      &TestConstruct{Name: "testkv"},
					Destination: &TestConstruct{Name: "testeu"},
					Properties:  emptyProperties,
				},
			},
		},
		{
			name:      "multiple dependencies",
			construct: &TestConstruct{Name: "test"},
			deps: []BaseConstruct{
				&TestConstruct{Name: "test1"},
				&TestConstruct{Name: "test2"},
			},
			want: []graph.Edge[BaseConstruct]{
				{
					Source:      &TestConstruct{Name: "test"},
					Destination: &TestConstruct{Name: "test1"},
					Properties:  emptyProperties,
				},
				{
					Source:      &TestConstruct{Name: "test"},
					Destination: &TestConstruct{Name: "test2"},
					Properties:  emptyProperties,
				},
			},
		},
		{
			name:      "no dependencies",
			construct: &TestConstruct{Name: "test"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			g := graph.NewDirected(construct2Hash, false)
			constructGraph := ConstructGraph{
				underlying: g,
			}
			g.AddVertex(tt.construct)
			for _, c := range tt.deps {
				g.AddVertex(c)
				g.AddEdge(tt.construct.Id().String(), c.Id().String(), nil)
			}
			deps := constructGraph.GetDownstreamDependencies(tt.construct)
			if tt.want != nil && !assert.NotNil(deps) {
				return
			}
			assert.ElementsMatch(tt.want, deps)
			dConstructs := constructGraph.GetDownstreamConstructs(tt.construct)
			if tt.want != nil && !assert.NotNil(dConstructs) {
				return
			}
			assert.ElementsMatch(tt.deps, dConstructs)
		})
	}

}

func Test_GetUpstreamDependencies(t *testing.T) {
	tests := []struct {
		name      string
		construct BaseConstruct
		deps      []BaseConstruct
		want      []graph.Edge[BaseConstruct]
	}{
		{
			name:      "single dependency",
			construct: &TestConstruct{Name: "test"},
			deps: []BaseConstruct{
				&TestConstruct{Name: "test1"},
			},
			want: []graph.Edge[BaseConstruct]{
				{
					Source:      &TestConstruct{Name: "test1"},
					Destination: &TestConstruct{Name: "test"},
					Properties:  emptyProperties,
				},
			},
		},
		{
			name:      "multiple dependencies",
			construct: &TestConstruct{Name: "test"},
			deps: []BaseConstruct{
				&TestConstruct{Name: "test1"},
				&TestConstruct{Name: "test2"},
			},
			want: []graph.Edge[BaseConstruct]{
				{
					Source:      &TestConstruct{Name: "test1"},
					Destination: &TestConstruct{Name: "test"},
					Properties:  emptyProperties,
				},
				{
					Source:      &TestConstruct{Name: "test2"},
					Destination: &TestConstruct{Name: "test"},
					Properties:  emptyProperties,
				},
			},
		},
		{
			name:      "no dependencies",
			construct: &TestConstruct{Name: "test"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			g := graph.NewDirected(construct2Hash, false)
			constructGraph := ConstructGraph{
				underlying: g,
			}
			g.AddVertex(tt.construct)
			for _, c := range tt.deps {
				g.AddVertex(c)
				g.AddEdge(c.Id().String(), tt.construct.Id().String(), nil)
			}
			deps := constructGraph.GetUpstreamDependencies(tt.construct)
			if tt.want != nil && !assert.NotNil(deps) {
				return
			}
			assert.ElementsMatch(tt.want, deps)
			dConstructs := constructGraph.GetUpstreamConstructs(tt.construct)
			if tt.want != nil && !assert.NotNil(dConstructs) {
				return
			}
			assert.ElementsMatch(tt.deps, dConstructs)
		})
	}

}

// func Test_GetResourcesOfCapability(t *testing.T) {
// 	tests := []struct {
// 		name       string
// 		constructs []Base
// 		capability string
// 		want       []Construct
// 	}{
// 		{
// 			name:       "single capability construct",
// 			constructs: []Construct{&Kv{Name: "test"}},
// 			capability: annotation.PersistCapability,
// 			want: []Construct{
// 				&Kv{Name: "test"},
// 			},
// 		},
// 		{
// 			name: "multiple capability construct",
// 			constructs: []Construct{
// 				&Kv{Name: "test"},
// 				&Orm{Name: "test2"},
// 			},
// 			capability: annotation.PersistCapability,
// 			want: []Construct{
// 				&Kv{Name: "test"},
// 				&Orm{Name: "test2"},
// 			},
// 		},
// 		{
// 			name: "no capability construct",
// 			constructs: []Construct{
// 				&Kv{Name: "test"},
// 				&Orm{Name: "test2"},
// 			},
// 			capability: annotation.ExposeCapability,
// 		},
// 		{
// 			name:       "no constructs",
// 			constructs: []Construct{},
// 			capability: annotation.ExposeCapability,
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			assert := assert.New(t)
// 			g := graph.NewDirected(construct2Hash)
// 			constructGraph := ConstructGraph{
// 				underlying: g,
// 			}
// 			for _, c := range tt.constructs {
// 				g.AddVertex(c)
// 			}
// 			constructs := constructGraph.GetResourcesOfCapability(tt.capability)
// 			if tt.want != nil && !assert.NotNil(constructs) {
// 				return
// 			}
// 			assert.ElementsMatch(tt.want, constructs)
// 		})
// 	}

// }

func construct2Hash(c BaseConstruct) string {
	return c.Id().String()
}
