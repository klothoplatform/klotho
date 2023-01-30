package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/pelletier/go-toml/v2"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type (
	Application struct {
		AppName  string `json:"app" yaml:"app" toml:"app"`
		Provider string `json:"provider" yaml:"provider" toml:"provider"`

		// Format is what format the file was originally in so that when we output
		// to compiled, it keeps the same format.
		Format string `json:"-" yaml:"-" toml:"-"`

		Path   string `json:"path" yaml:"path" toml:"path"`
		OutDir string `json:"out_dir" yaml:"out_dir" toml:"out_dir"`

		Defaults       Defaults                  `json:"defaults" yaml:"defaults" toml:"defaults"`
		ExecutionUnits map[string]*ExecutionUnit `json:"execution_units,omitempty" yaml:"execution_units,omitempty" toml:"execution_units,omitempty"`
		StaticUnit     map[string]*StaticUnit    `json:"static_unit,omitempty" yaml:"static_unit,omitempty" toml:"static_unit,omitempty"`
		Exposed        map[string]*Expose        `json:"exposed,omitempty" yaml:"exposed,omitempty" toml:"exposed,omitempty"`
		Persisted      map[string]*Persist       `json:"persisted,omitempty" yaml:"persisted,omitempty" toml:"persisted,omitempty"`
		PubSub         map[string]*PubSub        `json:"pubsub,omitempty" yaml:"pubsub,omitempty" toml:"pubsub,omitempty"`
	}
	Expose struct {
		Type                   string                 `json:"type" yaml:"type" toml:"type"`
		ContentDeliveryNetwork ContentDeliveryNetwork `json:"content_delivery_network,omitempty" yaml:"content_delivery_network,omitempty" toml:"content_delivery_network,omitempty"`
		InfraParams            InfraParams            `json:"pulumi_params,omitempty" yaml:"pulumi_params,omitempty" toml:"pulumi_params,omitempty"`
	}

	Persist struct {
		Type        string      `json:"type" yaml:"type" toml:"type"`
		InfraParams InfraParams `json:"pulumi_params,omitempty" yaml:"pulumi_params,omitempty" toml:"pulumi_params,omitempty"`
	}

	ExecutionUnit struct {
		Type             string            `json:"type" yaml:"type" toml:"type"`
		NetworkPlacement string            `json:"network_placement,omitempty" yaml:"network_placement,omitempty" toml:"network_placement,omitempty"`
		HelmChartOptions *HelmChartOptions `json:"helm_chart_options,omitempty" yaml:"helm_chart_options,omitempty" toml:"helm_chart_options,omitempty"`
		InfraParams      InfraParams       `json:"pulumi_params,omitempty" yaml:"pulumi_params,omitempty" toml:"pulumi_params,omitempty"`
	}

	// A HelmChartOptions represents configuration for execution units attempting to generate helm charts
	HelmChartOptions struct {
		Directory   string   `json:"directory,omitempty" yaml:"directory,omitempty" toml:"directory,omitempty"` // Directory signals the directory which will contain the helm chart outputs
		Install     bool     `json:"install,omitempty" yaml:"install,omitempty" toml:"install,omitempty"`
		ValuesFiles []string `json:"values_files,omitempty" yaml:"values_files,omitempty" toml:"values_files,omitempty"`
	}

	PubSub struct {
		Type        string      `json:"type" yaml:"type" toml:"type"`
		InfraParams InfraParams `json:"pulumi_params,omitempty" yaml:"pulumi_params,omitempty" toml:"pulumi_params,omitempty"`
	}

	StaticUnit struct {
		Type                   string                 `json:"type" yaml:"type" toml:"type"`
		InfraParams            InfraParams            `json:"pulumi_params,omitempty" yaml:"pulumi_params,omitempty" toml:"pulumi_params,omitempty"`
		ContentDeliveryNetwork ContentDeliveryNetwork `json:"content_delivery_network,omitempty" yaml:"content_delivery_network,omitempty" toml:"content_delivery_network,omitempty"`
	}

	Defaults struct {
		ExecutionUnit KindDefaults        `json:"execution_unit" yaml:"execution_unit" toml:"execution_unit"`
		StaticUnit    KindDefaults        `json:"static_unit" yaml:"static_unit" toml:"static_unit"`
		Expose        KindDefaults        `json:"expose" yaml:"expose" toml:"expose"`
		Persist       PersistKindDefaults `json:"persist" yaml:"persist" toml:"persist"`
		PubSub        KindDefaults        `json:"pubsub" yaml:"pubsub" toml:"pubsub"`
	}

	KindDefaults struct {
		Type              string                 `json:"type" yaml:"type" toml:"type"`
		InfraParamsByType map[string]InfraParams `json:"pulumi_params_by_type,omitempty" yaml:"pulumi_params_by_type,omitempty" toml:"pulumi_params_by_type,omitempty"`
	}

	PersistKindDefaults struct {
		KV           KindDefaults `json:"kv" yaml:"kv" toml:"kv"`
		FS           KindDefaults `json:"fs" yaml:"fs" toml:"fs"`
		Secret       KindDefaults `json:"secret" yaml:"secret" toml:"secret"`
		ORM          KindDefaults `json:"orm" yaml:"orm" toml:"orm"`
		RedisNode    KindDefaults `json:"redis_node" yaml:"redis_node" toml:"redis_node"`
		RedisCluster KindDefaults `json:"redis_cluster" yaml:"redis_cluster" toml:"redis_cluster"`
	}

	ContentDeliveryNetwork struct {
		Id string `json:"id,omitempty" yaml:"id,omitempty" toml:"id,omitempty"`
	}

	// InfraParams are passed as-is to the generated IaC
	InfraParams map[string]interface{}
)

