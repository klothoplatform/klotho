package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/stretchr/testify/assert"
)

func Test_TargetGroupCreate(t *testing.T) {
	initialRefs := []core.AnnotationKey{{ID: "first"}}
	cases := []struct {
		name    string
		tg      *TargetGroup
		want    coretesting.ResourcesExpectation
		wantErr bool
	}{
		{
			name: "nil target group",
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:target_group:app-my-tg",
					"aws:vpc:app",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:target_group:app-my-tg", Destination: "aws:vpc:app"},
				},
			},
		},
		{
			name:    "existing target group",
			tg:      &TargetGroup{Name: "app-my-tg", ConstructsRef: initialRefs},
			wantErr: true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			if tt.tg != nil {
				dag.AddResource(tt.tg)
			}
			metadata := TargetGroupCreateParams{
				AppName:         "app",
				TargetGroupName: "my-tg",
				Refs:            []core.AnnotationKey{{ID: "test", Capability: annotation.ExecutionUnitCapability}},
				Port:            80,
				Protocol:        "HTTP",
				TargetType:      "ip",
			}
			tg := &TargetGroup{}
			err := tg.Create(dag, metadata)

			if tt.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			tt.want.Assert(t, dag)

			graphTg := dag.GetResourceByVertexId(tg.Id().String()).(*TargetGroup)

			assert.Equal(graphTg.Name, "app-my-tg")
			assert.Equal(graphTg.ConstructsRef, metadata.Refs)
			assert.Equal(graphTg.Port, 80)
			assert.Equal(graphTg.Protocol, "HTTP")
			assert.Equal(graphTg.TargetType, "ip")
		})
	}
}
