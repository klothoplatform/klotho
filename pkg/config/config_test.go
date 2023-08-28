package config

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/stretchr/testify/assert"
)

func Test_GetResourceType(t *testing.T) {
	tests := []struct {
		name     string
		cfg      Application
		resource construct.Construct
		want     string
	}{
		{
			name: "expose",
			cfg: Application{
				Defaults: Defaults{
					Expose: KindDefaults{
						Type: "apigateway",
					},
				},
			},
			resource: &types.Gateway{Name: "test"},
			want:     "apigateway",
		},
		{
			name: "exec unit",
			cfg: Application{
				Defaults: Defaults{
					ExecutionUnit: KindDefaults{
						Type: "lambda",
					},
				},
			},
			resource: &types.ExecutionUnit{Name: "test"},
			want:     "lambda",
		},
		{
			name: "persist kv",
			cfg: Application{
				Defaults: Defaults{
					PersistKv: KindDefaults{
						Type: "dynamodb",
					},
				},
			},
			resource: &types.Kv{Name: "test"},
			want:     "dynamodb",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			testType := tt.cfg.GetResourceType(tt.resource)
			assert.Equal(tt.want, testType)

		})
	}
}

func Test_UpdateForResources(t *testing.T) {
	tests := []struct {
		name      string
		cfg       Application
		resources []construct.Construct
		want      Application
	}{
		{
			name: "expose",
			cfg: Application{
				Defaults: Defaults{
					Expose: KindDefaults{
						Type: "apigateway",
					},
					ExecutionUnit: KindDefaults{
						Type: "lambda",
					},
					PersistKv: KindDefaults{
						Type: "dynamodb",
					},
				},
			},
			resources: []construct.Construct{&types.Gateway{Name: "test"}, &types.ExecutionUnit{Name: "test"},
				&types.Kv{Name: "test"}},
			want: Application{
				Defaults: Defaults{
					Expose: KindDefaults{
						Type: "apigateway",
					},
					ExecutionUnit: KindDefaults{
						Type: "lambda",
					},
					PersistKv: KindDefaults{
						Type: "dynamodb",
					},
				},
				Exposed:        map[string]*Expose{"test": {Type: "apigateway"}},
				ExecutionUnits: map[string]*ExecutionUnit{"test": {Type: "lambda", NetworkPlacement: "private", EnvironmentVariables: make(map[string]string)}},
				PersistKv:      map[string]*Persist{"test": {Type: "dynamodb"}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			tt.cfg.EnsureMapsExist()
			tt.want.EnsureMapsExist()
			tt.cfg.UpdateForResources(tt.resources)
			assert.Equal(tt.want, tt.cfg)

		})
	}
}

func Test_MergeKindDefaults(t *testing.T) {
	tests := []struct {
		name     string
		cfg      KindDefaults
		defaults KindDefaults
		want     KindDefaults
	}{
		{
			name: "basic defaults merge",
			cfg: KindDefaults{
				Type: "apigateway",
				InfraParamsByType: map[string]InfraParams{
					"apigateway": {
						"key1": "value100",
					},
				},
			},
			defaults: KindDefaults{
				Type: "somethingelse",
				InfraParamsByType: map[string]InfraParams{
					"apigateway": {
						"key1": "value1",
						"key2": "value2",
					},
					"somethingelse": {"1": "2"},
				},
			},
			want: KindDefaults{
				Type: "apigateway",
				InfraParamsByType: map[string]InfraParams{
					"apigateway": {
						"key1": "value100",
						"key2": "value2",
					},
					"somethingelse": {"1": "2"},
				},
			},
		},
		{
			name: "basic defaults merge",
			cfg: KindDefaults{
				Type: "apigateway",
				InfraParamsByType: map[string]InfraParams{
					"apigateway": {
						"key1": "value100",
						"key2": "value2",
					},
				},
			},
			defaults: KindDefaults{
				Type: "somethingelse",
				InfraParamsByType: map[string]InfraParams{
					"apigateway": {
						"key1": 1234,
						"key2": []int{1234},
					},
				},
			},
			want: KindDefaults{
				Type: "apigateway",
				InfraParamsByType: map[string]InfraParams{
					"apigateway": {
						"key1": "value100",
						"key2": "value2",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			tt.cfg.ApplyDefaults(tt.defaults)
			assert.Equal(tt.want, tt.cfg)

		})
	}
}

func Test_MergeInfraParams(t *testing.T) {
	tests := []struct {
		name     string
		cfg      InfraParams
		defaults InfraParams
		want     InfraParams
	}{
		{
			name: "basic infra params merge",
			cfg: InfraParams{
				"key1": "value100",
			},
			defaults: InfraParams{
				"key1": "value1",
				"key2": "value2",
				"1":    "2",
			},
			want: InfraParams{
				"key1": "value100",
				"key2": "value2",
				"1":    "2",
			},
		},
		{
			name: "nested infra params merge",
			cfg: InfraParams{
				"apigateway": map[string]interface{}{
					"key1": "value100",
				},
			},
			defaults: InfraParams{
				"apigateway": map[string]interface{}{
					"key1": "value1",
					"key2": "value2",
				},
				"somethingelse": map[string]interface{}{"1": "2"},
			},
			want: InfraParams{
				"apigateway": map[string]interface{}{
					"key1": "value100",
					"key2": "value2",
				},
				"somethingelse": map[string]interface{}{"1": "2"},
			},
		},
		{
			name: "empty params",
			defaults: InfraParams{
				"key1": "value1",
			},
			want: map[string]interface{}{
				"key1": "value1",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			tt.cfg.ApplyDefaults(tt.defaults)
			assert.Equal(tt.want, tt.cfg)

		})
	}
}

func TestReadConfigReader(t *testing.T) {
	tests := []struct {
		name string
		cfg  Application
	}{
		{
			name: "empty config",
			cfg:  Application{},
		},
		{
			name: "config with imports",
			cfg: Application{
				Imports: map[construct.ResourceId]string{
					{Provider: "prov", Type: "type", Namespace: "ns", Name: "name"}: "1",
					{Provider: "prov", Type: "type", Name: "name"}:                  "2",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, cfgFmt := range []string{"json", "yaml", "toml"} {
				t.Run(cfgFmt, func(t *testing.T) {
					assert := assert.New(t)
					tt.cfg.Format = cfgFmt

					buf := new(bytes.Buffer)
					err := tt.cfg.WriteTo(buf)
					if !assert.NoError(err) {
						return
					}
					t.Logf("config:\n%s", buf.String())

					cfg, err := ReadConfigReader(fmt.Sprintf("klotho.%s", cfgFmt), buf)
					if !assert.NoError(err) {
						return
					}
					assert.Equal(tt.cfg, cfg)
				})
			}
		})
	}
}
