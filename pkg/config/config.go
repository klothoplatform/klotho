package config

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"

	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/klothoplatform/klotho/pkg/construct"
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

		Defaults            Defaults                        `json:"defaults" yaml:"defaults" toml:"defaults"`
		ExecutionUnits      map[string]*ExecutionUnit       `json:"execution_units,omitempty" yaml:"execution_units,omitempty" toml:"execution_units,omitempty"`
		StaticUnit          map[string]*StaticUnit          `json:"static_unit,omitempty" yaml:"static_unit,omitempty" toml:"static_unit,omitempty"`
		Exposed             map[string]*Expose              `json:"exposed,omitempty" yaml:"exposed,omitempty" toml:"exposed,omitempty"`
		PersistKv           map[string]*Persist             `json:"persist_kv,omitempty" yaml:"persist_kv,omitempty" toml:"persist_kv,omitempty"`
		PersistOrm          map[string]*Persist             `json:"persist_orm,omitempty" yaml:"persist_orm,omitempty" toml:"persist_orm,omitempty"`
		PersistFs           map[string]*Persist             `json:"persist_fs,omitempty" yaml:"persist_fs,omitempty" toml:"persist_fs,omitempty"`
		PersistSecrets      map[string]*Persist             `json:"persist_secrets,omitempty" yaml:"persist_secrets,omitempty" toml:"persist_secrets,omitempty"`
		PersistRedisNode    map[string]*Persist             `json:"persist_redis_node,omitempty" yaml:"persist_redis_node,omitempty" toml:"persist_redis_node,omitempty"`
		PersistRedisCluster map[string]*Persist             `json:"persist_redis_cluster,omitempty" yaml:"persist_redis_cluster,omitempty" toml:"persist_redis_cluster,omitempty"`
		Config              map[string]*Config              `json:"config,omitempty" yaml:"config,omitempty" toml:"config,omitempty"`
		Imports             map[construct.ResourceId]string `json:"imports,omitempty" yaml:"imports,omitempty" toml:"imports,omitempty"`
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

	return ReadConfigReader(fpath, f)
}

func ReadConfigReader(fpath string, reader io.Reader) (Application, error) {
	var appCfg Application

	switch filepath.Ext(fpath) {
	case ".json":
		err := json.NewDecoder(reader).Decode(&appCfg)
		appCfg.Format = "json"
		return appCfg, err

	case ".yaml", ".yml":
		err := yaml.NewDecoder(reader).Decode(&appCfg)
		appCfg.Format = "yaml"
		return appCfg, err

	case ".toml":
		err := toml.NewDecoder(reader).Decode(&appCfg)
		appCfg.Format = "toml"
		return appCfg, err
	}

	return appCfg, nil
}

func (a *Application) WriteTo(writer io.Writer) error {
	switch a.Format {
	case "json":
		return json.NewEncoder(writer).Encode(a)

	case "yaml":
		return yaml.NewEncoder(writer).Encode(a)

	case "toml":
		return toml.NewEncoder(writer).Encode(a)
	}

	return nil
}

func (a Application) GetResourceType(resource construct.Construct) string {
	switch resource.(type) {
	case *types.ExecutionUnit:
		cfg := a.GetExecutionUnit(resource.Id().Name)
		return cfg.Type

	case *types.StaticUnit:
		cfg := a.GetStaticUnit(resource.Id().Name)
		return cfg.Type

	case *types.Gateway:
		cfg := a.GetExpose(resource.Id().Name)
		return cfg.Type

	case *types.Fs:
		cfg := a.GetPersistFs(resource.Id().Name)
		return cfg.Type

	case *types.Kv:
		cfg := a.GetPersistKv(resource.Id().Name)
		return cfg.Type

	case *types.Orm:
		cfg := a.GetPersistOrm(resource.Id().Name)
		return cfg.Type

	case *types.Secrets:
		cfg := a.GetPersistSecrets(resource.Id().Name)
		return cfg.Type

	case *types.RedisCluster:
		cfg := a.GetPersistRedisCluster(resource.Id().Name)
		return cfg.Type

	case *types.RedisNode:
		cfg := a.GetPersistRedisNode(resource.Id().Name)
		return cfg.Type

	case *types.Config:
		cfg := a.GetConfig(resource.Id().Name)
		return cfg.Type
	}
	return ""
}

