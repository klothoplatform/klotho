package resources

import (
	"fmt"
	"testing"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/stretchr/testify/assert"
)

func Test_RdsInstanceCreate(t *testing.T) {
	orm := &core.Orm{AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.PersistCapability}}
	cases := []struct {
		name     string
		instance *RdsInstance
		want     coretesting.ResourcesExpectation
		wantErr  bool
	}{
		{
			name: "nil instance",
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:availability_zones:AvailabilityZones",
					"aws:elastic_ip:my_app_0",
					"aws:elastic_ip:my_app_1",
					"aws:internet_gateway:my_app_igw",
					"aws:nat_gateway:my_app_0",
					"aws:nat_gateway:my_app_1",
					"aws:rds_instance:my-app-test",
					"aws:rds_subnet_group:my-app-test",
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
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:my_app"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:elastic_ip:my_app_0"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:subnet_public:my_app:my_app_public0"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:elastic_ip:my_app_1"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:subnet_public:my_app:my_app_public1"},
					{Source: "aws:rds_instance:my-app-test", Destination: "aws:rds_subnet_group:my-app-test"},
					{Source: "aws:rds_instance:my-app-test", Destination: "aws:security_group:my_app:my-app"},
					{Source: "aws:rds_subnet_group:my-app-test", Destination: "aws:subnet_private:my_app:my_app_private0"},
					{Source: "aws:rds_subnet_group:my-app-test", Destination: "aws:subnet_private:my_app:my_app_private1"},
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
		},
		{
			name:     "existing instance",
			instance: &RdsInstance{Name: "my-app-test"},
			wantErr:  true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {

			assert := assert.New(t)
			dag := core.NewResourceGraph()

			if tt.instance != nil {
				dag.AddResource(tt.instance)
			}

			metadata := RdsInstanceCreateParams{
				AppName: "my-app",
				Refs:    []core.AnnotationKey{orm.AnnotationKey},
				Name:    orm.ID,
			}
			instance := &RdsInstance{}
			err := instance.Create(dag, metadata)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			tt.want.Assert(t, dag)

			graphInstance := dag.GetResource(instance.Id())
			instance = graphInstance.(*RdsInstance)

			assert.Equal(instance.Name, "my-app-test")
			assert.ElementsMatch(instance.ConstructsRef, []core.AnnotationKey{orm.AnnotationKey})
		})
	}
}

func Test_RdsSubnetGroupCreate(t *testing.T) {
	orm := &core.Orm{AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.PersistCapability}}
	initialRefs := []core.AnnotationKey{{ID: "first"}}
	cases := []struct {
		name        string
		subnetGroup *RdsSubnetGroup
		want        coretesting.ResourcesExpectation
	}{
		{
			name: "nil instance",
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:availability_zones:AvailabilityZones",
					"aws:elastic_ip:my_app_0",
					"aws:elastic_ip:my_app_1",
					"aws:internet_gateway:my_app_igw",
					"aws:nat_gateway:my_app_0",
					"aws:nat_gateway:my_app_1",
					"aws:rds_subnet_group:my-app-test",
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
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:my_app"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:elastic_ip:my_app_0"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:subnet_public:my_app:my_app_public0"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:elastic_ip:my_app_1"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:subnet_public:my_app:my_app_public1"},
					{Source: "aws:rds_subnet_group:my-app-test", Destination: "aws:subnet_private:my_app:my_app_private0"},
					{Source: "aws:rds_subnet_group:my-app-test", Destination: "aws:subnet_private:my_app:my_app_private1"},
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
					{Source: "aws:subnet_private:my_app:my_app_private0", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_private:my_app:my_app_private1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_private:my_app:my_app_private1", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_public:my_app:my_app_public0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:my_app:my_app_public0", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_public:my_app:my_app_public1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:my_app:my_app_public1", Destination: "aws:vpc:my_app"},
				},
			},
		},
		{
			name:        "existing instance",
			subnetGroup: &RdsSubnetGroup{Name: "my-app-test", ConstructsRef: initialRefs},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:rds_subnet_group:my-app-test",
				},
				Deps: []coretesting.StringDep{},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {

			assert := assert.New(t)
			dag := core.NewResourceGraph()

			if tt.subnetGroup != nil {
				dag.AddResource(tt.subnetGroup)
			}

			metadata := RdsSubnetGroupCreateParams{
				AppName: "my-app",
				Refs:    []core.AnnotationKey{orm.AnnotationKey},
				Name:    orm.ID,
			}
			subnetGroup := &RdsSubnetGroup{}
			err := subnetGroup.Create(dag, metadata)

			if !assert.NoError(err) {
				return
			}
			tt.want.Assert(t, dag)

			graphSG := dag.GetResource(subnetGroup.Id())
			subnetGroup = graphSG.(*RdsSubnetGroup)

			assert.Equal(subnetGroup.Name, "my-app-test")
			if tt.subnetGroup == nil {
				assert.Len(subnetGroup.Subnets, 2)
				assert.Equal(subnetGroup.ConstructsRef, metadata.Refs)
			} else {
				assert.Equal(subnetGroup.KlothoConstructRef(), append(initialRefs, core.AnnotationKey{ID: "test", Capability: annotation.PersistCapability}))
			}
		})
	}
}

