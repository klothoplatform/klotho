package path_selection

import (
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/construct/graphtest"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledgebase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			satisfied := pathSatisfiesClassification(kb, tt.path, tt.classification)
			assert.Equal(t, tt.want, satisfied)
		})
	}
}
