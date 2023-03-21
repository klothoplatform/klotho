package aws

import (
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

type (
	TemplateConfig struct {
		provider.TemplateConfig
		PayloadsBucketName string
	}

	TemplateData struct {
		provider.TemplateData
		TemplateConfig
		UseVPC                  bool
		CloudfrontDistributions []*resources.CloudfrontDistribution
		APIGateways             []provider.Gateway
		ALBs                    []provider.Gateway
		Buckets                 []provider.FS
		SecretManagerSecrets    []provider.Config
		RdsInstances            []provider.ORM
		MemoryDBClusters        []provider.Redis
		ElasticacheInstances    []provider.Redis
	}
)

var AwsTemplateDataKind = "aws_template_data"

func (*TemplateData) Type() string { return "" }

func (t *TemplateData) Key() core.ResourceKey {
	return core.ResourceKey{
		Name: t.AppName,
		Kind: AwsTemplateDataKind,
	}
}

func NewTemplateData(config *config.Application) *TemplateData {
	return &TemplateData{
		TemplateConfig: TemplateConfig{
			TemplateConfig: provider.TemplateConfig{
				AppName: config.AppName,
			},
		},
	}
}

func (c *AWS) Name() string { return "aws" }

// Enums for the types we allow in the aws provider so that we can reuse the same string within the provider
const (
	Eks                    = "eks"
	Ecs                    = "ecs"
	Lambda                 = "lambda"
	Rds_postgres           = "rds_postgres"
	Secrets_manager        = "secrets_manager"
	S3                     = "s3"
	Dynamodb               = "dynamodb"
	Elasticache            = "elasticache"
	Memorydb               = "memorydb"
	Sns                    = "sns"
	Cockroachdb_serverless = "cockroachdb_serverless"
	ApiGateway             = "apigateway"
	Alb                    = "alb"
)

var (
	eksDefaults = config.KubernetesTypeParams{
		NodeType: "fargate",
		Replicas: 2,
	}

	ecsDefaults = config.ContainerTypeParams{
		Memory: 512,
		Cpu:    256,
	}

	lambdaDefaults = config.ServerlessTypeParams{
		Timeout: 180,
		Memory:  512,
	}
)

var defaultConfig = config.Defaults{
	ExecutionUnit: config.KindDefaults{
		Type: Lambda,
		InfraParamsByType: map[string]config.InfraParams{
			Lambda: config.ConvertToInfraParams(lambdaDefaults),
			Ecs:    config.ConvertToInfraParams(ecsDefaults),
			Eks:    config.ConvertToInfraParams(eksDefaults),
		},
	},
	StaticUnit: config.KindDefaults{
		Type: S3,
	},
	Expose: config.KindDefaults{
		Type: ApiGateway,
		InfraParamsByType: map[string]config.InfraParams{
			ApiGateway: config.ConvertToInfraParams(config.GatewayKindParams{
				ApiType: "REST",
			}),
			Alb: config.ConvertToInfraParams(config.LoadBalancerKindParams{}),
		},
	},
	PubSub: config.KindDefaults{
		Type: Sns,
	},
	Config: config.KindDefaults{
		Type: S3,
	},
	PersistKv: config.KindDefaults{
		Type: Dynamodb,
		InfraParamsByType: map[string]config.InfraParams{
			Dynamodb: config.ConvertToInfraParams(config.InfraParams{}),
		},
	},
	PersistFs: config.KindDefaults{
		Type: S3,
		InfraParamsByType: map[string]config.InfraParams{
			S3: config.ConvertToInfraParams(config.InfraParams{}),
		},
	},
	PersistSecrets: config.KindDefaults{
		Type: Secrets_manager,
		InfraParamsByType: map[string]config.InfraParams{
			Secrets_manager: config.ConvertToInfraParams(config.InfraParams{}),
		},
	},
	PersistOrm: config.KindDefaults{
		Type: Rds_postgres,
		InfraParamsByType: map[string]config.InfraParams{
			Rds_postgres: config.ConvertToInfraParams(config.InfraParams{}),
		},
	},
	PersistRedisNode: config.KindDefaults{
		Type: Elasticache,
		InfraParamsByType: map[string]config.InfraParams{
			Elasticache: config.ConvertToInfraParams(config.InfraParams{}),
		},
	},
	PersistRedisCluster: config.KindDefaults{
		Type: Memorydb,
		InfraParamsByType: map[string]config.InfraParams{
			Memorydb: config.ConvertToInfraParams(config.InfraParams{}),
		},
	},
}

func (a *AWS) GetDefaultConfig() config.Defaults {
	return defaultConfig
}

// GetKindTypeMappings returns a list of valid types for the aws provider based on the kind passed in
func (a *AWS) GetKindTypeMappings(kind string) ([]string, bool) {
	switch kind {
	case core.ExecutionUnitKind:
		return []string{Eks, Ecs, Lambda}, true
	case core.GatewayKind:
		return []string{string(ApiGateway), string(Alb)}, true
	case core.StaticUnitKind:
		return []string{S3}, true
	case string(core.PersistFileKind):
		return []string{S3}, true
	case string(core.PersistKVKind):
		return []string{Dynamodb}, true
	case string(core.PersistORMKind):
		return []string{Rds_postgres}, true
	case string(core.PersistRedisNodeKind):
		return []string{Elasticache}, true
	case string(core.PersistRedisClusterKind):
		return []string{Memorydb}, true
	case string(core.PersistSecretKind):
		return []string{S3}, true
	case core.PubSubKind:
		return []string{Sns}, true
	case core.ConfigKind:
		return []string{S3, Secrets_manager}, true
	}
	return nil, false
}
