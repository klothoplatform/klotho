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
	cfg := StaticUnit{
		Type: a.Defaults.StaticUnit.Type,
	}

	ecfg, hasOverride := a.StaticUnit[id]
	if hasOverride {
		overrideValue(&cfg.Type, ecfg.Type)
		overrideValue(&cfg.ContentDeliveryNetwork, ecfg.ContentDeliveryNetwork)
		cfg.InfraParams = ecfg.InfraParams
	}
	cfg.InfraParams.ApplyDefaults(a.Defaults.StaticUnit.InfraParamsByType[cfg.Type])

	return cfg
}
