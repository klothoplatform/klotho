package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/core/coretesting"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/stretchr/testify/assert"
)

func Test_ElasticacheClusterCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "test"}
	eu2 := &core.ExecutionUnit{Name: "first"}
	initialRefs := core.BaseConstructSetOf(eu2)
	cases := []coretesting.CreateCase[ElasticacheClusterCreateParams, *ElasticacheCluster]{
		{
			Name: "nil function",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:elasticache_cluster:my-app-ec",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, ec *ElasticacheCluster) {
				assert.Equal(ec.Name, "my-app-ec")
				assert.Equal(ec.ConstructRefs, core.BaseConstructSetOf(eu))
			},
		},
		{
			Name:     "existing function",
			Existing: &ElasticacheCluster{Name: "my-app-ec", ConstructRefs: initialRefs},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:elasticache_cluster:my-app-ec",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, ec *ElasticacheCluster) {
				assert.Equal(ec.Name, "my-app-ec")
				assert.Equal(ec.ConstructRefs, core.BaseConstructSetOf(eu, eu2))
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = ElasticacheClusterCreateParams{
				Refs:    core.BaseConstructSetOf(eu),
				AppName: "my-app",
				Name:    "ec",
			}
			tt.Run(t)
		})
	}
}

func Test_ElasticacheSubnetGroupCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "test"}
	eu2 := &core.ExecutionUnit{Name: "first"}
	initialRefs := core.BaseConstructSetOf(eu2)
	cases := []coretesting.CreateCase[ElasticacheSubnetgroupCreateParams, *ElasticacheSubnetgroup]{
		{
			Name: "nil function",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:elasticache_subnetgroup:my-app-ec",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, ec *ElasticacheSubnetgroup) {
				assert.Equal(ec.Name, "my-app-ec")
				assert.Equal(ec.ConstructRefs, core.BaseConstructSetOf(eu))
			},
		},
		{
			Name:     "existing function",
			Existing: &ElasticacheSubnetgroup{Name: "my-app-ec", ConstructRefs: initialRefs},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:elasticache_subnetgroup:my-app-ec",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, ec *ElasticacheSubnetgroup) {
				assert.Equal(ec.Name, "my-app-ec")
				assert.Equal(ec.ConstructRefs, core.BaseConstructSetOf(eu, eu2))
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = ElasticacheSubnetgroupCreateParams{
				Refs:    core.BaseConstructSetOf(eu),
				AppName: "my-app",
				Name:    "ec",
			}
			tt.Run(t)
		})
	}
}
