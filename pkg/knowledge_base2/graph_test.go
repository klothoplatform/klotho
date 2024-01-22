package knowledgebase2

import (
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/construct2/graphtest"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestDownstream(t *testing.T) {
	type args struct {
		initialState []any
		rid          construct.ResourceId
		layer        DependencyLayer
	}
	tests := []struct {
		name    string
		args    args
		want    []construct.ResourceId
		wantErr bool
	}{
		{
			name: "all downstream, simple",
			args: args{
				initialState: []any{
					"a:a:a", "a:a:b", "a:a:c", "a:b:a", "a:b:b", "a:b:c",
					"a:a:a -> a:a:b", "a:a:b -> a:a:c",
				},
				rid:   construct.ResourceId{Provider: "a", Type: "a", Name: "a"},
				layer: AllDepsLayer,
			},
			want: []construct.ResourceId{
				{Provider: "a", Type: "a", Name: "b"},
				{Provider: "a", Type: "a", Name: "c"},
			},
		},
		{
			name: "all downstream, multiple paths for same resources",
			args: args{
				initialState: []any{
					"a:a:a", "a:a:b", "a:a:c", "a:b:a", "a:b:b", "a:b:c",
					"a:a:a -> a:a:b", "a:a:b -> a:a:c",
					"a:a:b -> a:b:b", "a:b:b -> a:b:c",
					"a:a:c -> a:b:b", "a:b:b -> a:b:c",
				},
				rid:   construct.ResourceId{Provider: "a", Type: "a", Name: "a"},
				layer: AllDepsLayer,
			},
			want: []construct.ResourceId{
				{Provider: "a", Type: "a", Name: "b"},
				{Provider: "a", Type: "a", Name: "c"},
				{Provider: "a", Type: "b", Name: "b"},
				{Provider: "a", Type: "b", Name: "c"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			ctrl := gomock.NewController(t)
			kb := NewMockTemplateKB(ctrl)
			g := graphtest.MakeGraph(t, construct.NewGraph(), tt.args.initialState...)
			got, err := Downstream(g, kb, tt.args.rid, tt.args.layer)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)
			assert.ElementsMatch(tt.want, got)
		})
	}
}
