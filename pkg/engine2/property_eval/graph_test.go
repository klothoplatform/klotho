package property_eval

import (
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/construct2/graphtest"
	"github.com/klothoplatform/klotho/pkg/engine2/constraints"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_depsForProp(t *testing.T) {
	testGraph := graphtest.MakeGraph(t, construct.NewGraph(),
		"mock:resource1:A",
		"mock:resource1:B",
		"mock:resource1:C",
		"mock:resource1:A -> mock:resource1:B",
		"mock:resource1:B -> mock:resource1:C",
	)
	cfgCtx := knowledgebase.DynamicValueContext{DAG: testGraph}
	ref := construct.PropertyRef{
		Resource: construct.ResourceId{Provider: "mock", Type: "resource1", Name: "B"},
		Property: "Res4",
	}

	tests := []struct {
		name       string
		constraint *constraints.ResourceConstraint
		value      string
		want       []string
		wantErr    bool
	}{
		{
			name:  "no deps",
			value: "1",
			want:  []string{},
		},
		{
			name:  "simple dep",
			value: `{{ fieldValue "Res4" "mock:resource1:C" }}`,
			want:  []string{"mock:resource1:C#Res4"},
		},
		{
			name:  "dep from dynamic res",
			value: `{{ downstream "mock:resource1" .Self | fieldValue "Res4" }}`,
			want:  []string{"mock:resource1:C#Res4"},
		},
		{
			name: "multiple deps",
			value: `{{ $c := downstream "mock:" .Self }}
			{{ $a := upstream "mock:" .Self }}
			mock:{{ fieldValue "Res4" $c }}:{{ fieldValue "Res2s" $a }}:{{ fieldValue "Name" $a }}`,
			want: []string{
				"mock:resource1:A#Res2s",
				"mock:resource1:A#Name",
				"mock:resource1:C#Res4",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert, require := assert.New(t), require.New(t)

			res := &construct.Resource{
				ID:         ref.Resource,
				Properties: make(construct.Properties),
			}
			path, err := res.PropertyPath(ref.Property)
			require.NoError(err)

			v := &PropertyVertex{
				Ref:  ref,
				Path: path,
				Template: knowledgebase.Property{
					Path:         ref.Property,
					DefaultValue: tt.value,
					// rest of fields don't matter for this test
				},
				Constraint: tt.constraint,
			}

			deps, err := depsForProp(cfgCtx, v)
			if tt.wantErr {
				require.Error(err)
				return
			}
			require.NoError(err)

			got := make([]string, len(deps))
			for i, dep := range deps {
				got[i] = dep.String()
			}

			assert.ElementsMatch(tt.want, got)
		})
	}
}
