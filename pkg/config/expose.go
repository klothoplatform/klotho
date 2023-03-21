package config

type (
	// Expose is how expose Klotho constructs are represented in the klotho configuration
	Expose struct {
		Type                   string                 `json:"type" yaml:"type" toml:"type"`
		ContentDeliveryNetwork ContentDeliveryNetwork `json:"content_delivery_network,omitempty" yaml:"content_delivery_network,omitempty" toml:"content_delivery_network,omitempty"`
		InfraParams            InfraParams            `json:"infra_params,omitempty" yaml:"infra_params,omitempty" toml:"infra_params,omitempty"`
	}

	GatewayTypeParams struct {
		ApiType string
	}

	LoadBalancerTypeParams struct {
	}
)

// GetExpose returns the `Expose` config for the resource specified by `id`
// merged with the defaults.
func (a Application) GetExpose(id string) Expose {
	cfg := Expose{}
	if ecfg, ok := a.Exposed[id]; ok {
		defaultParams, ok := a.Defaults.Expose.InfraParamsByType[ecfg.Type]
		if ok {
			if ecfg.InfraParams == nil {
				ecfg.InfraParams = defaultParams
			} else {
				ecfg.InfraParams = ecfg.InfraParams.Merge(defaultParams)
			}
		}
		return *ecfg
	}
	cfg.Type = a.Defaults.Expose.Type
	defaultParams, ok := a.Defaults.Expose.InfraParamsByType[cfg.Type]
	if ok {
		cfg.InfraParams = cfg.InfraParams.Merge(defaultParams)

	}
	return cfg
}
