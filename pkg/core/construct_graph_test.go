package core

import (
	"testing"

	dgraph "github.com/dominikbraun/graph"
	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/graph"
	"github.com/stretchr/testify/assert"
)

func Test_AddConstruct(t *testing.T) {
	assert := assert.New(t)
	g := graph.NewDirected(construct2Hash)
	constructGraph := ConstructGraph{
		underlying: g,
	}
	gw := &Gateway{Name: "test"}
	constructGraph.AddConstruct(gw)
	construct := g.GetVertex("klotho:expose:test")
	storedGw, ok := construct.(*Gateway)
	if !assert.True(ok) {
		return
	}
	assert.Equal(storedGw, gw)
}

func Test_AddDependency(t *testing.T) {
	assert := assert.New(t)
	g := graph.NewDirected(construct2Hash)
	constructGraph := ConstructGraph{
		underlying: g,
	}
	kv := &Kv{Name: "test"}
	eu := &ExecutionUnit{Name: "test"}
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
	g := graph.NewDirected(construct2Hash)
	constructGraph := ConstructGraph{
		underlying: g,
	}
	gw := &Gateway{Name: "test"}
	g.AddVertex(gw)
	construct := constructGraph.GetConstruct(gw.Id())
	storedGw, ok := construct.(*Gateway)
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
	g := graph.NewDirected(construct2Hash)
	constructGraph := ConstructGraph{
		underlying: g,
	}
	kv := &Kv{Name: "test"}
	eu := &ExecutionUnit{Name: "test"}
	g.AddVertex(kv)
	g.AddVertex(eu)
	constructs := ListConstructs[BaseConstruct](&constructGraph)
	expect := []BaseConstruct{kv, eu}
	assert.ElementsMatch(expect, constructs)
}

func Test_ListDependencies(t *testing.T) {
	assert := assert.New(t)
	g := graph.NewDirected(construct2Hash)
	constructGraph := ConstructGraph{
		underlying: g,
	}
	kv := &Kv{Name: "test"}
	eu := &ExecutionUnit{Name: "test"}
	g.AddVertex(kv)
	g.AddVertex(eu)
	constructs := ListConstructs[BaseConstruct](&constructGraph)
	expect := []Construct{kv, eu}
	assert.ElementsMatch(expect, constructs)
}

