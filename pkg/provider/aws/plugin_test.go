package aws

import (
	"testing"

	dgraph "github.com/dominikbraun/graph"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/klothoplatform/klotho/pkg/graph"
	"github.com/klothoplatform/klotho/pkg/infra/kubernetes"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/stretchr/testify/assert"
)

func Test_ExpandConstructs(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "test", DockerfilePath: "path"}
	orm := &core.Orm{Name: "test"}
	cases := []struct {
		name       string
		constructs []core.Construct
		config     *config.Application
		want       coretesting.ResourcesExpectation
	}{
		{
			name:       "lambda and rds",
			constructs: []core.Construct{eu, orm},
			config: &config.Application{
				AppName: "my-app",
				Defaults: config.Defaults{
					ExecutionUnit: config.KindDefaults{Type: Lambda},
					PersistOrm:    defaultConfig.PersistOrm,
				},
			},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:availability_zones:AvailabilityZones",
					"aws:ecr_image:my-app-test",
					"aws:ecr_repo:my-app",
					"aws:elastic_ip:my_app_0",
					"aws:elastic_ip:my_app_1",
					"aws:iam_role:my-app-test-ExecutionRole",
					"aws:internet_gateway:my_app_igw",
					"aws:lambda_function:my-app-test",
					"aws:log_group:my-app-test",
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
					{Source: "aws:ecr_image:my-app-test", Destination: "aws:ecr_repo:my-app"},
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:my_app"},
					{Source: "aws:lambda_function:my-app-test", Destination: "aws:ecr_image:my-app-test"},
					{Source: "aws:lambda_function:my-app-test", Destination: "aws:iam_role:my-app-test-ExecutionRole"},
					{Source: "aws:lambda_function:my-app-test", Destination: "aws:log_group:my-app-test"},
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
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			result := core.NewConstructGraph()

			for _, construct := range tt.constructs {
				result.AddConstruct(construct)
			}

			aws := AWS{
				Config: tt.config,
			}
			err := aws.ExpandConstructs(result, dag)

			if !assert.NoError(err) {
				return
			}
			tt.want.Assert(t, dag)
		})
	}
}

func Test_CopyConstructEdgesToDag(t *testing.T) {
	orm := &core.Orm{Name: "test"}
	eu := &core.ExecutionUnit{
		Name:                 "test",
		EnvironmentVariables: core.EnvironmentVariables{core.GenerateOrmConnStringEnvVar(orm)},
	}
	gw := &core.Gateway{Name: "test", Routes: []core.Route{{Path: "my/route", Verb: "get", ExecUnitName: eu.Name}}}
	cases := []struct {
		name                 string
		constructs           []graph.Edge[core.Construct]
		config               *config.Application
		constructResourceMap map[core.ResourceId][]core.Resource
		want                 []*graph.Edge[core.Resource]
	}{
		{
			name: "lambda and rds",
			constructs: []graph.Edge[core.Construct]{
				{Source: eu, Destination: orm},
			},
			config: &config.Application{
				AppName: "my-app",
				ExecutionUnits: map[string]*config.ExecutionUnit{
					"test": {Type: Lambda},
				},
			},
			constructResourceMap: map[core.ResourceId][]core.Resource{
				stubId("execution_unit"): {&resources.LambdaFunction{Name: "lambda"}},
				stubId("orm"):            {&resources.RdsInstance{Name: "rds"}},
			},
			want: []*graph.Edge[core.Resource]{
				{Source: &resources.LambdaFunction{Name: "lambda"}, Destination: &resources.RdsInstance{Name: "rds"}, Properties: dgraph.EdgeProperties{
					Attributes: make(map[string]string),
					Data: knowledgebase.EdgeData{
						AppName:     "my-app",
						Source:      &resources.LambdaFunction{Name: "lambda"},
						Destination: &resources.RdsInstance{Name: "rds"},
						Constraint: knowledgebase.EdgeConstraint{
							NodeMustExist: []core.Resource{&resources.RdsProxy{}},
						},
						EnvironmentVariables: []core.EnvironmentVariable{core.GenerateOrmConnStringEnvVar(orm)},
					},
				}},
			},
		},
		{
			name: "api and helm",
			constructs: []graph.Edge[core.Construct]{
				{Source: gw, Destination: eu},
			},
			config: &config.Application{
				AppName: "my-app",
			},
			constructResourceMap: map[core.ResourceId][]core.Resource{
				stubId("execution_unit"): {&kubernetes.HelmChart{Name: "lambda", Values: map[string]any{
					"tg": core.IaCValue{Resource: &resources.TargetGroup{Name: "tg", ConstructsRef: core.BaseConstructSetOf(eu)}},
				}}},
				stubId("expose"): {&resources.RestApi{Name: "api"}},
			},
			want: []*graph.Edge[core.Resource]{
				{Source: &resources.RestApi{Name: "api"}, Destination: &resources.TargetGroup{Name: "tg", ConstructsRef: core.BaseConstructSetOf(eu)}, Properties: dgraph.EdgeProperties{
					Attributes: make(map[string]string),
					Data: knowledgebase.EdgeData{
						AppName:     "my-app",
						Source:      &resources.RestApi{Name: "api"},
						Destination: &resources.TargetGroup{Name: "tg", ConstructsRef: core.BaseConstructSetOf(eu)},
						Routes:      []core.Route{{Path: "my/route", Verb: "get", ExecUnitName: eu.Name}},
					},
				}},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			result := core.NewConstructGraph()

			for _, dep := range tt.constructs {
				result.AddConstruct(dep.Source)
				result.AddConstruct(dep.Destination)
				result.AddDependency(dep.Source.Id(), dep.Destination.Id())
			}
			for _, rs := range tt.constructResourceMap {
				for _, r := range rs {
					dag.AddResource(r)
				}
			}
			aws := AWS{
				Config:                 tt.config,
				constructIdToResources: tt.constructResourceMap,
			}
			err := aws.CopyConstructEdgesToDag(result, dag)

			if !assert.NoError(err) {
				return
			}
			for _, dep := range tt.want {
				edge := dag.GetDependency(dep.Source.Id(), dep.Destination.Id())
				assert.Equal(edge, dep)
			}
		})
	}
}

