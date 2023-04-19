package config

type (
	// Config is how config Klotho constructs and other related resources (or meta-resources) are represented in the klotho configuration
	Config struct {
		Type        string      `json:"type" yaml:"type" toml:"type"`
		Path        string      `json:"path,omitempty" yaml:"path,omitempty" toml:"path,omitempty"`
		InfraParams InfraParams `json:"infra_params,omitempty" yaml:"infra_params,omitempty" toml:"infra_params,omitempty"`
	}
)

// GetConfig returns the `Config` config for the resource specified by `id`
// merged with the defaults.
func (a Application) GetConfig(id string) Config {
	cfg := Config{
		Type: a.Defaults.Config.Type,
	}
	ecfg, hasOverride := a.Config[id]
	if hasOverride {
		overrideValue(&cfg.Type, ecfg.Type)
		cfg.InfraParams = ecfg.InfraParams
	}
	cfg.InfraParams.ApplyDefaults(a.Defaults.Config.InfraParamsByType[cfg.Type])

	return cfg
}
