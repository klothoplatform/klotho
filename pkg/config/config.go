package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

type (
	// Application is used to define the configuration for the application being compiled by Klotho.
	//The application configuration contains the necessary information to depict the architecture as well as klotho compilation configuration.
	Application struct {
		AppName  string `json:"app" yaml:"app" toml:"app"`
		Provider string `json:"provider" yaml:"provider" toml:"provider"`

		// Format is what format the file was originally in so that when we output
		// to compiled, it keeps the same format.
		Format string `json:"-" yaml:"-" toml:"-"`

		Path   string `json:"path" yaml:"path" toml:"path"`
		OutDir string `json:"out_dir" yaml:"out_dir" toml:"out_dir"`

		Defaults            Defaults                  `json:"defaults" yaml:"defaults" toml:"defaults"`
		ExecutionUnits      map[string]*ExecutionUnit `json:"execution_units,omitempty" yaml:"execution_units,omitempty" toml:"execution_units,omitempty"`
		StaticUnit          map[string]*StaticUnit    `json:"static_unit,omitempty" yaml:"static_unit,omitempty" toml:"static_unit,omitempty"`
		Exposed             map[string]*Expose        `json:"exposed,omitempty" yaml:"exposed,omitempty" toml:"exposed,omitempty"`
		PersistKv           map[string]*Persist       `json:"persist_kv,omitempty" yaml:"persist_kv,omitempty" toml:"persist_kv,omitempty"`
		PersistOrm          map[string]*Persist       `json:"persist_orm,omitempty" yaml:"persist_orm,omitempty" toml:"persist_orm,omitempty"`
		PersistFs           map[string]*Persist       `json:"persist_fs,omitempty" yaml:"persist_fs,omitempty" toml:"persist_fs,omitempty"`
		PersistSecrets      map[string]*Persist       `json:"persist_secrets,omitempty" yaml:"persist_secrets,omitempty" toml:"persist_secrets,omitempty"`
		PersistRedisNode    map[string]*Persist       `json:"persist_redis_node,omitempty" yaml:"persist_redis_node,omitempty" toml:"persist_redis_node,omitempty"`
		PersistRedisCluster map[string]*Persist       `json:"persist_redis_cluster,omitempty" yaml:"persist_redis_cluster,omitempty" toml:"persist_redis_cluster,omitempty"`
		PubSub              map[string]*PubSub        `json:"pubsub,omitempty" yaml:"pubsub,omitempty" toml:"pubsub,omitempty"`
		Config              map[string]*Config        `json:"config,omitempty" yaml:"config,omitempty" toml:"config,omitempty"`
	}

	ContentDeliveryNetwork struct {
		Id string `json:"id,omitempty" yaml:"id,omitempty" toml:"id,omitempty"`
	}

	InfraParams map[string]interface{}

	// Defaults represent the default characteristics the application will be configured with on Klotho compilation
	// If a new field is added to defaults, that field will need to be added to the MergeDefaults method
	Defaults struct {
		ExecutionUnit       KindDefaults `json:"execution_unit" yaml:"execution_unit" toml:"execution_unit"`
		StaticUnit          KindDefaults `json:"static_unit" yaml:"static_unit" toml:"static_unit"`
		Expose              KindDefaults `json:"expose" yaml:"expose" toml:"expose"`
		PersistKv           KindDefaults `json:"persist_kv,omitempty" yaml:"persist_kv,omitempty" toml:"persist_kv,omitempty"`
		PersistOrm          KindDefaults `json:"persist_orm,omitempty" yaml:"persist_orm,omitempty" toml:"persist_orm,omitempty"`
		PersistFs           KindDefaults `json:"persist_fs,omitempty" yaml:"persist_fs,omitempty" toml:"persist_fs,omitempty"`
		PersistSecrets      KindDefaults `json:"persist_secrets,omitempty" yaml:"persist_secrets,omitempty" toml:"persist_secrets,omitempty"`
		PersistRedisNode    KindDefaults `json:"persist_redis_node,omitempty" yaml:"persist_redis_node,omitempty" toml:"persist_redis_node,omitempty"`
		PersistRedisCluster KindDefaults `json:"persist_redis_cluster,omitempty" yaml:"persist_redis_cluster,omitempty" toml:"persist_redis_cluster,omitempty"`
		PubSub              KindDefaults `json:"pubsub" yaml:"pubsub" toml:"pubsub"`
		Config              KindDefaults `json:"config" yaml:"config" toml:"config"`
	}

	KindDefaults struct {
		Type              string                 `json:"type" yaml:"type" toml:"type"`
		InfraParamsByType map[string]InfraParams `json:"infra_params_by_type,omitempty" yaml:"infra_params_by_type,omitempty" toml:"infra_params_by_type,omitempty"`
	}
)

