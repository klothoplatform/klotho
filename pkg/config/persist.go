package config

type (
	// Persist is how persist Klotho constructs are represented in the klotho configuration
	Persist struct {
		// Type represents the service used for the persist construct
		Type string `json:"type" yaml:"type" toml:"type"`
		// KindParams represents the set of configuration to customize the kind of persist construct represented
		InfraParams InfraParams `json:"infra_params,omitempty" yaml:"infra_params,omitempty" toml:"infra_params,omitempty"`
	}
)

func getPersist(id string, kindDefaults KindDefaults, overrides map[string]*Persist) Persist {
	cfg := Persist{
		Type: kindDefaults.Type,
	}

	ecfg, hasOverride := overrides[id]
	if hasOverride {
		overrideValue(&cfg.Type, ecfg.Type)
		cfg.InfraParams = ecfg.InfraParams
	}
	cfg.InfraParams.ApplyDefaults(kindDefaults.InfraParamsByType[cfg.Type])

	return cfg
}

// GetPersistKv returns the `Persist` config for the persist_kv resource specified by `id`
// merged with the defaults.
func (a Application) GetPersistKv(id string) Persist {
	return getPersist(id, a.Defaults.PersistKv, a.PersistKv)
}

// GetPersistFs returns the `Persist` config for the persist_fs resource specified by `id`
// merged with the defaults.
func (a Application) GetPersistFs(id string) Persist {
	return getPersist(id, a.Defaults.PersistFs, a.PersistFs)
}

// GetPersistSecrets returns the `Persist` config for the persist_secrets resource specified by `id`
// merged with the defaults.
func (a Application) GetPersistSecrets(id string) Persist {
	return getPersist(id, a.Defaults.PersistSecrets, a.PersistSecrets)
}

// GetPersistOrm returns the `Persist` config for the persist_orm resource specified by `id`
// merged with the defaults.
func (a Application) GetPersistOrm(id string) Persist {
	return getPersist(id, a.Defaults.PersistOrm, a.PersistOrm)
}

// GetPersistRedisNode returns the `Persist` config for the persist_redis_node resource specified by `id`
// merged with the defaults.
func (a Application) GetPersistRedisNode(id string) Persist {
	return getPersist(id, a.Defaults.PersistRedisNode, a.PersistRedisNode)
}

// GetPersistRedisCluster returns the `Persist` config for the persist_redis_cluster resource specified by `id`
// merged with the defaults.
func (a Application) GetPersistRedisCluster(id string) Persist {
	return getPersist(id, a.Defaults.PersistRedisCluster, a.PersistRedisCluster)
}
