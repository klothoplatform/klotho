package path_selection

import (
	"context"
	"fmt"
	"testing"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/construct/graphtest"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledgebase"
	"github.com/klothoplatform/klotho/pkg/knowledgebase/kbtesting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestBuildPathSelectionGraph(t *testing.T) {
	addRes := func(kb *kbtesting.MockKB, s string, is ...string) {
		r := graphtest.ParseId(t, s)
		kb.On("GetResourceTemplate", mock.MatchedBy(r.Matches)).
			Return(&knowledgebase.ResourceTemplate{
				Classification: knowledgebase.Classification{Is: is},
			}, nil)
	}
	addEdge := func(kb *kbtesting.MockKB, s string) {
		e := graphtest.ParseEdge(t, s)
		kb.On("GetEdgeTemplate", mock.MatchedBy(e.Source.Matches), mock.MatchedBy(e.Target.Matches)).
			Return(&knowledgebase.EdgeTemplate{})
	}
	type args struct {
		dep            string
		kb             func(t *testing.T, kb *kbtesting.MockKB)
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
				kb: func(t *testing.T, kb *kbtesting.MockKB) {
					addRes(kb, "p:t")
					kb.On("AllPaths", mock.Anything, mock.Anything).Return([][]*knowledgebase.ResourceTemplate{}, nil)
				},
				classification: "network",
			},
			want: []any{"p:t:a", "p:t:b"},
		},
		{
			name: "path through classification",
			args: args{
				dep: "p:a:a -> p:c:c",
				kb: func(t *testing.T, kb *kbtesting.MockKB) {
					addRes(kb, "p:a")
					addRes(kb, "p:b", "network")
					addRes(kb, "p:c")
					addEdge(kb, "p:a -> p:b")
					addEdge(kb, "p:b -> p:c")
					kb.On("AllPaths", mock.Anything, mock.Anything).Return([][]*knowledgebase.ResourceTemplate{
						{
							{QualifiedTypeName: "p:a"},
							{QualifiedTypeName: "p:b"},
							{QualifiedTypeName: "p:c"},
						},
					}, nil)
				},
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

			kb := &kbtesting.MockKB{}
			kb.Test(t)
			tt.args.kb(t, kb)
			kb.On("GetEdgeTemplate", mock.Anything, mock.Anything).
				Return((*knowledgebase.EdgeTemplate)(nil))

			got, err := BuildPathSelectionGraph(
				context.Background(),
				construct.SimpleEdge{Source: dep.Source, Target: dep.Target},
				kb,
				tt.args.classification,
				true,
			)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			want := graphtest.MakeGraph(t, construct.NewGraph(), tt.want...)
			wantS, _ := construct.String(want)
			fmt.Println(wantS)
			for s, ww := range tt.wantWeights {
				e := graphtest.ParseEdge(t, s)
				require.NoError(t, want.UpdateEdge(e.Source, e.Target, graph.EdgeWeight(ww)))
			}
			graphtest.AssertGraphEqual(t, want, got, "")

			assert.True(t, got.Traits().IsWeighted, "not weighted: %+v", got.Traits())
		})
	}
}

func TestPathSatisfiesClassification(t *testing.T) {
	tests := []struct {
		name              string
		resourceTemplates []*knowledgebase.ResourceTemplate
		EdgeTemplates     []*knowledgebase.EdgeTemplate
		path              []construct.ResourceId
		classification    string
		want              bool
	}{
		{
			name: "empty classification",
			path: []construct.ResourceId{
				graphtest.ParseId(t, "p:a:a"),
				graphtest.ParseId(t, "p:b:b"),
			},
			resourceTemplates: []*knowledgebase.ResourceTemplate{
				{
					QualifiedTypeName: "p:a",
					Classification:    knowledgebase.Classification{Is: []string{"network"}},
				},
				{
					QualifiedTypeName: "p:b",
				},
			},
			EdgeTemplates: []*knowledgebase.EdgeTemplate{
				{
					Source: graphtest.ParseId(t, "p:a:"),
					Target: graphtest.ParseId(t, "p:b:"),
				},
			},
			classification: "",
			want:           true,
		},
		{
			name: "resource template satisfies classification",
			resourceTemplates: []*knowledgebase.ResourceTemplate{
				{
					QualifiedTypeName: "p:a",
					Classification:    knowledgebase.Classification{Is: []string{"network"}},
				},
				{
					QualifiedTypeName: "p:b",
				},
			},
			EdgeTemplates: []*knowledgebase.EdgeTemplate{
				{
					Source: graphtest.ParseId(t, "p:a:"),
					Target: graphtest.ParseId(t, "p:b:"),
				},
			},
			path: []construct.ResourceId{
				graphtest.ParseId(t, "p:a:a"),
				graphtest.ParseId(t, "p:b:b"),
			},
			classification: "network",
			want:           true,
		},
		{
			name: "resource template does not satisfy classification",
			resourceTemplates: []*knowledgebase.ResourceTemplate{
				{
					QualifiedTypeName: "p:a",
					Classification:    knowledgebase.Classification{Is: []string{"network"}},
				},
				{
					QualifiedTypeName: "p:b",
				},
			},
			EdgeTemplates: []*knowledgebase.EdgeTemplate{
				{
					Source:         graphtest.ParseId(t, "p:a:"),
					Target:         graphtest.ParseId(t, "p:b:"),
					Classification: []string{"network"},
				},
			},
			path: []construct.ResourceId{
				graphtest.ParseId(t, "p:a:a"),
				graphtest.ParseId(t, "p:b:b"),
			},
			classification: "storage",
			want:           false,
		},
		{
			name: "resource template denies classification",
			resourceTemplates: []*knowledgebase.ResourceTemplate{
				{
					QualifiedTypeName: "p:a",
					Classification:    knowledgebase.Classification{Is: []string{"network"}},
				},
				{
					QualifiedTypeName: "p:b",
					PathSatisfaction: knowledgebase.PathSatisfaction{
						DenyClassifications: []string{"network"},
					},
				},
			},
			EdgeTemplates: []*knowledgebase.EdgeTemplate{
				{
					Source:         graphtest.ParseId(t, "p:a:"),
					Target:         graphtest.ParseId(t, "p:b:"),
					Classification: []string{"network"},
				},
			},
			path: []construct.ResourceId{
				graphtest.ParseId(t, "p:a:a"),
				graphtest.ParseId(t, "p:b:b"),
			},
			classification: "network",
			want:           false,
		},
		{
			name: "edge template satisfies classification",
			resourceTemplates: []*knowledgebase.ResourceTemplate{
				{
					QualifiedTypeName: "p:a",
				},
				{
					QualifiedTypeName: "p:b",
				},
			},
			EdgeTemplates: []*knowledgebase.EdgeTemplate{
				{
					Source:         graphtest.ParseId(t, "p:a:"),
					Target:         graphtest.ParseId(t, "p:b:"),
					Classification: []string{"network"},
				},
			},
			path: []construct.ResourceId{
				graphtest.ParseId(t, "p:a:a"),
				graphtest.ParseId(t, "p:b:b"),
			},
			classification: "network",
			want:           true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kb := knowledgebase.NewKB()
			for _, rt := range tt.resourceTemplates {
				err := kb.AddResourceTemplate(rt)
				require.NoError(t, err)
			}
			for _, et := range tt.EdgeTemplates {
				err := kb.AddEdgeTemplate(et)
				require.NoError(t, err)
			}
			satisfied := PathSatisfiesClassification(kb, tt.path, tt.classification)
			assert.Equal(t, tt.want, satisfied)
		})
	}
}