func Test_configureResources(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "test"}
	cases := []struct {
		name       string
		config     *config.Application
		constructs []core.Construct
		resources  []core.Resource
		want       []core.Resource
	}{
		{
			name: "lambda and rds",
			config: &config.Application{
				AppName: "my-app",
				ExecutionUnits: map[string]*config.ExecutionUnit{
					"test": {
						InfraParams: config.ConvertToInfraParams(config.ServerlessTypeParams{Timeout: 100, Memory: 200}),
					},
				},
			},
			constructs: []core.Construct{
				&core.ExecutionUnit{
					Name:                 "test",
					EnvironmentVariables: core.EnvironmentVariables{core.NewEnvironmentVariable("env1", nil, "val1")}},
			},
			resources: []core.Resource{
				&resources.LambdaFunction{Name: "lambda", ConstructsRef: core.BaseConstructSetOf(eu)},
				&resources.RdsProxy{Name: "rds"},
			},
			want: []core.Resource{
				&resources.LambdaFunction{Name: "lambda", Timeout: 100, MemorySize: 200, ConstructsRef: core.BaseConstructSetOf(eu), EnvironmentVariables: resources.EnvironmentVariables{"env1": core.IaCValue{Property: "val1"}}},
				&resources.RdsProxy{Name: "rds", EngineFamily: "POSTGRESQL", IdleClientTimeout: 1800}},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			result := core.NewConstructGraph()
			for _, construct := range tt.constructs {
				result.AddConstruct(construct)
			}
			for _, res := range tt.resources {
				dag.AddResource(res)
			}
			aws := AWS{
				Config: tt.config,
			}
			err := aws.configureResources(result, dag)

			if !assert.NoError(err) {
				return
			}
			for _, res := range tt.want {
				graphRes := dag.GetResource(res.Id())
				assert.Equal(graphRes, res)
			}
		})
	}
}

func Test_getS3BucketConfig(t *testing.T) {
	cases := []struct {
		name       string
		constructs []core.Construct
		want       resources.S3BucketConfigureParams
		wantErr    bool
	}{
		{
			name: "no constructs", // Not an expected case, but it should work anyway
			want: getFsConfiguration(),
		},
		{
			name: "fs construct",
			constructs: []core.Construct{
				&core.Fs{Name: "fs-a"},
			},
			want: getFsConfiguration(),
		},
		{
			name: "multiple exec units", // for lambda payloads
			constructs: []core.Construct{
				&core.Fs{Name: "unit-a"},
				&core.Fs{Name: "unit-b"},
			},
			want: getFsConfiguration(),
		},
		{
			name: "single static unit",
			constructs: []core.Construct{
				&core.StaticUnit{
					Name:          "unit-a",
					IndexDocument: "my index doc",
				},
			},
			want: resources.S3BucketConfigureParams{
				ForceDestroy:  true,
				IndexDocument: "my index doc",
			},
		},
		{
			name: "multiple static units",
			constructs: []core.Construct{
				&core.StaticUnit{
					Name:          "unit-a",
					IndexDocument: "my index doc 001",
				},
				&core.StaticUnit{
					Name:          "unit-b",
					IndexDocument: "my index doc 002",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			bucket := &resources.S3Bucket{}
			constructs := core.NewConstructGraph()
			for _, cons := range tt.constructs {
				constructs.AddConstruct(cons)
				bucket.ConstructsRef.Add(cons)
			}

			actualConfig, actualErr := getS3BucketConfig(bucket, constructs)
			if tt.wantErr {
				assert.Error(actualErr)
			} else {
				assert.NoError(actualErr)
				assert.Equal(tt.want, actualConfig)
			}
		})
	}
}

func stubId(cap string) core.ResourceId {
	return core.ResourceId{
		Provider: core.AbstractConstructProvider,
		Type:     cap,
		Name:     "test",
	}
}