func Test_RdsProxyCreate(t *testing.T) {
	orm := &core.Orm{AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.PersistCapability}}
	initialRefs := []core.AnnotationKey{{ID: "first"}}
	cases := []struct {
		name  string
		proxy *RdsProxy
		want  coretesting.ResourcesExpectation
	}{
		{
			name: "nil proxy",
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:availability_zones:AvailabilityZones",
					"aws:elastic_ip:my_app_0",
					"aws:elastic_ip:my_app_1",
					"aws:internet_gateway:my_app_igw",
					"aws:nat_gateway:my_app_0",
					"aws:nat_gateway:my_app_1",
					"aws:rds_proxy:my-app-test",
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
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:my_app"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:elastic_ip:my_app_0"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:subnet_public:my_app:my_app_public0"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:elastic_ip:my_app_1"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:subnet_public:my_app:my_app_public1"},
					{Source: "aws:rds_proxy:my-app-test", Destination: "aws:security_group:my_app:my-app"},
					{Source: "aws:rds_proxy:my-app-test", Destination: "aws:subnet_private:my_app:my_app_private0"},
					{Source: "aws:rds_proxy:my-app-test", Destination: "aws:subnet_private:my_app:my_app_private1"},
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
		},
		{
			name:  "existing instance",
			proxy: &RdsProxy{Name: "my-app-test", ConstructsRef: initialRefs},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:rds_proxy:my-app-test",
				},
				Deps: []coretesting.StringDep{},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {

			assert := assert.New(t)
			dag := core.NewResourceGraph()

			if tt.proxy != nil {
				dag.AddResource(tt.proxy)
			}

			metadata := RdsProxyCreateParams{
				AppName: "my-app",
				Refs:    []core.AnnotationKey{orm.AnnotationKey},
				Name:    orm.ID,
			}
			proxy := &RdsProxy{}
			err := proxy.Create(dag, metadata)

			if !assert.NoError(err) {
				return
			}
			tt.want.Assert(t, dag)

			graphSG := dag.GetResource(proxy.Id())
			proxy = graphSG.(*RdsProxy)

			assert.Equal(proxy.Name, "my-app-test")
			if tt.proxy == nil {
				assert.Len(proxy.Subnets, 2)
				assert.Len(proxy.SecurityGroups, 1)
				assert.Equal(proxy.ConstructsRef, metadata.Refs)
			} else {
				assert.Equal(proxy.KlothoConstructRef(), append(initialRefs, core.AnnotationKey{ID: "test", Capability: annotation.PersistCapability}))
			}
		})
	}
}

func Test_RdsProxyTargetGroupCreate(t *testing.T) {
	orm := &core.Orm{AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.PersistCapability}}
	initialRefs := []core.AnnotationKey{{ID: "first"}}
	cases := []struct {
		name        string
		targetGroup *RdsProxyTargetGroup
		want        coretesting.ResourcesExpectation
	}{
		{
			name: "nil proxy",
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:rds_proxy_target_group:my-app-test",
				},
				Deps: []coretesting.StringDep{},
			},
		},
		{
			name:        "existing instance",
			targetGroup: &RdsProxyTargetGroup{Name: "my-app-test", ConstructsRef: initialRefs},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:rds_proxy_target_group:my-app-test",
				},
				Deps: []coretesting.StringDep{},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {

			assert := assert.New(t)
			dag := core.NewResourceGraph()

			if tt.targetGroup != nil {
				dag.AddResource(tt.targetGroup)
			}

			metadata := RdsProxyTargetGroupCreateParams{
				AppName: "my-app",
				Refs:    []core.AnnotationKey{orm.AnnotationKey},
				Name:    orm.ID,
			}
			targetGroup := &RdsProxyTargetGroup{}
			err := targetGroup.Create(dag, metadata)

			if !assert.NoError(err) {
				return
			}
			tt.want.Assert(t, dag)

			graphSG := dag.GetResource(targetGroup.Id())
			targetGroup = graphSG.(*RdsProxyTargetGroup)

			assert.Equal(targetGroup.Name, "my-app-test")
			if tt.targetGroup == nil {
				assert.Equal(targetGroup.ConstructsRef, metadata.Refs)
			} else {
				assert.Equal(targetGroup.KlothoConstructRef(), append(initialRefs, core.AnnotationKey{ID: "test", Capability: annotation.PersistCapability}))
			}
		})
	}
}

