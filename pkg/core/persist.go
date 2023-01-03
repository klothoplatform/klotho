package core

import (
	"fmt"
	"strings"
)

type (
	Persist struct {
		Name        string
		PersistType string
		Kind        PersistKind
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

func (p *Persist) Type() string { return p.PersistType }

func (p *Persist) Key() ResourceKey {
	return ResourceKey{
		Name: p.Name,
		Kind: string(p.Kind),
	}
}

func GenerateRedisHostEnvVar(id string, kind string) EnvironmentVariable {
	return EnvironmentVariable{
		Name:       fmt.Sprintf("%s%s", strings.ToUpper(id), REDIS_HOST_ENV_VAR_NAME_SUFFIX),
		Kind:       kind,
		ResourceID: id,
		Value:      string(HOST),
	}
}

func GenerateRedisPortEnvVar(id string, kind string) EnvironmentVariable {
	return EnvironmentVariable{
		Name:       fmt.Sprintf("%s%s", strings.ToUpper(id), REDIS_PORT_ENV_VAR_NAME_SUFFIX),
		Kind:       kind,
		ResourceID: id,
		Value:      string(PORT),
	}
}

func GenerateOrmConnStringEnvVar(id string, kind string) EnvironmentVariable {
	return EnvironmentVariable{
		Name:       fmt.Sprintf("%s%s", strings.ToUpper(id), ORM_ENV_VAR_NAME_SUFFIX),
		Kind:       kind,
		ResourceID: id,
		Value:      string(CONNECTION_STRING),
	}
}
