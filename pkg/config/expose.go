package config

import (
	"encoding/json"

	"go.uber.org/zap"
)

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
		if ecfg.InfraParams == nil {
			ecfg.InfraParams = make(InfraParams)
		}
		defaultParams, ok := a.Defaults.Expose.InfraParamsByType[ecfg.Type]
		if ok {
			ecfg.InfraParams = ecfg.InfraParams.Merge(defaultParams)
		}
		return *ecfg
	}
	cfg.Type = a.Defaults.Expose.Type
	defaultParams, ok := a.Defaults.Expose.InfraParamsByType[cfg.Type]
	cfg.InfraParams = make(InfraParams)
	if ok {
		cfg.InfraParams = cfg.InfraParams.Merge(defaultParams)

	}
	return cfg
}

func (a Application) GetExposeKindParams(cfg Expose) interface{} {

	infraParams := cfg.InfraParams
	jsonString, err := json.Marshal(infraParams)
	if err != nil {
		zap.S().Debug(err)
	}

	gatewayParams := GatewayTypeParams{}
	if err := json.Unmarshal(jsonString, &gatewayParams); err != nil {
		zap.S().Debug(err)
	} else {
		return gatewayParams
	}

	loadBalancerParams := LoadBalancerTypeParams{}
	if err := json.Unmarshal(jsonString, &loadBalancerParams); err != nil {
		zap.S().Debug(err)
	} else {
		return loadBalancerParams
	}

	return nil
}
