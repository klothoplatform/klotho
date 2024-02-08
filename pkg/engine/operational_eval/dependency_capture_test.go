package operational_eval

import (
	"sort"
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/construct2/graphtest"
	"github.com/klothoplatform/klotho/pkg/engine/enginetesting"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledgebase"
	"github.com/klothoplatform/klotho/pkg/knowledgebase/properties"
	"github.com/klothoplatform/klotho/pkg/set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type expectedChanges struct {
	nodes []Key
	edges map[Key]set.Set[Key]
}

func keysToStrings(ks []Key) []string {
	ss := make([]string, 0, len(ks))
	for _, k := range ks {
		ss = append(ss, k.String())
	}
	sort.Strings(ss)
	return ss
}

func (expected expectedChanges) assert(t *testing.T, actual graphChanges) {
	assert := assert.New(t)

	actualNodes := make([]Key, 0, len(actual.nodes))
	for node := range actual.nodes {
		actualNodes = append(actualNodes, node)
	}
	assert.Equal(keysToStrings(expected.nodes), keysToStrings(actualNodes), "nodes")

	for from, tos := range expected.edges {
		actualEs := actual.edges[from]
		if !assert.NotNil(actualEs, "missing edges from %s", from) {
			continue
		}
		assert.Equal(keysToStrings(tos.ToSlice()), keysToStrings(actualEs.ToSlice()), "edges from %s", from)
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
	kb := &enginetesting.MockKB{}
	kb.On("GetResourceTemplate", mock.Anything).Return(&knowledgebase.ResourceTemplate{
		Properties: knowledgebase.Properties{
			"Res2s": &properties.StringProperty{},
			"Res4":  &properties.StringProperty{},
			"Name":  &properties.StringProperty{},
		},
	}, nil)
	cfgCtx := knowledgebase.DynamicValueContext{
		Graph:         testGraph,
		KnowledgeBase: kb,
	}
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
				nodes: []Key{
					{GraphState: "Downstream(mock:resource1, mock:resource1:B)"},
				},
				edges: map[Key]set.Set[Key]{
					srcKey: set.SetOf(
						Key{Ref: graphtest.ParseRef(t, "mock:resource1:C#Res4")},
						Key{GraphState: "Downstream(mock:resource1, mock:resource1:B)"},
					),
				},
			},
		},
		{
			name: "multiple deps",
			tmpl: `{{ $c := downstream "mock:" .Self }}
			{{ $a := upstream "mock:" .Self }}
			mock:{{ fieldValue "Res4" $c }}:{{ fieldValue "Res2s" $a }}:{{ fieldValue "Name" $a }}`,
			want: expectedChanges{
				nodes: []Key{
					{GraphState: "Downstream(mock:, mock:resource1:B)"},
					{GraphState: "Upstream(mock:, mock:resource1:B)"},
				},
				edges: map[Key]set.Set[Key]{
					srcKey: set.SetOf(
						Key{Ref: graphtest.ParseRef(t, "mock:resource1:A#Res2s")},
						Key{Ref: graphtest.ParseRef(t, "mock:resource1:A#Name")},
						Key{Ref: graphtest.ParseRef(t, "mock:resource1:C#Res4")},
						Key{GraphState: "Downstream(mock:, mock:resource1:B)"},
						Key{GraphState: "Upstream(mock:, mock:resource1:B)"},
					),
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
