package core

import (
	"fmt"
	"strings"
)

type (
	Secrets struct {
		AnnotationKey
		Secrets []string
	}

	Fs struct {
		AnnotationKey
	}

	Kv struct {
		AnnotationKey
	}

	Orm struct {
		AnnotationKey
	}

	RedisNode struct {
		AnnotationKey
	}

	RedisCluster struct {
		AnnotationKey
	}
)

const (
	KLOTHO_KV_DYNAMODB_TABLE_NAME = "KLOTHO_KV_DYNAMODB_TABLE_NAME"
)

func (p *Secrets) Provenance() AnnotationKey {
	return p.AnnotationKey
}

func (p *Secrets) RId() ResourceId {
	return ConstructId(p.AnnotationKey).ToRid()
}

func (p *Fs) Provenance() AnnotationKey {
	return p.AnnotationKey
}

func (p *Fs) RId() ResourceId {
	return ConstructId(p.AnnotationKey).ToRid()
}

func (p *Kv) Provenance() AnnotationKey {
	return p.AnnotationKey
}

func (p *Kv) RId() ResourceId {
	return ConstructId(p.AnnotationKey).ToRid()
}

func (p *Orm) Provenance() AnnotationKey {
	return p.AnnotationKey
}

func (p *Orm) RId() ResourceId {
	return ConstructId(p.AnnotationKey).ToRid()
}

func (p *RedisNode) Provenance() AnnotationKey {
	return p.AnnotationKey
}

func (p *RedisNode) RId() ResourceId {
	return ConstructId(p.AnnotationKey).ToRid()
}

func (p *RedisCluster) Provenance() AnnotationKey {
	return p.AnnotationKey
}

func (p *RedisCluster) RId() ResourceId {
	return ConstructId(p.AnnotationKey).ToRid()
}

func GenerateRedisHostEnvVar(cfg Construct) environmentVariable {
	return NewEnvironmentVariable(fmt.Sprintf("%s%s", strings.ToUpper(cfg.Provenance().ID), REDIS_HOST_ENV_VAR_NAME_SUFFIX), cfg, string(HOST))
}

func GenerateRedisPortEnvVar(cfg Construct) environmentVariable {
	return NewEnvironmentVariable(fmt.Sprintf("%s%s", strings.ToUpper(cfg.Provenance().ID), REDIS_PORT_ENV_VAR_NAME_SUFFIX), cfg, string(PORT))
}

func GenerateOrmConnStringEnvVar(cfg Construct) environmentVariable {
	return NewEnvironmentVariable(fmt.Sprintf("%s%s", strings.ToUpper(cfg.Provenance().ID), ORM_ENV_VAR_NAME_SUFFIX), cfg, string(CONNECTION_STRING))
}

func GenerateBucketEnvVar(cfg Construct) environmentVariable {
	return NewEnvironmentVariable(fmt.Sprintf("%s%s", strings.ToUpper(cfg.Provenance().ID), BUCKET_NAME_SUFFIX), cfg, string(BUCKET_NAME))
}

func GenerateKvTableNameEnvVar(cfg Construct) environmentVariable {
	return NewEnvironmentVariable(KLOTHO_KV_DYNAMODB_TABLE_NAME, cfg, string(KV_DYNAMODB_TABLE_NAME))
}
