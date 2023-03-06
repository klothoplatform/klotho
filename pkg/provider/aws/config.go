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

type GatewayType string

// Enums for the types we allow in the aws provider so that we can reuse the same string within the provider
const (
	Eks                                = "eks"
	Ecs                                = "ecs"
	Lambda                             = "lambda"
	Rds_postgres                       = "rds_postgres"
	Secrets_manager                    = "secrets_manager"
	S3                                 = "s3"
	Dynamodb                           = "dynamodb"
	Elasticache                        = "elasticache"
	Memorydb                           = "memorydb"
	Sns                                = "sns"
	Cockroachdb_serverless             = "cockroachdb_serverless"
	ApiGateway             GatewayType = "apigateway"
	Alb                    GatewayType = "alb"
)

var defaultConfig = config.Defaults{
	ExecutionUnit: config.KindDefaults{
		Type: Lambda,
		InfraParamsByType: map[string]config.InfraParams{
			Lambda: {
				"memorySize": 512,
				"timeout":    180,
			},
			Ecs: {
				"memory": 512,
				"cpu":    256,
			},
			Eks: {
				"nodeType": "fargate",
				"replicas": 2,
			},
		},
	},
	StaticUnit: config.KindDefaults{
		Type: S3,
		InfraParamsByType: map[string]config.InfraParams{
			S3: {
				"cloudFrontEnabled": true,
			},
		},
	},
	Expose: config.ExposeDefaults{
		KindDefaults: config.KindDefaults{
			Type: string(ApiGateway),
		},
		ApiType: "REST",
	},
	PubSub: config.KindDefaults{
		Type: Sns,
	},
	Config: config.KindDefaults{
		Type: S3,
	},
	Persist: config.PersistKindDefaults{
		KV: config.KindDefaults{
			Type: Dynamodb,
		},
		FS: config.KindDefaults{
			Type: S3,
		},
		Secret: config.KindDefaults{
			Type: S3,
		},
		ORM: config.KindDefaults{
			Type: Rds_postgres,
			InfraParamsByType: map[string]config.InfraParams{
				Rds_postgres: {
					"instanceClass":     "db.t4g.micro",
					"allocatedStorage":  20,
					"skipFinalSnapshot": true,
					"engineVersion":     "13.7",
				},
				Cockroachdb_serverless: {},
			},
		},
		RedisNode: config.KindDefaults{
			Type: Elasticache,
			InfraParamsByType: map[string]config.InfraParams{
				Elasticache: {
					"nodeType":      "cache.t3.micro",
					"numCacheNodes": 1,
				},
			},
		},
		RedisCluster: config.KindDefaults{
			Type: Memorydb,
			InfraParamsByType: map[string]config.InfraParams{
				Memorydb: {
					"nodeType":            "db.t4g.small",
					"numReplicasPerShard": 1,
					"numShards":           "2",
				},
			},
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