func Test_CreateRdsInstance(t *testing.T) {
	appName := "test-app"
	orm := &core.Orm{AnnotationKey: core.AnnotationKey{ID: "test", Capability: "orm"}}
	subnets := []*Subnet{NewSubnet("subnet", NewVpc(appName), "0", PrivateSubnet, core.IaCValue{})}
	sgs := []*SecurityGroup{{Name: "test"}}
	cases := []struct {
		name         string
		proxyEnabled bool
		want         coretesting.ResourcesExpectation
	}{
		{
			name:         "proxy enabled",
			proxyEnabled: true,
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:iam_policy:test-app-test-ormsecretpolicy",
					"aws:iam_role:test-app-test-ormsecretrole",
					"aws:rds_instance:test-app-test",
					"aws:rds_proxy:test-app-test",
					"aws:rds_proxy_target_group:test-app-test",
					"aws:rds_subnet_group:test-app-test",
					"aws:secret:test-app-orm:test",
					"aws:secret_version:test-app-orm:test",
					"aws:security_group:test",
					"aws:subnet_private:test_app:test_app_subnet",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:iam_policy:test-app-test-ormsecretpolicy", Destination: "aws:secret:test-app-orm:test"},
					{Source: "aws:iam_role:test-app-test-ormsecretrole", Destination: "aws:iam_policy:test-app-test-ormsecretpolicy"},
					{Source: "aws:rds_instance:test-app-test", Destination: "aws:rds_subnet_group:test-app-test"},
					{Source: "aws:rds_instance:test-app-test", Destination: "aws:security_group:test"},
					{Source: "aws:rds_proxy:test-app-test", Destination: "aws:iam_role:test-app-test-ormsecretrole"},
					{Source: "aws:rds_proxy:test-app-test", Destination: "aws:secret:test-app-orm:test"},
					{Source: "aws:rds_proxy:test-app-test", Destination: "aws:security_group:test"},
					{Source: "aws:rds_proxy:test-app-test", Destination: "aws:subnet_private:test_app:test_app_subnet"},
					{Source: "aws:rds_proxy_target_group:test-app-test", Destination: "aws:rds_instance:test-app-test"},
					{Source: "aws:rds_proxy_target_group:test-app-test", Destination: "aws:rds_proxy:test-app-test"},
					{Source: "aws:rds_subnet_group:test-app-test", Destination: "aws:subnet_private:test_app:test_app_subnet"},
					{Source: "aws:secret_version:test-app-orm:test", Destination: "aws:secret:test-app-orm:test"},
				},
			},
		},
		{
			name: "no proxy",
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:rds_instance:test-app-test",
					"aws:rds_subnet_group:test-app-test",
					"aws:security_group:test",
					"aws:subnet_private:test_app:test_app_subnet",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:rds_instance:test-app-test", Destination: "aws:rds_subnet_group:test-app-test"},
					{Source: "aws:rds_instance:test-app-test", Destination: "aws:security_group:test"},
					{Source: "aws:rds_subnet_group:test-app-test", Destination: "aws:subnet_private:test_app:test_app_subnet"},
				},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			cfg := &config.Application{AppName: appName}
			instance, proxy, err := CreateRdsInstance(cfg, orm, tt.proxyEnabled, subnets, sgs, dag)

			if !assert.NoError(err) {
				return
			}
			if !assert.NotNil(instance) {
				return
			}
			if tt.proxyEnabled {
				assert.NotNil(proxy)
				assert.ElementsMatch(proxy.ConstructsRef, []core.AnnotationKey{orm.AnnotationKey})
			}
			assert.ElementsMatch(instance.ConstructsRef, []core.AnnotationKey{orm.AnnotationKey})
			tt.want.Assert(t, dag)
			if tt.proxyEnabled {
				res := dag.GetResource(core.ResourceId{Provider: "aws", Type: "rds_instance", Name: "test-app-test"})
				instance, ok := res.(*RdsInstance)
				if !assert.True(ok) {
					return
				}
				files := instance.GetOutputFiles()
				assert.Len(files, 1)
				f, ok := files[0].(*core.RawFile)
				if !assert.True(ok) {
					return
				}
				assert.Equal(f.Path(), "secrets/"+orm.Id())
				assert.Equal(string(f.Content), fmt.Sprintf("{\n\"username\": \"%s\",\n\"password\": \"%s\"\n}", instance.Username, instance.Password))
			}
		})
	}
}
