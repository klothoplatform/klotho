package types

import (
	"fmt"
	"strings"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/construct"
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

func (p *Secrets) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: construct.AbstractConstructProvider,
		Type:     SECRETS_TYPE,
		Name:     p.Name,
	}
}

func (p *Secrets) AnnotationCapability() string {
	return annotation.PersistCapability
}

func (p *Secrets) Functionality() construct.Functionality {
	return construct.Storage
}

func (p *Secrets) Attributes() map[string]any {
	return map[string]any{
		"secrets": nil,
	}
}

func (p *Fs) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: construct.AbstractConstructProvider,
		Type:     FS_TYPE,
		Name:     p.Name,
	}
}

func (p *Fs) AnnotationCapability() string {
	return annotation.PersistCapability
}

func (p *Fs) Functionality() construct.Functionality {
	return construct.Storage
}

func (p *Fs) Attributes() map[string]any {
	return map[string]any{
		"blob": nil,
	}
}

func (p *Kv) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: construct.AbstractConstructProvider,
		Type:     KV_TYPE,
		Name:     p.Name,
	}
}

func (p *Kv) AnnotationCapability() string {
	return annotation.PersistCapability
}

func (p *Kv) Functionality() construct.Functionality {
	return construct.Storage
}

func (p *Kv) Attributes() map[string]any {
	return map[string]any{
		"kv": nil,
	}
}

func (p *Orm) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: construct.AbstractConstructProvider,
		Type:     ORM_TYPE,
		Name:     p.Name,
	}
}

func (p *Orm) AnnotationCapability() string {
	return annotation.PersistCapability
}

func (p *Orm) Functionality() construct.Functionality {
	return construct.Storage
}

func (p *Orm) Attributes() map[string]any {
	return map[string]any{
		"relational": nil,
	}
}

func (p *RedisNode) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: construct.AbstractConstructProvider,
		Type:     REDIS_NODE_TYPE,
		Name:     p.Name,
	}
}

func (p *RedisNode) AnnotationCapability() string {
	return annotation.PersistCapability
}

func (p *RedisNode) Functionality() construct.Functionality {
	return construct.Storage
}

func (p *RedisNode) Attributes() map[string]any {
	return map[string]any{
		"redis": nil,
	}
}

func (p *RedisCluster) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: construct.AbstractConstructProvider,
		Type:     REDIS_CLUSTER_TYPE,
		Name:     p.Name,
	}
}

func (p *RedisCluster) AnnotationCapability() string {
	return annotation.PersistCapability
}

func (p *RedisCluster) Functionality() construct.Functionality {
	return construct.Storage
}

func (p *RedisCluster) Attributes() map[string]any {
	return map[string]any{
		"redis":   nil,
		"cluster": nil,
	}
}

func GenerateRedisHostEnvVar(cfg construct.Construct) environmentVariable {
	return NewEnvironmentVariable(fmt.Sprintf("%s%s", strings.ToUpper(cfg.Id().Name), REDIS_HOST_ENV_VAR_NAME_SUFFIX), cfg, string(HOST))
}

func GenerateRedisPortEnvVar(cfg construct.Construct) environmentVariable {
	return NewEnvironmentVariable(fmt.Sprintf("%s%s", strings.ToUpper(cfg.Id().Name), REDIS_PORT_ENV_VAR_NAME_SUFFIX), cfg, string(PORT))
}

func GenerateOrmConnStringEnvVar(cfg construct.Construct) environmentVariable {
	return NewEnvironmentVariable(fmt.Sprintf("%s%s", strings.ToUpper(cfg.Id().Name), ORM_ENV_VAR_NAME_SUFFIX), cfg, string(CONNECTION_STRING))
}

func GenerateBucketEnvVar(cfg construct.Construct) environmentVariable {
	return NewEnvironmentVariable(fmt.Sprintf("%s%s", strings.ToUpper(cfg.Id().Name), BUCKET_NAME_SUFFIX), cfg, string(BUCKET_NAME))
}

func GenerateKvTableNameEnvVar(cfg construct.Construct) environmentVariable {
	return NewEnvironmentVariable(KLOTHO_KV_DYNAMODB_TABLE_NAME, cfg, string(KV_DYNAMODB_TABLE_NAME))
}
