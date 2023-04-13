package aws

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/stretchr/testify/assert"
)

func Test_GenerateOrmResources(t *testing.T) {
	appName := "test-app"
	orm := &core.Orm{AnnotationKey: core.AnnotationKey{ID: "test"}}
	cases := []struct {
		name         string
		proxyEnabled bool
		want         coretesting.ResourcesExpectation
		wantMap      map[string][]core.Resource
	}{
		{
			name:         "proxy enabled",
			proxyEnabled: true,
			want: coretesting.ResourcesExpectation{
				Deps: []coretesting.StringDep{
					{Source: "aws:rds_proxy:test-app-test", Destination: "aws:vpc_subnet:test_app_private1"},
					{Source: "aws:rds_proxy:test-app-test", Destination: "aws:vpc_subnet:test_app_private2"},
					{Source: "aws:security_group:test-app", Destination: "aws:vpc:test_app"},
					{Source: "aws:rds_subnet_group:test-app-test", Destination: "aws:vpc_subnet:test_app_private1"},
					{Source: "aws:rds_subnet_group:test-app-test", Destination: "aws:vpc_subnet:test_app_private2"},
				},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			result := core.NewConstructGraph()
			cfg := &config.Application{AppName: appName}
			aws := AWS{Config: cfg}
			err := aws.GenerateOrmResources(orm, result, dag)
			if !assert.NoError(err) {
				return
			}
			for _, dep := range tt.want.Deps {
				assert.NotNilf(dag.GetDependency(dep.Source, dep.Destination), "Did not find dependency for %s -> %s", dep.Source, dep.Destination)
			}
			assert.Len(aws.constructIdToResources[orm.Id()], 2)
		})
	}
}