// UpdateForResources mutates the Application config for all the resources, applying the defaults.
func (a *Application) UpdateForResources(res []construct.Construct) {
	for _, r := range res {
		switch r.(type) {
		case *types.ExecutionUnit:
			cfg := a.GetExecutionUnit(r.Id().Name)
			a.ExecutionUnits[r.Id().Name] = &cfg

		case *types.StaticUnit:
			cfg := a.GetStaticUnit(r.Id().Name)
			a.StaticUnit[r.Id().Name] = &cfg

		case *types.Gateway:
			cfg := a.GetExpose(r.Id().Name)
			a.Exposed[r.Id().Name] = &cfg

		case *types.Fs:
			cfg := a.GetPersistFs(r.Id().Name)
			a.PersistFs[r.Id().Name] = &cfg

		case *types.Kv:
			cfg := a.GetPersistKv(r.Id().Name)
			a.PersistKv[r.Id().Name] = &cfg

		case *types.Orm:
			cfg := a.GetPersistOrm(r.Id().Name)
			a.PersistOrm[r.Id().Name] = &cfg

		case *types.Secrets:
			cfg := a.GetPersistSecrets(r.Id().Name)
			a.PersistSecrets[r.Id().Name] = &cfg

		case *types.RedisCluster:
			cfg := a.GetPersistRedisCluster(r.Id().Name)
			a.PersistRedisCluster[r.Id().Name] = &cfg

		case *types.RedisNode:
			cfg := a.GetPersistRedisNode(r.Id().Name)
			a.PersistRedisNode[r.Id().Name] = &cfg
		}
	}
}

func ConvertToInfraParams(p any) InfraParams {
	jsonString, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}
	params := InfraParams{}
	err = json.Unmarshal(jsonString, &params)
	if err != nil {
		panic(err)
	}
	return params
}

func ConvertFromInfraParams[T any](params InfraParams) T {
	jsonString, err := json.Marshal(params)
	if err != nil {
		panic(err)
	}
	var out T
	err = json.Unmarshal(jsonString, &out)
	if err != nil {
		panic(err)
	}
	return out
}

func (a *Application) MergeDefaults(other Defaults) {
	a.Defaults.ExecutionUnit.ApplyDefaults(other.ExecutionUnit)
	a.Defaults.Expose.ApplyDefaults(other.Expose)
	a.Defaults.PersistFs.ApplyDefaults(other.PersistFs)
	a.Defaults.PersistKv.ApplyDefaults(other.PersistKv)
	a.Defaults.PersistOrm.ApplyDefaults(other.PersistOrm)
	a.Defaults.PersistRedisCluster.ApplyDefaults(other.PersistRedisCluster)
	a.Defaults.PersistRedisNode.ApplyDefaults(other.PersistRedisNode)
	a.Defaults.PersistSecrets.ApplyDefaults(other.PersistSecrets)
	a.Defaults.PubSub.ApplyDefaults(other.PubSub)
	a.Defaults.StaticUnit.ApplyDefaults(other.StaticUnit)
	a.Defaults.Config.ApplyDefaults(other.Config)
}

func (cfg *KindDefaults) ApplyDefaults(dflt KindDefaults) {
	if cfg.Type == "" {
		cfg.Type = dflt.Type
	}
	if cfg.InfraParamsByType == nil {
		cfg.InfraParamsByType = make(map[string]InfraParams)
	}
	for name, unit := range dflt.InfraParamsByType {
		params := cfg.InfraParamsByType[name]
		params.ApplyDefaults(unit)
		cfg.InfraParamsByType[name] = params
	}
}

var (
	MaxDepth = 32
)

// ApplyDefaults applies the defaults to params
func (params *InfraParams) ApplyDefaults(dflt InfraParams) {
	if len(dflt) == 0 {
		return
	}
	if *params == nil {
		*params = make(InfraParams)
	}
	merge(*params, dflt, 0)
}

// merge applies all k,v pairs from src into dst. If the value is a map, it will
// try to recusively merge those as well. When keys conflict, dst will win out
// because this is used for ApplyDefaults where dst is the specific annotation's
// values and src is the default values.
func merge(dst, src map[string]interface{}, depth int) {
	if depth > MaxDepth {
		panic("merge recursion max depth exceeded")
	}
	if dst == nil {
		panic("destination map is nil")
	}
	for key, srcVal := range src {
		if dstVal, ok := dst[key]; ok {
			srcMap, srcMapOk := mapify(srcVal)
			dstMap, dstMapOk := mapify(dstVal)
			if srcMapOk && dstMapOk {
				merge(dstMap, srcMap, depth+1)
				srcVal = dstMap
			} else {
				continue
			}
		}
		dst[key] = srcVal
	}
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

func overrideValue[T comparable](src *T, override T) {
	var zero T
	if override == zero {
		return
	}
	*src = override
}

// ValueOrDefault returns the input value if it is not nil or zero, otherwise it returns the default value
func ValueOrDefault[T any](input T, defaultValue T) T {
	t := reflect.ValueOf(input)
	if !t.IsValid() || (t.IsValid() && t.IsZero()) {
		return defaultValue
	}
	return input
}
