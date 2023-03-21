package config

type (
	// StaticUnit is how static unit Klotho constructs are represented in the klotho configuration
	StaticUnit struct {
		Type                   string                 `json:"type" yaml:"type" toml:"type"`
		InfraParams            InfraParams            `json:"infra_params,omitempty" yaml:"infra_params,omitempty" toml:"infra_params,omitempty"`
		ContentDeliveryNetwork ContentDeliveryNetwork `json:"content_delivery_network,omitempty" yaml:"content_delivery_network,omitempty" toml:"content_delivery_network,omitempty"`
	}
)

// GetConfig returns the `Config` config for the resource specified by `id`
// merged with the defaults.
func (a Application) GetStaticUnit(id string) StaticUnit {
	cfg := StaticUnit{}
	if ecfg, ok := a.StaticUnit[id]; ok {
		defaultParams, ok := a.Defaults.StaticUnit.InfraParamsByType[ecfg.Type]
		if ok {
			if ecfg.InfraParams == nil {
				ecfg.InfraParams = defaultParams
			} else {
				ecfg.InfraParams = ecfg.InfraParams.Merge(defaultParams)
			}
		}
		return *ecfg
	}
	cfg.Type = a.Defaults.StaticUnit.Type
	defaultParams, ok := a.Defaults.StaticUnit.InfraParamsByType[cfg.Type]
	if ok {
		cfg.InfraParams = cfg.InfraParams.Merge(defaultParams)

	}
	return cfg
}
