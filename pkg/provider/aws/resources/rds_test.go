package resources

import (
	"fmt"
	"testing"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/stretchr/testify/assert"
)

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