func ReadConfig(fpath string) (Application, error) {
	var appCfg Application

	f, err := os.Open(fpath)
	if err != nil {
		return appCfg, err
	}
	defer f.Close() // nolint:errcheck

	switch filepath.Ext(fpath) {
	case ".json":
		err = json.NewDecoder(f).Decode(&appCfg)
		appCfg.Format = "json"

	case ".yaml", ".yml":
		err = yaml.NewDecoder(f).Decode(&appCfg)
		appCfg.Format = "yaml"

	case ".toml":
		err = toml.NewDecoder(f).Decode(&appCfg)
		appCfg.Format = "toml"
	}
	return appCfg, err
}

func (cfg *InfraParams) Merge(other InfraParams) {
	if *cfg == nil {
		*cfg = make(InfraParams)
	}
	// TODO do a deeper merge
	for k, v := range other {
		(*cfg)[k] = v
	}

}

func (cfg *KindDefaults) Merge(other KindDefaults) {
	if other.Type != "" {
		cfg.Type = other.Type
	}
	if cfg.InfraParamsByType == nil {
		cfg.InfraParamsByType = make(map[string]InfraParams)
	}
	for name, unit := range other.InfraParamsByType {
		paramsByType := cfg.InfraParamsByType[name]
		paramsByType.Merge(unit)
		cfg.InfraParamsByType[name] = paramsByType
	}
}

func (cfg *ExecutionUnit) Merge(other ExecutionUnit) {
	if other.Type != "" {
		cfg.Type = other.Type
	}
	if cfg.Type == "fargate" {
		zap.S().Warn("Execution unit type 'fargate' is now renamed to 'ecs'")
		cfg.Type = "ecs"
	}
	cfg.NetworkPlacement = other.NetworkPlacement
	if other.NetworkPlacement == "" {
		cfg.NetworkPlacement = "private"
	}
	cfg.HelmChartOptions = other.HelmChartOptions
	cfg.InfraParams.Merge(other.InfraParams)
}

func (cfg *Expose) Merge(other Expose) {
	if other.Type != "" {
		cfg.Type = other.Type
	}
	cfg.ContentDeliveryNetwork = other.ContentDeliveryNetwork
	cfg.InfraParams.Merge(other.InfraParams)
}

func (cfg *Persist) Merge(other Persist) {
	if other.Type != "" {
		cfg.Type = other.Type
	}
	cfg.InfraParams.Merge(other.InfraParams)
}

func (cfg *PubSub) Merge(other PubSub) {
	if other.Type != "" {
		cfg.Type = other.Type
	}
	cfg.InfraParams.Merge(other.InfraParams)
}

func (cfg *StaticUnit) Merge(other StaticUnit) {
	if other.Type != "" {
		cfg.Type = other.Type
	}
	cfg.ContentDeliveryNetwork = other.ContentDeliveryNetwork
	cfg.InfraParams.Merge(other.InfraParams)
}

