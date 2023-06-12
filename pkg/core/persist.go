package core

import (
	"fmt"
	"strings"

	"github.com/klothoplatform/klotho/pkg/annotation"
)

type (
	Secrets struct {
		Name    string
		Secrets []string
	}

	Fs struct {
		Name string
	}

	Kv struct {
		Name string
	}

	Orm struct {
		Name string
	}

	RedisNode struct {
		Name string
	}

	RedisCluster struct {
		Name string
	}
)

const (
	KLOTHO_KV_DYNAMODB_TABLE_NAME = "KLOTHO_KV_DYNAMODB_TABLE_NAME"

	SECRETS_TYPE       = "secrets"
	FS_TYPE            = "fs"
	KV_TYPE            = "kv"
	ORM_TYPE           = "orm"
	REDIS_NODE_TYPE    = "redis_node"
	REDIS_CLUSTER_TYPE = "redis_cluster"
)

func (p *Secrets) Id() ResourceId {
	return ResourceId{
		Provider: AbstractConstructProvider,
		Type:     SECRETS_TYPE,
		Name:     p.Name,
	}
}

func (p *Secrets) AnnotationCapability() string {
	return annotation.PersistCapability
}

func (p *Fs) Id() ResourceId {
	return ResourceId{
		Provider: AbstractConstructProvider,
		Type:     FS_TYPE,
		Name:     p.Name,
	}
}

func (p *Fs) AnnotationCapability() string {
	return annotation.PersistCapability
}

func (p *Kv) Id() ResourceId {
	return ResourceId{
		Provider: AbstractConstructProvider,
		Type:     KV_TYPE,
		Name:     p.Name,
	}
}

func (p *Kv) AnnotationCapability() string {
	return annotation.PersistCapability
}

func (p *Orm) Id() ResourceId {
	return ResourceId{
		Provider: AbstractConstructProvider,
		Type:     ORM_TYPE,
		Name:     p.Name,
	}
}

func (p *Orm) AnnotationCapability() string {
	return annotation.PersistCapability
}

func (p *RedisNode) Id() ResourceId {
	return ResourceId{
		Provider: AbstractConstructProvider,
		Type:     REDIS_NODE_TYPE,
		Name:     p.Name,
	}
}

func (p *RedisNode) AnnotationCapability() string {
	return annotation.PersistCapability
}

func (p *RedisCluster) Id() ResourceId {
	return ResourceId{
		Provider: AbstractConstructProvider,
		Type:     REDIS_CLUSTER_TYPE,
		Name:     p.Name,
	}
}

func (p *RedisCluster) AnnotationCapability() string {
	return annotation.PersistCapability
}

func GenerateRedisHostEnvVar(cfg Construct) environmentVariable {
	return NewEnvironmentVariable(fmt.Sprintf("%s%s", strings.ToUpper(cfg.Id().Name), REDIS_HOST_ENV_VAR_NAME_SUFFIX), cfg, string(HOST))
}

func GenerateRedisPortEnvVar(cfg Construct) environmentVariable {
	return NewEnvironmentVariable(fmt.Sprintf("%s%s", strings.ToUpper(cfg.Id().Name), REDIS_PORT_ENV_VAR_NAME_SUFFIX), cfg, string(PORT))
}

func GenerateOrmConnStringEnvVar(cfg Construct) environmentVariable {
	return NewEnvironmentVariable(fmt.Sprintf("%s%s", strings.ToUpper(cfg.Id().Name), ORM_ENV_VAR_NAME_SUFFIX), cfg, string(CONNECTION_STRING))
}

func GenerateBucketEnvVar(cfg Construct) environmentVariable {
	return NewEnvironmentVariable(fmt.Sprintf("%s%s", strings.ToUpper(cfg.Id().Name), BUCKET_NAME_SUFFIX), cfg, string(BUCKET_NAME))
}

func GenerateKvTableNameEnvVar(cfg Construct) environmentVariable {
	return NewEnvironmentVariable(KLOTHO_KV_DYNAMODB_TABLE_NAME, cfg, string(KV_DYNAMODB_TABLE_NAME))
}
