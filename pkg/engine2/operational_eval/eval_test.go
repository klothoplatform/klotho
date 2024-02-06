package operational_eval

import (
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/enginetesting"
	"github.com/stretchr/testify/assert"
)

func TestEvaluator_cleanupPropertiesSubVertices(t *testing.T) {

	tests := []struct {
		name         string
		initialState []Vertex
		ref          construct.PropertyRef
		resource     *construct.Resource
		want         []Key
		wantErr      bool
	}{
		{
			name: "simple resource",
			initialState: []Vertex{
				&propertyVertex{
					Ref: construct.PropertyRef{
						Resource: construct.ResourceId{Name: "a"},
						Property: "prop1[0]",
					},
				},
				&propertyVertex{
					Ref: construct.PropertyRef{
						Resource: construct.ResourceId{Name: "a"},
						Property: "prop1[0].prop2",
					},
				},
			},
			ref: construct.PropertyRef{
				Resource: construct.ResourceId{Name: "a"},
				Property: "prop1[0]",
			},
			resource: &construct.Resource{
				ID: construct.ResourceId{Name: "a"},
				Properties: map[string]any{
					"prop1": "value1",
				},
			},
			want: []Key{},
		},
		{
			name: "doesnt remove if path parent 2 back exists",
			initialState: []Vertex{
				&propertyVertex{
					Ref: construct.PropertyRef{
						Resource: construct.ResourceId{Name: "a"},
						Property: "prop1[0]",
					},
				},
				&propertyVertex{
					Ref: construct.PropertyRef{
						Resource: construct.ResourceId{Name: "a"},
						Property: "prop1[0].prop2",
					},
				},
			},
			ref: construct.PropertyRef{
				Resource: construct.ResourceId{Name: "a"},
				Property: "prop1[0].prop2",
			},
			resource: &construct.Resource{
				ID: construct.ResourceId{Name: "a"},
				Properties: map[string]any{
					"prop1": "value1",
				},
			},
			want: []Key{
				{Ref: construct.PropertyRef{
					Resource: construct.ResourceId{Name: "a"},
					Property: "prop1[0]",
				}},
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
				err = eval.unevaluated.AddVertex(v)
				assert.NoError(err)
			}
			err := eval.cleanupPropertiesSubVertices(tt.ref, tt.resource)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)
			adj, err := eval.graph.AdjacencyMap()
			assert.NoError(err)
			assert.Equal(len(adj), len(tt.want))
			for key := range adj {
				assert.Contains(tt.want, key)
			}

			// check that the unevaluated graph is also cleaned up
			adj, err = eval.unevaluated.AdjacencyMap()
			assert.NoError(err)
			assert.Equal(len(adj), len(tt.want))
			for key := range adj {
				assert.Contains(tt.want, key)
			}
		})
	}
}