func Test_GetDownstreamDependencies(t *testing.T) {
	tests := []struct {
		name      string
		construct Construct
		deps      []Construct
		want      []graph.Edge[BaseConstruct]
	}{
		{
			name:      "single dependency",
			construct: &Gateway{Name: "test"},
			deps: []Construct{
				&ExecutionUnit{Name: "test"},
			},
			want: []graph.Edge[BaseConstruct]{
				{
					Source:      &Gateway{Name: "test"},
					Destination: &ExecutionUnit{Name: "test"},
				},
			},
		},
		{
			name:      "multiple dependencies",
			construct: &Gateway{Name: "test"},
			deps: []Construct{
				&ExecutionUnit{Name: "test"},
				&ExecutionUnit{Name: "test2"},
			},
			want: []graph.Edge[BaseConstruct]{
				{
					Source:      &Gateway{Name: "test"},
					Destination: &ExecutionUnit{Name: "test"},
				},
				{
					Source:      &Gateway{Name: "test"},
					Destination: &ExecutionUnit{Name: "test2"},
				},
			},
		},
		{
			name:      "no dependencies",
			construct: &Gateway{Name: "test"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			g := graph.NewDirected(construct2Hash)
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
		construct Construct
		deps      []Construct
		want      []graph.Edge[BaseConstruct]
	}{
		{
			name:      "single dependency",
			construct: &Kv{Name: "test"},
			deps: []Construct{
				&ExecutionUnit{Name: "test"},
			},
			want: []graph.Edge[BaseConstruct]{
				{
					Source:      &ExecutionUnit{Name: "test"},
					Destination: &Kv{Name: "test"},
				},
			},
		},
		{
			name:      "multiple dependencies",
			construct: &Kv{Name: "test"},
			deps: []Construct{
				&ExecutionUnit{Name: "test"},
				&ExecutionUnit{Name: "test2"},
			},
			want: []graph.Edge[BaseConstruct]{
				{
					Source:      &ExecutionUnit{Name: "test"},
					Destination: &Kv{Name: "test"},
				},
				{
					Source:      &ExecutionUnit{Name: "test2"},
					Destination: &Kv{Name: "test"},
				},
			},
		},
		{
			name:      "no dependencies",
			construct: &Kv{Name: "test"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			g := graph.NewDirected(construct2Hash)
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

func Test_GetResourcesOfCapability(t *testing.T) {
	tests := []struct {
		name       string
		constructs []Construct
		capability string
		want       []Construct
	}{
		{
			name:       "single capability construct",
			constructs: []Construct{&Kv{Name: "test"}},
			capability: annotation.PersistCapability,
			want: []Construct{
				&Kv{Name: "test"},
			},
		},
		{
			name: "multiple capability construct",
			constructs: []Construct{
				&Kv{Name: "test"},
				&Orm{Name: "test2"},
			},
			capability: annotation.PersistCapability,
			want: []Construct{
				&Kv{Name: "test"},
				&Orm{Name: "test2"},
			},
		},
		{
			name: "no capability construct",
			constructs: []Construct{
				&Kv{Name: "test"},
				&Orm{Name: "test2"},
			},
			capability: annotation.ExposeCapability,
		},
		{
			name:       "no constructs",
			constructs: []Construct{},
			capability: annotation.ExposeCapability,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			g := graph.NewDirected(construct2Hash)
			constructGraph := ConstructGraph{
				underlying: g,
			}
			for _, c := range tt.constructs {
				g.AddVertex(c)
			}
			constructs := constructGraph.GetResourcesOfCapability(tt.capability)
			if tt.want != nil && !assert.NotNil(constructs) {
				return
			}
			assert.ElementsMatch(tt.want, constructs)
		})
	}

}

func Test_LoadConstructsIntoGraph(t *testing.T) {
	type result struct {
		nodes []BaseConstruct
		edges []graph.Edge[BaseConstruct]
	}
	tests := []struct {
		name       string
		constructs OutputGraph
		want       result
	}{
		{
			name: "single construct",
			constructs: OutputGraph{
				Resources: []ResourceId{
					{Provider: AbstractConstructProvider, Type: "execution_unit", Name: "test_eu"},
					{Provider: AbstractConstructProvider, Type: "static_unit", Name: "test_static"},
					{Provider: AbstractConstructProvider, Type: "expose", Name: "test_expose"},
					{Provider: AbstractConstructProvider, Type: "orm", Name: "test_orm"},
					{Provider: AbstractConstructProvider, Type: "kv", Name: "test_kv"},
					{Provider: AbstractConstructProvider, Type: "fs", Name: "test_fs"},
					{Provider: AbstractConstructProvider, Type: "config", Name: "test_secret"},
					{Provider: AbstractConstructProvider, Type: "redis_node", Name: "test_redis"},
				},
				Edges: []OutputEdge{
					{
						Source:      ResourceId{Provider: AbstractConstructProvider, Type: "execution_unit", Name: "test_eu"},
						Destination: ResourceId{Provider: AbstractConstructProvider, Type: "redis_node", Name: "test_redis"},
					},
					{
						Source:      ResourceId{Provider: AbstractConstructProvider, Type: "expose", Name: "test_expose"},
						Destination: ResourceId{Provider: AbstractConstructProvider, Type: "execution_unit", Name: "test_eu"},
					},
				},
			},
			want: result{
				nodes: []BaseConstruct{
					&ExecutionUnit{Name: "test_eu"},
					&StaticUnit{Name: "test_static"},
					&Gateway{Name: "test_expose"},
					&Orm{Name: "test_orm"},
					&Kv{Name: "test_kv"},
					&Fs{Name: "test_fs"},
					&RedisNode{Name: "test_redis"},
					&Config{Name: "test_secret", Secret: false},
				},
				edges: []graph.Edge[BaseConstruct]{
					{
						Source:      &ExecutionUnit{Name: "test_eu"},
						Destination: &RedisNode{Name: "test_redis"},
						Properties:  dgraph.EdgeProperties{Attributes: map[string]string{}},
					},
					{
						Source:      &Gateway{Name: "test_expose"},
						Destination: &ExecutionUnit{Name: "test_eu"},
						Properties:  dgraph.EdgeProperties{Attributes: map[string]string{}},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			graph := NewConstructGraph()
			err := LoadConstructsIntoGraph(tt.constructs, graph)
			if !assert.NoError(err) {
				return
			}
			assert.ElementsMatch(tt.want.nodes, ListConstructs[BaseConstruct](graph))
			assert.ElementsMatch(tt.want.edges, graph.ListDependencies())
		})
	}
}

func construct2Hash(c BaseConstruct) string {
	return c.Id().String()
}
