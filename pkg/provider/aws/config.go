package aws

import (
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/infra/kubernetes"
	"github.com/klothoplatform/klotho/pkg/provider"
)

type (
	TemplateConfig struct {
		provider.TemplateConfig
	}
)

func (a *AWS) Name() string { return "aws" }

// Enums for the types we allow in the aws provider so that we can reuse the same string within the provider
const (
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
	AppRunner              = "app_runner"
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
			Lambda:                    config.ConvertToInfraParams(lambdaDefaults),
			Ecs:                       config.ConvertToInfraParams(ecsDefaults),
			kubernetes.KubernetesType: config.ConvertToInfraParams(eksDefaults),
		},
	},
	StaticUnit: config.KindDefaults{
		Type: S3,
	},
	Expose: config.KindDefaults{
		Type: ApiGateway,
		InfraParamsByType: map[string]config.InfraParams{
			ApiGateway: config.ConvertToInfraParams(config.GatewayTypeParams{
				ApiType: "REST",
			}),
			Alb: config.ConvertToInfraParams(config.LoadBalancerTypeParams{}),
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
func (a *AWS) GetKindTypeMappings(construct core.Construct) ([]string, bool) {
	switch construct.(type) {
	case *core.ExecutionUnit:
		return []string{kubernetes.KubernetesType, Ecs, Lambda}, true
	case *core.Gateway:
		return []string{string(ApiGateway), string(Alb)}, true
	case *core.StaticUnit:
		return []string{S3}, true
	case *core.Fs:
		return []string{S3}, true
	case *core.Kv:
		return []string{Dynamodb}, true
	case *core.Orm:
		return []string{Rds_postgres}, true
	case *core.RedisNode:
		return []string{Elasticache}, true
	case *core.RedisCluster:
		return []string{Memorydb}, true
	case *core.Secrets:
		return []string{S3}, true
	case *core.PubSub:
		return []string{Sns}, true
	case *core.Config:
		return []string{S3, Secrets_manager}, true
	}
	return nil, false
}