func (cfg *Persist) GetPersistDefaults(other PersistKindDefaults, persistType core.PersistKind) Persist {
	var t *KindDefaults

	switch persistType {
	case core.PersistKVKind:
		t = &other.KV

	case core.PersistFileKind:
		t = &other.FS

	case core.PersistSecretKind:
		t = &other.Secret

	case core.PersistORMKind:
		t = &other.ORM

	case core.PersistRedisClusterKind:
		t = &other.RedisCluster
	case core.PersistRedisNodeKind:
		t = &other.RedisNode
	}
	if t != nil {
		defaultType := t.Type
		if cfg.Type != "" {
			defaultType = cfg.Type
		}
		dcfg := Persist{
			Type:        defaultType,
			InfraParams: t.InfraParamsByType[defaultType],
		}
		return dcfg
	}
	return *cfg
}

func (cfg *PersistKindDefaults) Merge(other PersistKindDefaults) {
	cfg.FS.Merge(other.FS)
	cfg.KV.Merge(other.KV)
	cfg.Secret.Merge(other.Secret)
	cfg.ORM.Merge(other.ORM)
	cfg.RedisCluster.Merge(other.RedisCluster)
	cfg.RedisNode.Merge(other.RedisNode)
}

func (cfg *Defaults) Merge(other Defaults) {
	cfg.ExecutionUnit.Merge(other.ExecutionUnit)
	cfg.Expose.Merge(other.Expose)
	cfg.Persist.Merge(other.Persist)
	cfg.PubSub.Merge(other.PubSub)
	cfg.StaticUnit.Merge(other.StaticUnit)
}

// GetExecutionUnit returns the `ExecutionUnit` config for the resource specified by `id`
// merged with the defaults.
func (a Application) GetExecutionUnit(id string) ExecutionUnit {
	cfg := ExecutionUnit{}
	defaultType := a.Defaults.ExecutionUnit.Type
	if ecfg, ok := a.ExecutionUnits[id]; ok {
		unitType := ecfg.Type
		if unitType == "" {
			unitType = defaultType
		}
		defaults := ExecutionUnit{
			Type:        unitType,
			InfraParams: a.Defaults.ExecutionUnit.InfraParamsByType[unitType],
		}
		if ecfg.Type == defaults.Type || ecfg.Type == "" {
			cfg.Merge(defaults)
		}
		cfg.Merge(*ecfg)
	} else {
		defaults := ExecutionUnit{
			Type:        defaultType,
			InfraParams: a.Defaults.ExecutionUnit.InfraParamsByType[defaultType],
		}
		cfg.Merge(defaults)
	}
	return cfg
}

// GetExposed returns the `Expose` config for the resource specified by `id`
// merged with the defaults.
func (a Application) GetExposed(id string) Expose {
	cfg := Expose{}
	defaultType := a.Defaults.Expose.Type
	if ecfg, ok := a.Exposed[id]; ok {
		exposeType := ecfg.Type
		if exposeType == "" {
			exposeType = defaultType
		}
		defaults := Expose{
			Type:        exposeType,
			InfraParams: a.Defaults.Expose.InfraParamsByType[exposeType],
		}
		if ecfg.Type == defaults.Type || ecfg.Type == "" {
			cfg.Merge(defaults)
		}
		cfg.Merge(*ecfg)
	} else {
		defaults := Expose{
			Type:        defaultType,
			InfraParams: a.Defaults.Expose.InfraParamsByType[defaultType],
		}
		cfg.Merge(defaults)
	}
	return cfg
}

// GetPersisted returns the `Persist` config for the resource specified by `id`
// merged with the defaults for `kind`.
func (a Application) GetPersisted(id string, kind core.PersistKind) Persist {
	cfg := Persist{}
	if ecfg, ok := a.Persisted[id]; ok {
		defaults := ecfg.GetPersistDefaults(a.Defaults.Persist, kind)
		if ecfg.Type == defaults.Type || ecfg.Type == "" {
			cfg.Merge(defaults)
		}
		cfg.Merge(*ecfg)
	} else {
		defaults := cfg.GetPersistDefaults(a.Defaults.Persist, kind)
		cfg.Merge(defaults)
	}
	return cfg
}

