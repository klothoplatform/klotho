package kbtesting

import (
	"fmt"
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/construct/graphtest"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledgebase"
	"github.com/klothoplatform/klotho/pkg/knowledgebase/properties"
	"github.com/stretchr/testify/assert"
)

func TestDownstream(t *testing.T) {
	// The following variables are provided as a convenience for writing tests that all can use the same
	// basic structure / parameters
	defaultGraph := graphtest.MakeGraph(t, construct.NewGraph(),
		"p:A:A -> p:B:B -> p:C:C", // A -> B -> C all within the same operational boundary
		"p:B:B -> p:D:D",          // B -> D crosses operational boundaries
		"p:B:B -> p:X:X -> p:Y:Y", // X is a functional boundary
	)
	defaultResource := graphtest.ParseId(t, "p:A:A")

	makeOpProperty := func(name string, namespace bool) knowledgebase.Property {
		p := &properties.ResourceProperty{}
		p.Name = name
		p.Path = name
		p.Namespace = namespace
		p.OperationalRule = &knowledgebase.PropertyRule{
			Step: knowledgebase.OperationalStep{
				Resources: []knowledgebase.ResourceSelector{{Selector: "p:" + name + ":" + name}},
			},
		}
		return p
	}

	a := &knowledgebase.ResourceTemplate{
		QualifiedTypeName: "p:A",
		Classification:    knowledgebase.Classification{Is: []string{"compute"}},
		Properties: knowledgebase.Properties{
			"B": makeOpProperty("B", false),
		},
	}
	b := &knowledgebase.ResourceTemplate{
		QualifiedTypeName: "p:B",
		Properties: knowledgebase.Properties{
			"C": makeOpProperty("C", false),
			"D": &properties.ResourceProperty{}, // not an operational property
		},
	}
	x := &knowledgebase.ResourceTemplate{
		QualifiedTypeName: "p:X",
		Classification:    knowledgebase.Classification{Is: []string{"compute"}},
	}

	kb := MakeKB(t, a, b, "p:C", "p:D", x, "p:Y")

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
	setProp("p:A:A", "B", "p:B:B")
	setProp("p:B:B", "C", "p:C:C")
	setProp("p:B:B", "D", "p:D:D")

	tests := []struct {
		name     string
		graph    []any
		resource string
		layer    knowledgebase.DependencyLayer
		want     []string
		wantErr  bool
	}{
		// Local
		{
			name:  "local",
			layer: knowledgebase.ResourceLocalLayer,
			want:  []string{"p:B:B", "p:C:C"},
		},

		// Direct
		{
			name:     "no downstream",
			resource: "p:C:C",
			layer:    knowledgebase.ResourceDirectLayer,
		},
		{
			name:  "direct downstream",
			layer: knowledgebase.ResourceDirectLayer,
			want:  []string{"p:B:B"},
		},
		{
			name:     "direct downstream B",
			layer:    knowledgebase.ResourceDirectLayer,
			resource: "p:B:B",
			want:     []string{"p:C:C", "p:D:D", "p:X:X"},
		},
		// Glue
		{
			name:  "glue",
			layer: knowledgebase.ResourceGlueLayer,
			want:  []string{"p:B:B", "p:C:C", "p:D:D"},
		},
		// Functional
		{
			name:  "functional",
			layer: knowledgebase.FirstFunctionalLayer,
			want:  []string{"p:B:B", "p:C:C", "p:D:D", "p:X:X"},
		},
		// All
		{
			name:  "all",
			layer: knowledgebase.AllDepsLayer,
			want:  []string{"p:B:B", "p:C:C", "p:D:D", "p:X:X", "p:Y:Y"},
		},
		{
			name: "all downstream, simple",
			graph: []any{
				"p:A:a", "p:A:b", "p:A:c", "p:B:a", "p:B:b", "p:B:c",
				"p:A:a -> p:A:b", "p:A:b -> p:A:c",
			},
			resource: "p:A:a",
			layer:    knowledgebase.AllDepsLayer,
			want: []string{
				"p:A:b", "p:A:c",
			},
		},
		{
			name: "all downstream, multiple paths for same resources",
			graph: []any{
				"p:A:a", "p:A:b", "p:A:c", "p:B:a", "p:B:b", "p:B:c",
				"p:A:a -> p:A:b", "p:A:b -> p:A:c",
				"p:A:b -> p:B:b", "p:B:b -> p:B:c",
				"p:A:c -> p:B:b", "p:B:b -> p:B:c",
			},
			resource: "p:A:a",
			layer:    knowledgebase.AllDepsLayer,
			want: []string{
				"p:A:b", "p:A:c", "p:B:b", "p:B:c",
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
