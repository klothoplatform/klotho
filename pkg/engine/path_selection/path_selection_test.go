package path_selection

import (
	"context"
	"testing"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/construct/graphtest"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledgebase"
	"github.com/klothoplatform/klotho/pkg/knowledgebase/kbtesting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildPathSelectionGraph(t *testing.T) {
	class := func(is ...string) knowledgebase.Classification {
		return knowledgebase.Classification{Is: is}
	}
	type args struct {
		dep            string
		kb             *knowledgebase.KnowledgeBase
		classification string
	}
	tests := []struct {
		name        string
		args        args
		want        []any
		wantWeights map[string]int
		wantErr     bool
	}{
		{
			name: "no edge",
			args: args{
				dep: "p:t:a -> p:t:b",
				kb: kbtesting.MakeKB(t,
					&knowledgebase.ResourceTemplate{QualifiedTypeName: "p:t"},
				),
				classification: "network",
			},
			want: []any{"p:t:a", "p:t:b"},
		},
		{
			name: "path through classification",
			args: args{
				dep: "p:a:a -> p:c:c",
				kb: kbtesting.MakeKB(t,
					&knowledgebase.ResourceTemplate{QualifiedTypeName: "p:b", Classification: class("network")},
					"p:a -> p:b -> p:c",
				),
				classification: "network",
			},
			want: []any{"p:a:a -> p:b:phantom$0 -> p:c:c"},
			wantWeights: map[string]int{
				"p:a:a -> p:b:phantom$0": 109,
				"p:b:phantom$0 -> p:c:c": 109,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dep := graphtest.ParseEdge(t, tt.args.dep)

			got, err := BuildPathSelectionGraph(
				context.Background(),
				construct.SimpleEdge{Source: dep.Source, Target: dep.Target},
				tt.args.kb,
				tt.args.classification,
				true,
			)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			want := graphtest.MakeGraph(t, construct.NewGraph(), tt.want...)
			// wantS, _ := construct.String(want)
			// t.Logf("want: %s", wantS)
			for s, ww := range tt.wantWeights {
				e := graphtest.ParseEdge(t, s)
				require.NoError(t, want.UpdateEdge(e.Source, e.Target, graph.EdgeWeight(ww)))
			}
			graphtest.AssertGraphEqual(t, want, got, "")

			assert.True(t, got.Traits().IsWeighted, "not weighted: %+v", got.Traits())
		})
	}
}
