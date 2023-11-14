package path_selection

import (
	"strings"
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/construct2/graphtest"
	"github.com/klothoplatform/klotho/pkg/engine2/enginetesting"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func parsePath(t *testing.T, str string) []construct.ResourceId {
	parts := strings.Split(str, "->")
	path := make([]construct.ResourceId, len(parts))
	for i, part := range parts {
		path[i] = graphtest.ParseId(t, strings.TrimSpace(part))
	}
	return path
}

func TestExpandEdge(t *testing.T) {
	tests := []struct {
		name         string
		init         []any
		dep          string
		selectedPath string
		unique       map[string]knowledgebase.Unique
		want         string
		wantErr      bool
	}{
		{
			name:         "path length 2",
			dep:          "p:t:A -> p:t:B",
			selectedPath: "p:t -> p:t",
			want:         "p:t:A -> p:t:B",
		},
		{
			name:         "add new middle",
			dep:          "p:a:A -> p:c:C",
			selectedPath: "p:a -> p:b -> p:c",
			want:         "p:a:A -> p:b:A_C -> p:c:C",
		},
		{
			name:         "reuse middle nonunique",
			init:         []any{"p:a:A", "p:b:B -> p:c:C"},
			dep:          "p:a:A -> p:c:C",
			selectedPath: "p:a -> p:b -> p:c",
			want:         "p:a:A -> p:b:B -> p:c:C",
		},
		{
			name:         "new middle unique src",
			init:         []any{"p:a:A", "p:b:B -> p:c:X"},
			dep:          "p:a:A -> p:c:C",
			selectedPath: "p:a -> p:b -> p:c",
			unique:       map[string]knowledgebase.Unique{"p:a -> p:b": {Source: true}},
			want:         "p:a:A -> p:b:A_C -> p:c:C",
		},
		{
			name:         "reuse middle unique src",
			init:         []any{"p:a:A", "p:b:B -> p:c:C"},
			dep:          "p:a:A -> p:c:C",
			selectedPath: "p:a -> p:b -> p:c",
			unique:       map[string]knowledgebase.Unique{"p:a -> p:b": {Source: true}},
			want:         "p:a:A -> p:b:B -> p:c:C",
		},
		{
			name:         "new middle unique trg",
			init:         []any{"p:a:A", "p:a:X -> p:b:B", "p:c:C"},
			dep:          "p:a:A -> p:c:C",
			selectedPath: "p:a -> p:b -> p:c",
			unique:       map[string]knowledgebase.Unique{"p:a -> p:b": {Target: true}},
			want:         "p:a:A -> p:b:A_C -> p:c:C",
		},
		{
			name:         "reuse middle unique trg",
			init:         []any{"p:a:A", "p:b:B -> p:c:C"},
			dep:          "p:a:A -> p:c:C",
			selectedPath: "p:a -> p:b -> p:c",
			unique:       map[string]knowledgebase.Unique{"p:a -> p:b": {Target: true}},
			want:         "p:a:A -> p:b:B -> p:c:C",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert, require := assert.New(t), require.New(t)

			depIDs := graphtest.ParseEdge(t, tt.dep)

			dep := construct.ResourceEdge{
				Source: construct.CreateResource(depIDs.Source),
				Target: construct.CreateResource(depIDs.Target),
			}

			ctx := enginetesting.NewTestSolution()
			ctx.KB.On("GetResourceTemplate", mock.Anything).Return(nil, nil)
			for edgeStr, unique := range tt.unique {
				edge := graphtest.ParseEdge(t, edgeStr)
				ctx.KB.On("GetEdgeTemplate", edge.Source, edge.Target).Return(&knowledgebase.EdgeTemplate{
					Unique: unique,
				})
			}
			ctx.KB.On("GetEdgeTemplate", mock.Anything, mock.Anything).Return(&knowledgebase.EdgeTemplate{
				Unique: knowledgebase.Unique{},
			})
			ctx.LoadState(t, tt.init...)

			got, err := ExpandEdge(ctx, dep, parsePath(t, tt.selectedPath))
			if tt.wantErr {
				require.Error(err)
				return
			}
			require.NoError(err)

			want := parsePath(t, tt.want)
			assert.Equal(want, got)
		})
	}
}
