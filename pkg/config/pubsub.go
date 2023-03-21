package config

type (
	// Pubsub is how pubsub Klotho constructs are represented in the klotho configuration
	PubSub struct {
		Type        string      `json:"type" yaml:"type" toml:"type"`
		InfraParams InfraParams `json:"infra_params,omitempty" yaml:"infra_params,omitempty" toml:"infra_params,omitempty"`
	}
)

// GetConfig returns the `Config` config for the resource specified by `id`
// merged with the defaults.
func (a Application) GetPubSub(id string) PubSub {
	cfg := PubSub{}
	if ecfg, ok := a.PubSub[id]; ok {
		defaultParams, ok := a.Defaults.PubSub.InfraParamsByType[ecfg.Type]
		if ok {
			if ecfg.InfraParams == nil {
				ecfg.InfraParams = defaultParams
			} else {
				ecfg.InfraParams = ecfg.InfraParams.Merge(defaultParams)
			}
		}
		return *ecfg
	}
	cfg.Type = a.Defaults.PubSub.Type
	defaultParams, ok := a.Defaults.PubSub.InfraParamsByType[cfg.Type]
	if ok {
		cfg.InfraParams = cfg.InfraParams.Merge(defaultParams)

	}
	return cfg
}