// GetPubSub returns the `PubSub` config for the resource specified by `id`
func (a Application) GetPubSub(id string) PubSub {
	cfg := PubSub{}
	defaultType := a.Defaults.PubSub.Type
	if ecfg, ok := a.PubSub[id]; ok {
		pubsubType := ecfg.Type
		if pubsubType == "" {
			pubsubType = defaultType
		}
		defaults := PubSub{
			Type:        pubsubType,
			InfraParams: a.Defaults.PubSub.InfraParamsByType[pubsubType],
		}
		if ecfg.Type == defaults.Type || ecfg.Type == "" {
			cfg.Merge(defaults)
		}
		cfg.Merge(*ecfg)
	} else {
		defaultType := a.Defaults.PubSub.Type
		defaults := PubSub{
			Type:        defaultType,
			InfraParams: a.Defaults.PubSub.InfraParamsByType[defaultType],
		}
		cfg.Merge(defaults)
	}
	return cfg
}

// GetStaticUnit returns the `StaticUnit` config for the resource specified by `id`
func (a Application) GetStaticUnit(id string) StaticUnit {
	cfg := StaticUnit{}
	defaultType := a.Defaults.StaticUnit.Type
	if ecfg, ok := a.StaticUnit[id]; ok {
		unitType := ecfg.Type
		if unitType == "" {
			unitType = defaultType
		}
		defaults := StaticUnit{
			Type:        unitType,
			InfraParams: a.Defaults.StaticUnit.InfraParamsByType[unitType],
		}
		if ecfg.Type == defaults.Type || ecfg.Type == "" {
			cfg.Merge(defaults)
		}
		cfg.Merge(*ecfg)
	} else {
		defaults := StaticUnit{
			Type:        defaultType,
			InfraParams: a.Defaults.StaticUnit.InfraParamsByType[defaultType],
		}
		cfg.Merge(defaults)
	}
	return cfg
}

func (a Application) GetResourceType(resource core.CloudResource) string {
	key := resource.Key()
	switch key.Kind {
	case core.ExecutionUnitKind:
		cfg := a.GetExecutionUnit(key.Name)
		return cfg.Type

	case core.StaticUnitKind:
		cfg := a.GetStaticUnit(key.Name)
		return cfg.Type

	case core.GatewayKind:
		cfg := a.GetExposed(key.Name)
		return cfg.Type

	case string(core.PersistFileKind), string(core.PersistKVKind), string(core.PersistORMKind), string(core.PersistSecretKind), string(core.PersistRedisClusterKind), string(core.PersistRedisNodeKind):
		cfg := a.GetPersisted(key.Name, core.PersistKind(key.Kind))
		return cfg.Type

	case core.PubSubKind:
		cfg := a.GetPubSub(key.Name)
		return cfg.Type
	}
	return ""
}

// UpdateForResources mutates the Application config for all the resources, applying the defaults.
func (a *Application) UpdateForResources(res []core.CloudResource) {
	if a.ExecutionUnits == nil {
		a.ExecutionUnits = make(map[string]*ExecutionUnit)
	}
	if a.Exposed == nil {
		a.Exposed = make(map[string]*Expose)
	}
	if a.Persisted == nil {
		a.Persisted = make(map[string]*Persist)
	}
	if a.PubSub == nil {
		a.PubSub = make(map[string]*PubSub)
	}
	if a.StaticUnit == nil {
		a.StaticUnit = make(map[string]*StaticUnit)
	}
	for _, r := range res {
		key := r.Key()
		switch key.Kind {
		case core.ExecutionUnitKind:
			cfg := a.GetExecutionUnit(key.Name)
			a.ExecutionUnits[key.Name] = &cfg

		case core.StaticUnitKind:
			cfg := a.GetStaticUnit(key.Name)
			a.StaticUnit[key.Name] = &cfg

		case core.GatewayKind:
			cfg := a.GetExposed(key.Name)
			a.Exposed[key.Name] = &cfg

		case string(core.PersistFileKind), string(core.PersistKVKind), string(core.PersistORMKind), string(core.PersistSecretKind), string(core.PersistRedisClusterKind), string(core.PersistRedisNodeKind):
			cfg := a.GetPersisted(key.Name, core.PersistKind(key.Kind))
			a.Persisted[key.Name] = &cfg

		case core.PubSubKind:
			cfg := a.GetPubSub(key.Name)
			a.PubSub[key.Name] = &cfg
		}
	}
}
