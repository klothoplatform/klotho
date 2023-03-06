package core

import (
	"fmt"
	"strings"
)

type (
	Persist struct {
		Name string
		Kind PersistKind
	}

	Secrets struct {
		Persist
		Secrets []string
	}
)

type PersistKind string

const (
	PersistKVKind           PersistKind = "persist_kv"
	PersistFileKind         PersistKind = "persist_fs"
	PersistSecretKind       PersistKind = "persist_secret"
	PersistORMKind          PersistKind = "persist_orm"
	PersistRedisNodeKind    PersistKind = "persist_redis_node"
	PersistRedisClusterKind PersistKind = "persist_redis_cluster"
)

func (p *Persist) Key() ResourceKey {
	return ResourceKey{
		Name: p.Name,
		Kind: string(p.Kind),
	}
}

func GenerateRedisHostEnvVar(id string, kind string) environmentVariable {
	return NewEnvironmentVariable(fmt.Sprintf("%s%s", strings.ToUpper(id), REDIS_HOST_ENV_VAR_NAME_SUFFIX), kind, id, string(HOST))
}

func GenerateRedisPortEnvVar(id string, kind string) environmentVariable {
	return NewEnvironmentVariable(fmt.Sprintf("%s%s", strings.ToUpper(id), REDIS_PORT_ENV_VAR_NAME_SUFFIX), kind, id, string(PORT))
}

func GenerateOrmConnStringEnvVar(id string) environmentVariable {
	return NewEnvironmentVariable(fmt.Sprintf("%s%s", strings.ToUpper(id), ORM_ENV_VAR_NAME_SUFFIX), string(PersistORMKind), id, string(CONNECTION_STRING))
}

func GenerateBucketEnvVar(id string) environmentVariable {
	return NewEnvironmentVariable(fmt.Sprintf("%s%s", strings.ToUpper(id), BUCKET_NAME_SUFFIX), string(PersistFileKind), id, string(BUCKET_NAME))
}
