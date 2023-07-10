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

func Test_ElasticacheClusterMakeOperational(t *testing.T) {
	cases := []coretesting.MakeOperationalCase[*ElasticacheCluster]{

		{
			Name:     "only cluster",
			Resource: &ElasticacheCluster{Name: "cluster"},
			AppName:  "my-app",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:availability_zones:AvailabilityZones",
					"aws:elastic_ip:my_app_0",
					"aws:elastic_ip:my_app_1",
					"aws:elasticache_cluster:cluster",
					"aws:elasticache_subnetgroup:my-app-cluster-subnetgroup",
					"aws:internet_gateway:my_app_igw",
					"aws:log_group:my-app-cluster-loggroup",
					"aws:nat_gateway:my_app_0",
					"aws:nat_gateway:my_app_1",
					"aws:route_table:my_app_private0",
					"aws:route_table:my_app_private1",
					"aws:route_table:my_app_public",
					"aws:security_group:my_app:my-app",
					"aws:subnet_private:my_app:my_app_private0",
					"aws:subnet_private:my_app:my_app_private1",
					"aws:subnet_public:my_app:my_app_public0",
					"aws:subnet_public:my_app:my_app_public1",
					"aws:vpc:my_app",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:elasticache_cluster:cluster", Destination: "aws:elasticache_subnetgroup:my-app-cluster-subnetgroup"},
					{Source: "aws:elasticache_cluster:cluster", Destination: "aws:log_group:my-app-cluster-loggroup"},
					{Source: "aws:elasticache_cluster:cluster", Destination: "aws:security_group:my_app:my-app"},
					{Source: "aws:elasticache_subnetgroup:my-app-cluster-subnetgroup", Destination: "aws:subnet_private:my_app:my_app_private0"},
					{Source: "aws:elasticache_subnetgroup:my-app-cluster-subnetgroup", Destination: "aws:subnet_private:my_app:my_app_private1"},
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:my_app"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:elastic_ip:my_app_1"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:subnet_public:my_app:my_app_public1"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:elastic_ip:my_app_0"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:subnet_public:my_app:my_app_public0"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:nat_gateway:my_app_0"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:subnet_private:my_app:my_app_private0"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:vpc:my_app"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:nat_gateway:my_app_1"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:subnet_private:my_app:my_app_private1"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:vpc:my_app"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:internet_gateway:my_app_igw"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:subnet_public:my_app:my_app_public0"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:subnet_public:my_app:my_app_public1"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:vpc:my_app"},
					{Source: "aws:security_group:my_app:my-app", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_private:my_app:my_app_private0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_private:my_app:my_app_private0", Destination: "aws:nat_gateway:my_app_0"},
					{Source: "aws:subnet_private:my_app:my_app_private0", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_private:my_app:my_app_private1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_private:my_app:my_app_private1", Destination: "aws:nat_gateway:my_app_1"},
					{Source: "aws:subnet_private:my_app:my_app_private1", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_public:my_app:my_app_public0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:my_app:my_app_public0", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_public:my_app:my_app_public1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:my_app:my_app_public1", Destination: "aws:vpc:my_app"},
				},
			},
			Check: func(assert *assert.Assertions, l *ElasticacheCluster) {
				assert.Equal(l.CloudwatchGroup.Name, "my-app-cluster-loggroup")
				assert.Len(l.SecurityGroups, 1)
				assert.Equal(l.SubnetGroup.Name, "my-app-cluster-subnetgroup")
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
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

func Test_ElasticacheSubnetGroupMakeOperational(t *testing.T) {
	cases := []coretesting.MakeOperationalCase[*ElasticacheSubnetgroup]{

		{
			Name:     "only subnet group",
			Resource: &ElasticacheSubnetgroup{Name: "sg"},
			AppName:  "my-app",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:availability_zones:AvailabilityZones",
					"aws:elastic_ip:my_app_0",
					"aws:elastic_ip:my_app_1",
					"aws:elasticache_subnetgroup:sg",
					"aws:internet_gateway:my_app_igw",
					"aws:nat_gateway:my_app_0",
					"aws:nat_gateway:my_app_1",
					"aws:route_table:my_app_private0",
					"aws:route_table:my_app_private1",
					"aws:route_table:my_app_public",
					"aws:subnet_private:my_app:my_app_private0",
					"aws:subnet_private:my_app:my_app_private1",
					"aws:subnet_public:my_app:my_app_public0",
					"aws:subnet_public:my_app:my_app_public1",
					"aws:vpc:my_app",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:elasticache_subnetgroup:sg", Destination: "aws:subnet_private:my_app:my_app_private0"},
					{Source: "aws:elasticache_subnetgroup:sg", Destination: "aws:subnet_private:my_app:my_app_private1"},
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:my_app"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:elastic_ip:my_app_1"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:subnet_public:my_app:my_app_public1"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:elastic_ip:my_app_0"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:subnet_public:my_app:my_app_public0"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:nat_gateway:my_app_0"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:subnet_private:my_app:my_app_private0"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:vpc:my_app"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:nat_gateway:my_app_1"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:subnet_private:my_app:my_app_private1"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:vpc:my_app"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:internet_gateway:my_app_igw"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:subnet_public:my_app:my_app_public0"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:subnet_public:my_app:my_app_public1"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_private:my_app:my_app_private0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_private:my_app:my_app_private0", Destination: "aws:nat_gateway:my_app_0"},
					{Source: "aws:subnet_private:my_app:my_app_private0", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_private:my_app:my_app_private1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_private:my_app:my_app_private1", Destination: "aws:nat_gateway:my_app_1"},
					{Source: "aws:subnet_private:my_app:my_app_private1", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_public:my_app:my_app_public0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:my_app:my_app_public0", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_public:my_app:my_app_public1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:my_app:my_app_public1", Destination: "aws:vpc:my_app"},
				},
			},
			Check: func(assert *assert.Assertions, l *ElasticacheSubnetgroup) {
				assert.Len(l.Subnets, 2)
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Run(t)
		})
	}
}
