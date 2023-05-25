package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/stretchr/testify/assert"
)

func Test_ElasticacheClusterCreate(t *testing.T) {
	cluster := &core.RedisNode{AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.PersistCapability}}
	existingKey := core.AnnotationKey{ID: "existing", Capability: annotation.PersistCapability}
	cases := []coretesting.CreateCase[ElasticacheClusterCreateParams, *ElasticacheCluster]{
		{
			Name: "nil elasticache cluster",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:availability_zones:AvailabilityZones",
					"aws:elastic_ip:my_app_0",
					"aws:elastic_ip:my_app_1",
					"aws:elasticache:my-app-test",
					"aws:elasticache_subnetgroup:my-app-test-subnetgroup",
					"aws:internet_gateway:my_app_igw",
					"aws:log_group:my-app-test",
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
					{Source: "aws:elasticache:my-app-test", Destination: "aws:elasticache_subnetgroup:my-app-test-subnetgroup"},
					{Source: "aws:elasticache:my-app-test", Destination: "aws:log_group:my-app-test"},
					{Source: "aws:elasticache:my-app-test", Destination: "aws:security_group:my_app:my-app"},
					{Source: "aws:elasticache_subnetgroup:my-app-test-subnetgroup", Destination: "aws:subnet_private:my_app:my_app_private0"},
					{Source: "aws:elasticache_subnetgroup:my-app-test-subnetgroup", Destination: "aws:subnet_private:my_app:my_app_private1"},
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:my_app"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:elastic_ip:my_app_0"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:subnet_public:my_app:my_app_public0"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:elastic_ip:my_app_1"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:subnet_public:my_app:my_app_public1"},
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
					{Source: "aws:subnet_private:my_app:my_app_private0", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_private:my_app:my_app_private1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_private:my_app:my_app_private1", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_public:my_app:my_app_public0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:my_app:my_app_public0", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_public:my_app:my_app_public1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:my_app:my_app_public1", Destination: "aws:vpc:my_app"},
				},
			},
			Check: func(assert *assert.Assertions, table *ElasticacheCluster) {
				assert.Equal(table.Name, "my-app-test")
				assert.Equal(table.ConstructsRef, core.AnnotationKeySetOf(cluster.AnnotationKey))
			},
		},
		{
			Name:     "existing elasticache cluster",
			Existing: []core.Resource{&ElasticacheCluster{Name: "my-app-test", ConstructsRef: core.AnnotationKeySetOf(existingKey)}},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:availability_zones:AvailabilityZones",
					"aws:elastic_ip:my_app_0",
					"aws:elastic_ip:my_app_1",
					"aws:elasticache:my-app-test",
					"aws:elasticache_subnetgroup:my-app-test-subnetgroup",
					"aws:internet_gateway:my_app_igw",
					"aws:log_group:my-app-test",
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
					{Source: "aws:elasticache:my-app-test", Destination: "aws:elasticache_subnetgroup:my-app-test-subnetgroup"},
					{Source: "aws:elasticache:my-app-test", Destination: "aws:log_group:my-app-test"},
					{Source: "aws:elasticache:my-app-test", Destination: "aws:security_group:my_app:my-app"},
					{Source: "aws:elasticache_subnetgroup:my-app-test-subnetgroup", Destination: "aws:subnet_private:my_app:my_app_private0"},
					{Source: "aws:elasticache_subnetgroup:my-app-test-subnetgroup", Destination: "aws:subnet_private:my_app:my_app_private1"},
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:my_app"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:elastic_ip:my_app_0"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:subnet_public:my_app:my_app_public0"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:elastic_ip:my_app_1"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:subnet_public:my_app:my_app_public1"},
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
					{Source: "aws:subnet_private:my_app:my_app_private0", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_private:my_app:my_app_private1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_private:my_app:my_app_private1", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_public:my_app:my_app_public0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:my_app:my_app_public0", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_public:my_app:my_app_public1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:my_app:my_app_public1", Destination: "aws:vpc:my_app"},
				}},
			Check: func(assert *assert.Assertions, table *ElasticacheCluster) {
				assert.Equal(table.Name, "my-app-test")
				assert.Equal(table.ConstructsRef, core.AnnotationKeySetOf(cluster.AnnotationKey, existingKey))
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = ElasticacheClusterCreateParams{
				AppName: "my-app",
				Refs:    core.AnnotationKeySetOf(cluster.AnnotationKey),
				Name:    "test",
			}

			tt.Run(t)
		})
	}
}
