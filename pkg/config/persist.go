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

// GetPersistKv returns the `Persist` config for the persist_kv resource specified by `id`
// merged with the defaults.
func (a Application) GetPersistKv(id string) Persist {
	cfg := Persist{}
	if ecfg, ok := a.PersistKv[id]; ok {
		defaultParams, ok := a.Defaults.PersistKv.InfraParamsByType[ecfg.Type]
		if ok {
			if ecfg.InfraParams == nil {
				ecfg.InfraParams = defaultParams
			} else {
				ecfg.InfraParams = ecfg.InfraParams.Merge(defaultParams)
			}
		}
		return *ecfg
	}
	cfg.Type = a.Defaults.PersistKv.Type
	defaultParams, ok := a.Defaults.PersistKv.InfraParamsByType[cfg.Type]
	if ok {
		cfg.InfraParams = cfg.InfraParams.Merge(defaultParams)

	}
	return cfg
}

// GetPersistFs returns the `Persist` config for the persist_fs resource specified by `id`
// merged with the defaults.
func (a Application) GetPersistFs(id string) Persist {
	cfg := Persist{}
	if ecfg, ok := a.PersistFs[id]; ok {
		defaultParams, ok := a.Defaults.PersistFs.InfraParamsByType[ecfg.Type]
		if ok {
			if ecfg.InfraParams == nil {
				ecfg.InfraParams = defaultParams
			} else {
				ecfg.InfraParams = ecfg.InfraParams.Merge(defaultParams)
			}
		}
		return *ecfg
	}
	cfg.Type = a.Defaults.PersistFs.Type
	defaultParams, ok := a.Defaults.PersistFs.InfraParamsByType[cfg.Type]
	if ok {
		cfg.InfraParams = cfg.InfraParams.Merge(defaultParams)

	}
	return cfg
}

// GetPersistSecrets returns the `Persist` config for the persist_secrets resource specified by `id`
// merged with the defaults.
func (a Application) GetPersistSecrets(id string) Persist {
	cfg := Persist{}
	if ecfg, ok := a.PersistSecrets[id]; ok {
		defaultParams, ok := a.Defaults.PersistSecrets.InfraParamsByType[ecfg.Type]
		if ok {
			if ecfg.InfraParams == nil {
				ecfg.InfraParams = defaultParams
			} else {
				ecfg.InfraParams = ecfg.InfraParams.Merge(defaultParams)
			}
		}
		return *ecfg
	}
	cfg.Type = a.Defaults.PersistSecrets.Type
	defaultParams, ok := a.Defaults.PersistSecrets.InfraParamsByType[cfg.Type]
	if ok {
		cfg.InfraParams = cfg.InfraParams.Merge(defaultParams)

	}
	return cfg
}

// GetPersistOrm returns the `Persist` config for the persist_orm resource specified by `id`
// merged with the defaults.
func (a Application) GetPersistOrm(id string) Persist {
	cfg := Persist{}
	if ecfg, ok := a.PersistOrm[id]; ok {
		defaultParams, ok := a.Defaults.PersistOrm.InfraParamsByType[ecfg.Type]
		if ok {
			if ecfg.InfraParams == nil {
				ecfg.InfraParams = defaultParams
			} else {
				ecfg.InfraParams = ecfg.InfraParams.Merge(defaultParams)
			}
		}
		return *ecfg
	}
	cfg.Type = a.Defaults.PersistOrm.Type
	defaultParams, ok := a.Defaults.PersistOrm.InfraParamsByType[cfg.Type]
	if ok {
		cfg.InfraParams = cfg.InfraParams.Merge(defaultParams)

	}
	return cfg
}

// GetPersistRedisNode returns the `Persist` config for the persist_redis_node resource specified by `id`
// merged with the defaults.
func (a Application) GetPersistRedisNode(id string) Persist {
	cfg := Persist{}
	if ecfg, ok := a.PersistRedisNode[id]; ok {
		defaultParams, ok := a.Defaults.PersistRedisNode.InfraParamsByType[ecfg.Type]
		if ok {
			if ecfg.InfraParams == nil {
				ecfg.InfraParams = defaultParams
			} else {
				ecfg.InfraParams = ecfg.InfraParams.Merge(defaultParams)
			}
		}
		return *ecfg
	}
	cfg.Type = a.Defaults.PersistRedisNode.Type
	defaultParams, ok := a.Defaults.PersistRedisNode.InfraParamsByType[cfg.Type]
	if ok {
		cfg.InfraParams = cfg.InfraParams.Merge(defaultParams)

	}
	return cfg
}

// GetPersistRedisCluster returns the `Persist` config for the persist_redis_cluster resource specified by `id`
// merged with the defaults.
func (a Application) GetPersistRedisCluster(id string) Persist {
	cfg := Persist{}
	if ecfg, ok := a.PersistRedisCluster[id]; ok {
		defaultParams, ok := a.Defaults.PersistRedisCluster.InfraParamsByType[ecfg.Type]
		if ok {
			if ecfg.InfraParams == nil {
				ecfg.InfraParams = defaultParams
			} else {
				ecfg.InfraParams = ecfg.InfraParams.Merge(defaultParams)
			}
		}
		return *ecfg
	}
	cfg.Type = a.Defaults.PersistRedisCluster.Type
	defaultParams, ok := a.Defaults.PersistRedisCluster.InfraParamsByType[cfg.Type]
	if ok {
		cfg.InfraParams = cfg.InfraParams.Merge(defaultParams)

	}
	return cfg
}
