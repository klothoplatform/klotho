package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/klothoplatform/klotho/pkg/construct/coretesting"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/stretchr/testify/assert"
)

func Test_ElasticacheClusterCreate(t *testing.T) {
	eu := &types.ExecutionUnit{Name: "test"}
	eu2 := &types.ExecutionUnit{Name: "first"}
	initialRefs := construct.BaseConstructSetOf(eu2)
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
				assert.Equal(ec.ConstructRefs, construct.BaseConstructSetOf(eu))
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
				assert.Equal(ec.ConstructRefs, construct.BaseConstructSetOf(eu, eu2))
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = ElasticacheClusterCreateParams{
				Refs:    construct.BaseConstructSetOf(eu),
				AppName: "my-app",
				Name:    "ec",
			}
			tt.Run(t)
		})
	}
}

func Test_ElasticacheSubnetGroupCreate(t *testing.T) {
	eu := &types.ExecutionUnit{Name: "test"}
	eu2 := &types.ExecutionUnit{Name: "first"}
	initialRefs := construct.BaseConstructSetOf(eu2)
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
				assert.Equal(ec.ConstructRefs, construct.BaseConstructSetOf(eu))
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
				assert.Equal(ec.ConstructRefs, construct.BaseConstructSetOf(eu, eu2))
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = ElasticacheSubnetgroupCreateParams{
				Refs:    construct.BaseConstructSetOf(eu),
				AppName: "my-app",
				Name:    "ec",
			}
			tt.Run(t)
		})
	}
}
