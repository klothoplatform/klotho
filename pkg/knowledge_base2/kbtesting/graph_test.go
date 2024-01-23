package kbtesting

import (
	"fmt"
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/construct2/graphtest"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/klothoplatform/klotho/pkg/knowledge_base2/properties"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDownstream(t *testing.T) {
	// The following variables are provided as a convenience for writing tests that all can use the same
	// basic structure / parameters
	defaultGraph := graphtest.MakeGraph(t, construct.NewGraph(),
		"p:t:A -> p:t:B -> p:t:C", // A -> B -> C all within the same operational boundary
		"p:t:B -> p:t:D",          // B -> D crosses operational boundaries
		"p:t:B -> p:t:X -> p:t:Y", // X is a functional boundary
	)
	defaultResource := graphtest.ParseId(t, "p:t:A")
	defaultKB := func(t *testing.T, kb *MockKB) {
		named := func(name string) func(construct.ResourceId) bool {
			return func(id construct.ResourceId) bool {
				return id.Name == name
			}
		}
		makeOpProperty := func(name string, namespace bool) knowledgebase.Property {
			p := &properties.ResourceProperty{}
			p.Name = name
			p.Path = name
			p.Namespace = namespace
			p.OperationalRule = &knowledgebase.PropertyRule{
				Step: knowledgebase.OperationalStep{
					Resources: []knowledgebase.ResourceSelector{{Selector: "p:t:" + name}},
				},
			}
			return p
		}

		kb.On("GetResourceTemplate", mock.MatchedBy(named("A"))).Return(&knowledgebase.ResourceTemplate{
			Classification: knowledgebase.Classification{Is: []string{"compute"}},
			Properties: knowledgebase.Properties{
				"B": makeOpProperty("B", false),
			},
		}, nil)
		kb.On("GetResourceTemplate", mock.MatchedBy(named("B"))).Return(&knowledgebase.ResourceTemplate{
			Properties: knowledgebase.Properties{
				"C": makeOpProperty("C", false),
				"D": &properties.ResourceProperty{}, // not an operational property
			},
		}, nil)
		kb.On("GetResourceTemplate", mock.MatchedBy(named("X"))).Return(&knowledgebase.ResourceTemplate{
			Classification: knowledgebase.Classification{Is: []string{"compute"}},
		}, nil)
		kb.On("GetResourceTemplate", mock.Anything).Return(&knowledgebase.ResourceTemplate{}, nil)
	}
	setProp := func(idStr string, prop string, resValueStr string) {
		id := graphtest.ParseId(t, idStr)
		resValue := graphtest.ParseId(t, resValueStr)
		r, err := defaultGraph.Vertex(id)
		if err != nil {
			t.Fatal(fmt.Errorf("error setting property %s.%s = %s: %w", idStr, prop, resValueStr, err))
		}
		err = r.SetProperty(prop, resValue)
		if err != nil {
			t.Fatal(fmt.Errorf("error setting property %s.%s = %s: %w", idStr, prop, resValueStr, err))
		}
	}
	setProp("p:t:A", "B", "p:t:B")
	setProp("p:t:B", "C", "p:t:C")
	setProp("p:t:B", "D", "p:t:D")

	tests := []struct {
		name     string
		graph    []any
		resource string
		layer    knowledgebase.DependencyLayer
		kbmock   func(*testing.T, *MockKB)
		want     []string
		wantErr  bool
	}{
		// Local
		{
			name:  "local",
			layer: knowledgebase.ResourceLocalLayer,
			want:  []string{"p:t:B", "p:t:C"},
		},

		// Direct
		{
			name:     "no downstream",
			resource: "p:t:C",
			layer:    knowledgebase.ResourceDirectLayer,
		},
		{
			name:  "direct downstream",
			layer: knowledgebase.ResourceDirectLayer,
			want:  []string{"p:t:B"},
		},
		{
			name:     "direct downstream B",
			layer:    knowledgebase.ResourceDirectLayer,
			resource: "p:t:B",
			want:     []string{"p:t:C", "p:t:D", "p:t:X"},
		},
		// Glue
		{
			name:  "glue",
			layer: knowledgebase.ResourceGlueLayer,
			want:  []string{"p:t:B", "p:t:C", "p:t:D"},
		},
		// Functional
		{
			name:  "functional",
			layer: knowledgebase.FirstFunctionalLayer,
			want:  []string{"p:t:B", "p:t:C", "p:t:D", "p:t:X"},
		},
		// All
		{
			name:  "all",
			layer: knowledgebase.AllDepsLayer,
			want:  []string{"p:t:B", "p:t:C", "p:t:D", "p:t:X", "p:t:Y"},
		},
		{
			name: "all downstream, simple",
			graph: []any{
				"a:a:a", "a:a:b", "a:a:c", "a:b:a", "a:b:b", "a:b:c",
				"a:a:a -> a:a:b", "a:a:b -> a:a:c",
			},
			resource: "a:a:a",
			layer:    knowledgebase.AllDepsLayer,
			want: []string{
				"a:a:b", "a:a:c",
			},
		},
		{
			name: "all downstream, multiple paths for same resources",
			graph: []any{
				"a:a:a", "a:a:b", "a:a:c", "a:b:a", "a:b:b", "a:b:c",
				"a:a:a -> a:a:b", "a:a:b -> a:a:c",
				"a:a:b -> a:b:b", "a:b:b -> a:b:c",
				"a:a:c -> a:b:b", "a:b:b -> a:b:c",
			},
			resource: "a:a:a",
			layer:    knowledgebase.AllDepsLayer,
			want: []string{
				"a:a:b", "a:a:c", "a:b:b", "a:b:c",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			var rid construct.ResourceId
			if tt.resource != "" {
				rid = graphtest.ParseId(t, tt.resource)
			} else {
				rid = defaultResource
			}

			var g construct.Graph
			if tt.graph != nil {
				g = graphtest.MakeGraph(t, construct.NewGraph(), tt.graph...)
			} else {
				g = defaultGraph
			}

			kb := &MockKB{}
			if tt.kbmock != nil {
				tt.kbmock(t, kb)
			} else {
				defaultKB(t, kb)
			}
			gotIds, err := knowledgebase.Downstream(g, kb, rid, tt.layer)
			if tt.wantErr {
				assert.Error(err)
				return
			} else if !assert.NoError(err) {
				return
			}
			var got []string
			if gotIds != nil {
				got = make([]string, len(gotIds))
				for i, w := range gotIds {
					got[i] = w.String()
				}
			}
			assert.ElementsMatch(tt.want, got)
		})
	}
}
