package operational_eval

import (
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/construct2/graphtest"
	"github.com/klothoplatform/klotho/pkg/engine/enginetesting"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledgebase"
	"github.com/klothoplatform/klotho/pkg/set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestEvaluator_isEvaluated(t *testing.T) {
	tests := []struct {
		name          string
		vertex        Vertex
		inGraph       bool
		inUnevaluated bool
		want          bool
		wantErr       bool
	}{
		{
			name: "simple evaluated vertex",
			vertex: &propertyVertex{
				Ref: construct.PropertyRef{
					Resource: graphtest.ParseId(t, "a:a:a"),
					Property: "prop1",
				},
			},
			inGraph: true,
			want:    true,
		},
		{
			name: "simple unevaluated vertex",
			vertex: &propertyVertex{
				Ref: construct.PropertyRef{
					Resource: graphtest.ParseId(t, "a:a:a"),
					Property: "prop1",
				},
			},
			inGraph:       true,
			inUnevaluated: true,
			want:          false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			testSol := enginetesting.NewTestSolution()
			eval := NewEvaluator(testSol)
			if tt.inGraph {
				err := eval.graph.AddVertex(tt.vertex)
				assert.NoError(err)
			}
			if tt.inUnevaluated {
				err := eval.unevaluated.AddVertex(tt.vertex)
				assert.NoError(err)
			}
			got, err := eval.isEvaluated(tt.vertex.Key())
			if tt.wantErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)
			assert.Equal(tt.want, got)
		})
	}
}

func TestEvaluator_addEdge(t *testing.T) {
	tests := []struct {
		name         string
		initialState []Vertex
		source       Key
		target       Key
		wantErr      bool
	}{
		{
			name: "simple add edge",
			initialState: []Vertex{
				&propertyVertex{
					Ref: construct.PropertyRef{
						Resource: graphtest.ParseId(t, "a:a:a"),
						Property: "prop1",
					},
				},
				&propertyVertex{
					Ref: construct.PropertyRef{
						Resource: graphtest.ParseId(t, "a:a:a"),
						Property: "prop2",
					},
				},
			},
			source: Key{Ref: construct.PropertyRef{
				Resource: graphtest.ParseId(t, "a:a:a"),
				Property: "prop1",
			}},
			target: Key{Ref: construct.PropertyRef{
				Resource: graphtest.ParseId(t, "a:a:a"),
				Property: "prop2",
			}},
		},
		{
			name: "add edge with missing source",
			initialState: []Vertex{
				&propertyVertex{
					Ref: construct.PropertyRef{
						Resource: graphtest.ParseId(t, "a:a:a"),
						Property: "prop2",
					},
				},
			},
			source: Key{Ref: construct.PropertyRef{
				Resource: graphtest.ParseId(t, "a:a:a"),
				Property: "prop1",
			}},
			target: Key{Ref: construct.PropertyRef{
				Resource: graphtest.ParseId(t, "a:a:a"),
				Property: "prop2",
			}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			testSol := enginetesting.NewTestSolution()
			eval := NewEvaluator(testSol)
			for _, v := range tt.initialState {
				err := eval.graph.AddVertex(v)
				assert.NoError(err)
				err = eval.unevaluated.AddVertex(v)
				assert.NoError(err)
			}
			err := eval.addEdge(tt.source, tt.target)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)
			_, err = eval.graph.Edge(tt.source, tt.target)
			assert.NoError(err)
			_, err = eval.unevaluated.Edge(tt.source, tt.target)
			assert.NoError(err)
		})
	}
}

