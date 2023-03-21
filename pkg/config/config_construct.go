package config

type (
	// Config is how config Klotho constructs and other related resources (or meta-resources) are represented in the klotho configuration
	Config struct {
		Type        string      `json:"type" yaml:"type" toml:"type"`
		InfraParams InfraParams `json:"infra_params,omitempty" yaml:"infra_params,omitempty" toml:"infra_params,omitempty"`
	}
)

// GetConfig returns the `Config` config for the resource specified by `id`
// merged with the defaults.
func (a Application) GetConfig(id string) Config {
	cfg := Config{}
	if ecfg, ok := a.Config[id]; ok {
		defaultParams, ok := a.Defaults.Config.InfraParamsByType[ecfg.Type]
		if ok {
			if ecfg.InfraParams == nil {
				ecfg.InfraParams = defaultParams
			} else {
				ecfg.InfraParams = ecfg.InfraParams.Merge(defaultParams)
			}
		}

		return *ecfg
	}
	cfg.Type = a.Defaults.Config.Type
	defaultParams, ok := a.Defaults.Config.InfraParamsByType[cfg.Type]
	if ok {
		cfg.InfraParams = defaultParams
	}
	return cfg
}
