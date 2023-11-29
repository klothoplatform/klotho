package operational_eval

import (
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/construct2/graphtest"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/klothoplatform/klotho/pkg/set"
	"github.com/stretchr/testify/assert"
)

type expectedChanges struct {
	nodes []Key
	edges map[Key]set.Set[Key]
}

func (expected expectedChanges) assert(t *testing.T, actual graphChanges) {
	assert := assert.New(t)

	actualNodes := make([]Key, 0, len(actual.nodes))
	for node := range actual.nodes {
		actualNodes = append(actualNodes, node)
	}
	assert.ElementsMatch(expected.nodes, actualNodes)

	for from, tos := range expected.edges {
		actualEs := actual.edges[from]
		if !assert.NotNil(actualEs, "missing edges from %s", from) {
			continue
		}
		assert.ElementsMatch(tos.ToSlice(), actualEs.ToSlice(), "edges from %s", from)
	}
}

func Test_fauxConfigContext_ExecuteDecode(t *testing.T) {
	testGraph := graphtest.MakeGraph(t, construct.NewGraph(),
		"mock:resource1:A",
		"mock:resource1:B",
		"mock:resource1:C",
		"mock:resource1:A -> mock:resource1:B",
		"mock:resource1:B -> mock:resource1:C",
	)
	cfgCtx := knowledgebase.DynamicValueContext{Graph: testGraph}
	ref := construct.PropertyRef{
		Resource: construct.ResourceId{Provider: "mock", Type: "resource1", Name: "B"},
		Property: "Res4",
	}
	data := knowledgebase.DynamicValueData{
		Resource: ref.Resource,
	}
	srcKey := Key{Ref: ref}

	tests := []struct {
		name    string
		tmpl    string
		want    expectedChanges
		wantErr bool
	}{
		{
			name: "no deps",
			tmpl: "1",
			want: expectedChanges{},
		},
		{
			name: "simple dep",
			tmpl: `{{ fieldValue "Res4" "mock:resource1:C" }}`,
			want: expectedChanges{
				edges: map[Key]set.Set[Key]{
					srcKey: set.SetOf(Key{Ref: graphtest.ParseRef(t, "mock:resource1:C#Res4")}),
				},
			},
		},
		{
			name: "dep from dynamic res",
			tmpl: `{{ downstream "mock:resource1" .Self | fieldValue "Res4" }}`,
			want: expectedChanges{
				edges: map[Key]set.Set[Key]{
					srcKey: set.SetOf(Key{Ref: graphtest.ParseRef(t, "mock:resource1:C#Res4")}),
				},
			},
		},
		{
			name: "multiple deps",
			tmpl: `{{ $c := downstream "mock:" .Self }}
			{{ $a := upstream "mock:" .Self }}
			mock:{{ fieldValue "Res4" $c }}:{{ fieldValue "Res2s" $a }}:{{ fieldValue "Name" $a }}`,
			want: expectedChanges{
				edges: map[Key]set.Set[Key]{
					srcKey: set.SetOf(Key{Ref: graphtest.ParseRef(t, "mock:resource1:A#Res2s")}),
					srcKey: set.SetOf(Key{Ref: graphtest.ParseRef(t, "mock:resource1:A#Name")}),
					srcKey: set.SetOf(Key{Ref: graphtest.ParseRef(t, "mock:resource1:C#Res4")}),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			ctx := &fauxConfigContext{
				propRef: ref,
				inner:   cfgCtx,
				changes: newChanges(),
				src:     Key{Ref: ref},
			}
			var v interface{}
			err := ctx.ExecuteDecode(tt.tmpl, data, &v)
			if tt.wantErr {
				assert.Error(err)
				return
			} else if !assert.NoError(err) {
				return
			}
			tt.want.assert(t, ctx.changes)
		})
	}
}
