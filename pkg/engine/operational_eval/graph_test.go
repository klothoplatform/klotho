package operational_eval

import (
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/construct/graphtest"
	"github.com/klothoplatform/klotho/pkg/engine/enginetesting"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledgebase"
	"github.com/klothoplatform/klotho/pkg/knowledgebase/properties"
	"github.com/klothoplatform/klotho/pkg/set"
	"github.com/stretchr/testify/assert"
)

func TestEvaluator_resourceVertices(t *testing.T) {
	tests := []struct {
		name    string
		res     *construct.Resource
		tmpl    *knowledgebase.ResourceTemplate
		want    graphChanges
		wantErr bool
	}{
		{
			name: "simple resource",
			res: &construct.Resource{
				ID: graphtest.ParseId(t, "a:a:a"),
				Properties: map[string]any{
					"prop1": "value1",
				},
			},
			tmpl: &knowledgebase.ResourceTemplate{
				Properties: map[string]knowledgebase.Property{
					"prop1": &properties.StringProperty{
						PropertyDetails: knowledgebase.PropertyDetails{
							Path: "prop1",
						},
					},
				},
			},
			want: graphChanges{
				nodes: map[Key]Vertex{
					{Ref: construct.PropertyRef{
						Resource: graphtest.ParseId(t, "a:a:a"),
						Property: "prop1",
					}}: &propertyVertex{
						Ref: construct.PropertyRef{
							Resource: graphtest.ParseId(t, "a:a:a"),
							Property: "prop1",
						},
						Template: &properties.StringProperty{
							PropertyDetails: knowledgebase.PropertyDetails{
								Path: "prop1",
							},
						},
						EdgeRules: map[construct.SimpleEdge][]knowledgebase.OperationalRule{},
					},
				},
				edges: map[Key]set.Set[Key]{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			testSol := enginetesting.NewTestSolution()
			err := testSol.DataflowGraph().AddVertex(tt.res)
			assert.NoError(err)
			eval := NewEvaluator(testSol)
			actual, err := eval.resourceVertices(tt.res, tt.tmpl)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)
			assert.Equal(tt.want, actual)
		})
	}
}

func TestEvaluator_RemoveEdge(t *testing.T) {
	tests := []struct {
		name         string
		initialState []Vertex
		source       construct.ResourceId
		target       construct.ResourceId
		want         map[Key]Vertex
		wantErr      bool
	}{
		{
			name: "remove edge",
			initialState: []Vertex{
				&edgeVertex{
					Edge: construct.SimpleEdge{
						Source: graphtest.ParseId(t, "a:a:a"),
						Target: graphtest.ParseId(t, "a:a:b"),
					},
				},
			},
			source: graphtest.ParseId(t, "a:a:a"),
			target: graphtest.ParseId(t, "a:a:b"),
			want:   map[Key]Vertex{},
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
			err := eval.RemoveEdge(tt.source, tt.target)
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
		})
	}
}

func TestEvaluator_RemoveResource(t *testing.T) {
	tests := []struct {
		name         string
		initialState []Vertex
		id           construct.ResourceId
		want         map[Key]Vertex
		wantErr      bool
	}{
		{
			name: "remove resource property vertex",
			initialState: []Vertex{
				&propertyVertex{
					Ref: construct.PropertyRef{
						Resource: graphtest.ParseId(t, "a:a:a"),
						Property: "prop1",
					},
				},
			},
			id:   graphtest.ParseId(t, "a:a:a"),
			want: map[Key]Vertex{},
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
			err := eval.RemoveResource(tt.id)
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
		})
	}
}
