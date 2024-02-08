package iac3

import (
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/construct/graphtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVariablesFromGraph(t *testing.T) {
	makegraph := func(args ...any) construct.Graph {
		return graphtest.MakeGraph(t, construct.NewGraph(), args...)
	}
	id := func(id string) (rid construct.ResourceId) {
		err := rid.UnmarshalText([]byte(id))
		if err != nil {
			t.Fatal(err)
		}
		return
	}
	tests := []struct {
		name    string
		graph   construct.Graph
		want    variables
		wantErr bool
	}{
		{
			name: "simple",
			graph: makegraph(
				"prov:type_a:res_a",
				"prov:type_b:res_b",
			),
			want: variables{
				id("prov:type_a:res_a"): "res_a",
				id("prov:type_b:res_b"): "res_b",
			},
		},
		{
			name: "same name, different type",
			graph: makegraph(
				"prov:type_a:myres",
				"prov:type_b:myres",
			),
			want: variables{
				id("prov:type_a:myres"): "type_a_myres",
				id("prov:type_b:myres"): "type_b_myres",
			},
		},
		{
			name: "same name, different namespace",
			graph: makegraph(
				"prov:type_c:ns1:myres",
				"prov:type_c:ns2:myres",
			),
			want: variables{
				id("prov:type_c:ns1:myres"): "ns1_myres",
				id("prov:type_c:ns2:myres"): "ns2_myres",
			},
		},
		{
			name: "same name, same namespace, different type",
			graph: makegraph(
				"prov:type_d:ns3:myres",
				"prov:type_e:ns3:myres",
			),
			want: variables{
				id("prov:type_d:ns3:myres"): "type_d_myres",
				id("prov:type_e:ns3:myres"): "type_e_myres",
			},
		},
		{
			name: "same name and type",
			graph: makegraph(
				"prov:type_a:ns1:myres",
				"prov:type_a:ns2:myres",
				"prov:type_b:myres",
			),
			want: variables{
				id("prov:type_a:ns1:myres"): "type_a_ns1_myres",
				id("prov:type_a:ns2:myres"): "type_a_ns2_myres",
				id("prov:type_b:myres"):     "type_b_myres",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert, require := assert.New(t), require.New(t)

			got, err := VariablesFromGraph(tt.graph)
			if tt.wantErr {
				require.Error(err)
				return
			}
			require.NoError(err)
			assert.Equal(tt.want, got)
		})
	}
}
