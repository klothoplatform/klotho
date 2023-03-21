package config

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/stretchr/testify/assert"
)

func Test_GetResourceType(t *testing.T) {
	tests := []struct {
		name     string
		cfg      Application
		resource core.CloudResource
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
			resource: &core.Gateway{Name: "test"},
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
			resource: &core.ExecutionUnit{Name: "test"},
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
			resource: &core.Persist{Name: "test", Kind: core.PersistKVKind},
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
		resources []core.CloudResource
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
			resources: []core.CloudResource{&core.Gateway{Name: "test"}, &core.ExecutionUnit{Name: "test"}, &core.Persist{Name: "test", Kind: core.PersistKVKind}},
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
				Exposed:        map[string]*Expose{"test": {Type: "apigateway", InfraParams: make(InfraParams)}},
				ExecutionUnits: map[string]*ExecutionUnit{"test": {Type: "lambda", NetworkPlacement: "private", EnvironmentVariables: make(map[string]string), InfraParams: make(InfraParams)}},
				PersistKv:      map[string]*Persist{"test": {Type: "dynamodb", InfraParams: make(InfraParams)}},
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
			tt.cfg.Merge(tt.defaults)
			assert.Equal(tt.want, tt.cfg)

		})
	}
}

func Test_MergeInfraParams(t *testing.T) {
	tests := []struct {
		name     string
		cfg      InfraParams
		defaults InfraParams
		want     map[string]interface{}
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
			want: map[string]interface{}{
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
			want: map[string]interface{}{
				"apigateway": map[string]interface{}{
					"key1": "value100",
					"key2": "value2",
				},
				"somethingelse": map[string]interface{}{"1": "2"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			result := tt.cfg.Merge(tt.defaults)
			assert.Equal(tt.want, result)

		})
	}
}
