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

func (p *Secrets) Provenance() AnnotationKey {
	return p.AnnotationKey
}

func (p *Secrets) Id() string {
	return p.AnnotationKey.ToId()
}

func (p *Fs) Provenance() AnnotationKey {
	return p.AnnotationKey
}

func (p *Fs) Id() string {
	return p.AnnotationKey.ToId()
}

func (p *Kv) Provenance() AnnotationKey {
	return p.AnnotationKey
}

func (p *Kv) Id() string {
	return p.AnnotationKey.ToId()
}

func (p *Orm) Provenance() AnnotationKey {
	return p.AnnotationKey
}

func (p *Orm) Id() string {
	return p.AnnotationKey.ToId()
}

func (p *RedisNode) Provenance() AnnotationKey {
	return p.AnnotationKey
}

func (p *RedisNode) Id() string {
	return p.AnnotationKey.ToId()
}

func (p *RedisCluster) Provenance() AnnotationKey {
	return p.AnnotationKey
}

func (p *RedisCluster) Id() string {
	return p.AnnotationKey.ToId()
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
	return NewEnvironmentVariable("KLOTHO_KV_DYNAMODB_TABLE_NAME", cfg, string(KV_DYNAMODB_TABLE_NAME))
}