func (appCfg *Application) EnsureMapsExist() {
	if appCfg.ExecutionUnits == nil {
		appCfg.ExecutionUnits = make(map[string]*ExecutionUnit)
	}
	if appCfg.Exposed == nil {
		appCfg.Exposed = make(map[string]*Expose)
	}
	if appCfg.PersistFs == nil {
		appCfg.PersistFs = make(map[string]*Persist)
	}
	if appCfg.PersistKv == nil {
		appCfg.PersistKv = make(map[string]*Persist)
	}
	if appCfg.PersistOrm == nil {
		appCfg.PersistOrm = make(map[string]*Persist)
	}
	if appCfg.PersistSecrets == nil {
		appCfg.PersistSecrets = make(map[string]*Persist)
	}
	if appCfg.PersistRedisNode == nil {
		appCfg.PersistRedisNode = make(map[string]*Persist)
	}
	if appCfg.PersistRedisCluster == nil {
		appCfg.PersistRedisCluster = make(map[string]*Persist)
	}
	if appCfg.PubSub == nil {
		appCfg.PubSub = make(map[string]*PubSub)
	}
	if appCfg.StaticUnit == nil {
		appCfg.StaticUnit = make(map[string]*StaticUnit)
	}
	if appCfg.Config == nil {
		appCfg.Config = make(map[string]*Config)
	}
}

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
		cfg := a.GetExpose(key.Name)
		return cfg.Type

	case string(core.PersistFileKind):
		cfg := a.GetPersistFs(key.Name)
		return cfg.Type

	case string(core.PersistKVKind):
		cfg := a.GetPersistKv(key.Name)
		return cfg.Type

	case string(core.PersistORMKind):
		cfg := a.GetPersistOrm(key.Name)
		return cfg.Type

	case string(core.PersistSecretKind):
		cfg := a.GetPersistSecrets(key.Name)
		return cfg.Type

	case string(core.PersistRedisClusterKind):
		cfg := a.GetPersistRedisCluster(key.Name)
		return cfg.Type

	case string(core.PersistRedisNodeKind):
		cfg := a.GetPersistRedisNode(key.Name)
		return cfg.Type

	case core.PubSubKind:
		cfg := a.GetPubSub(key.Name)
		return cfg.Type

	case core.ConfigKind:
		cfg := a.GetConfig(key.Name)
		return cfg.Type
	}
	return ""
}

// UpdateForResources mutates the Application config for all the resources, applying the defaults.
func (a *Application) UpdateForResources(res []core.CloudResource) {
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
			cfg := a.GetExpose(key.Name)
			a.Exposed[key.Name] = &cfg

		case string(core.PersistFileKind):
			cfg := a.GetPersistFs(key.Name)
			a.PersistFs[key.Name] = &cfg

		case string(core.PersistKVKind):
			cfg := a.GetPersistKv(key.Name)
			a.PersistKv[key.Name] = &cfg

		case string(core.PersistORMKind):
			cfg := a.GetPersistOrm(key.Name)
			a.PersistOrm[key.Name] = &cfg

		case string(core.PersistSecretKind):
			cfg := a.GetPersistSecrets(key.Name)
			a.PersistSecrets[key.Name] = &cfg

		case string(core.PersistRedisClusterKind):
			cfg := a.GetPersistRedisCluster(key.Name)
			a.PersistRedisCluster[key.Name] = &cfg

		case string(core.PersistRedisNodeKind):
			cfg := a.GetPersistRedisNode(key.Name)
			a.PersistRedisNode[key.Name] = &cfg

		case core.PubSubKind:
			cfg := a.GetPubSub(key.Name)
			a.PubSub[key.Name] = &cfg
		}
	}
}

func ConvertToInfraParams(p any) InfraParams {
	jsonString, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}
	params := InfraParams{}
	json.Unmarshal(jsonString, &params)
	return params
}

func (a *Application) MergeDefaults(other Defaults) {
	a.Defaults.ExecutionUnit.Merge(other.ExecutionUnit)
	a.Defaults.Expose.Merge(other.Expose)
	a.Defaults.PersistFs.Merge(other.PersistFs)
	a.Defaults.PersistKv.Merge(other.PersistKv)
	a.Defaults.PersistOrm.Merge(other.PersistOrm)
	a.Defaults.PersistRedisCluster.Merge(other.PersistRedisCluster)
	a.Defaults.PersistRedisNode.Merge(other.PersistRedisNode)
	a.Defaults.PersistSecrets.Merge(other.PersistSecrets)
	a.Defaults.PubSub.Merge(other.PubSub)
	a.Defaults.StaticUnit.Merge(other.StaticUnit)
	a.Defaults.Config.Merge(other.Config)
}

func (cfg *KindDefaults) Merge(other KindDefaults) {
	if other.Type != "" && cfg.Type == "" {
		cfg.Type = other.Type
	}
	if cfg.InfraParamsByType == nil {
		cfg.InfraParamsByType = make(map[string]InfraParams)
	}
	for name, unit := range other.InfraParamsByType {
		paramsByType := cfg.InfraParamsByType[name]
		cfg.InfraParamsByType[name] = paramsByType.Merge(unit)
	}
}

var (
	MaxDepth = 32
)

// Merge recursively merges the src and dst maps. Key conflicts are resolved by
// preferring src, or recursively descending, if both src and dst are maps.
func (src InfraParams) Merge(dst InfraParams) map[string]interface{} {
	return merge(dst, src, 0)
}

func merge(dst, src map[string]interface{}, depth int) map[string]interface{} {
	if depth > MaxDepth {
		panic("too deep!")
	}
	if dst == nil {
		dst = make(map[string]interface{})
	}
	for key, srcVal := range src {
		if dstVal, ok := dst[key]; ok {
			srcMap, srcMapOk := mapify(srcVal)
			dstMap, dstMapOk := mapify(dstVal)
			if srcMapOk && dstMapOk {
				srcVal = merge(dstMap, srcMap, depth+1)
			}
		}
		dst[key] = srcVal
	}
	return dst
}

func mapify(i interface{}) (map[string]interface{}, bool) {
	value := reflect.ValueOf(i)
	if value.Kind() == reflect.Map {
		m := map[string]interface{}{}
		for _, k := range value.MapKeys() {
			m[k.String()] = value.MapIndex(k).Interface()
		}
		return m, true
	}
	return nil, false
}
