package aws

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/core/coretesting"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/stretchr/testify/assert"
)

func Test_ExpandKv(t *testing.T) {
	kv := &core.Kv{Name: "test"}
	cases := []struct {
		name   string
		kv     *core.Kv
		config *config.Application
		want   coretesting.ResourcesExpectation
	}{
		{
			name:   "single kv",
			kv:     kv,
			config: &config.Application{AppName: "my-app"},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:dynamodb_table:my-app-kv",
				},
				Deps: []coretesting.StringDep{},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()

			aws := AWS{
				Config: tt.config,
			}
			err := aws.expandKv(dag, tt.kv)

			if !assert.NoError(err) {
				return
			}
			tt.want.Assert(t, dag)
			res, _ := aws.GetResourcesDirectlyTiedToConstruct(tt.kv)
			assert.NotEmpty(res)
		})
	}
}