func TestEvaluator_enqueue(t *testing.T) {
	tests := []struct {
		name         string
		changes      graphChanges
		initialState []Vertex
		want         map[Key][]Key
		wantErr      bool
	}{
		{
			name: "simple enqueue",
			initialState: []Vertex{
				&propertyVertex{
					Ref: construct.PropertyRef{
						Resource: graphtest.ParseId(t, "a:a:a"),
						Property: "prop2",
					},
				},
			},
			changes: graphChanges{
				nodes: map[Key]Vertex{
					{Ref: construct.PropertyRef{
						Resource: graphtest.ParseId(t, "a:a:a"),
						Property: "prop1",
					}}: &propertyVertex{
						Ref: construct.PropertyRef{
							Resource: graphtest.ParseId(t, "a:a:a"),
							Property: "prop1",
						},
					},
				},
				edges: map[Key]set.Set[Key]{

					{Ref: construct.PropertyRef{
						Resource: graphtest.ParseId(t, "a:a:a"),
						Property: "prop1",
					}}: set.SetOf[Key](
						Key{Ref: construct.PropertyRef{
							Resource: graphtest.ParseId(t, "a:a:a"),
							Property: "prop2",
						}},
					),
				},
			},
			want: map[Key][]Key{
				{Ref: construct.PropertyRef{
					Resource: graphtest.ParseId(t, "a:a:a"),
					Property: "prop1",
				}}: []Key{
					{Ref: construct.PropertyRef{
						Resource: graphtest.ParseId(t, "a:a:a"),
						Property: "prop2",
					}},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			testSol := enginetesting.NewTestSolution()
			eval := NewEvaluator(testSol)
			for _, v := range tt.initialState {
				err := eval.graph.AddVertex(v)
				assert.NoError(err)
			}
			err := eval.enqueue(tt.changes)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)
			adj, err := eval.graph.AdjacencyMap()
			assert.NoError(err)
			for k, v := range tt.want {
				p := adj[k]
				for _, dep := range v {
					assert.Contains(p, dep)
				}
			}
		})
	}
}

func TestEvaluator_UpdateId(t *testing.T) {

	tests := []struct {
		name         string
		initialGraph []any
		initialState []Vertex
		oldId        construct.ResourceId
		newId        construct.ResourceId
		want         map[Key]Vertex
		wantGraph    []any
		wantErr      bool
	}{
		{
			name:         "simple update with property vertex",
			initialGraph: []any{"a:a:a"},
			initialState: []Vertex{
				&propertyVertex{
					Ref: construct.PropertyRef{
						Resource: graphtest.ParseId(t, "a:a:a"),
						Property: "prop1",
					},
				},
			},
			oldId: graphtest.ParseId(t, "a:a:a"),
			newId: graphtest.ParseId(t, "b:b:b"),
			want: map[Key]Vertex{
				{Ref: construct.PropertyRef{
					Resource: graphtest.ParseId(t, "b:b:b"),
					Property: "prop1",
				}}: &propertyVertex{
					Ref: construct.PropertyRef{
						Resource: graphtest.ParseId(t, "b:b:b"),
						Property: "prop1",
					},
				},
			},
			wantGraph: []any{"b:b:b"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			testSol := enginetesting.NewTestSolution()
			testSol.KB.On("GetResourceTemplate", mock.Anything).Return(&knowledgebase.ResourceTemplate{}, nil)
			newG := graphtest.MakeGraph(t, construct.NewGraph(), tt.initialGraph...)
			err := testSol.RawView().AddVerticesFrom(newG)
			assert.NoError(err)
			err = testSol.RawView().AddVerticesFrom(newG)
			assert.NoError(err)
			eval := NewEvaluator(testSol)
			for _, v := range tt.initialState {
				err := eval.graph.AddVertex(v)
				assert.NoError(err)
			}
			err = eval.UpdateId(tt.oldId, tt.newId)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)
			adj, err := eval.graph.AdjacencyMap()
			assert.NoError(err)
			assert.Equal(len(adj), len(tt.want))
			for k, v := range tt.want {
				actual, err := eval.graph.Vertex(k)
				assert.NoError(err)
				assert.Equal(v, actual)
			}
			wantGraph := graphtest.MakeGraph(t, construct.NewGraph(), tt.wantGraph...)
			graphtest.AssertGraphEqual(t, testSol.RawView(), wantGraph, "graph")
		})
	}
}
