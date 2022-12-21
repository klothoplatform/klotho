package core

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
