package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_GetPersistKv(t *testing.T) {
	tests := []struct {
		name string
		cfg  Application
		id   string
		want Persist
	}{
		{
			name: "get base kv",
			cfg: Application{
				Defaults: Defaults{
					PersistKv: KindDefaults{
						Type: "dynamodb",
						InfraParamsByType: map[string]InfraParams{
							"dynamodb": {"key1": "value1"},
						},
					},
				},
			},
			id: "test",
			want: Persist{
				Type: "dynamodb",
				InfraParams: InfraParams{
					"key1": "value1",
				},
			},
		},
		{
			name: "get base type params exec unit",
			cfg: Application{
				Defaults: Defaults{
					PersistKv: KindDefaults{
						Type: "dynamodb",
						InfraParamsByType: map[string]InfraParams{
							"dynamodb": {"key1": "value1"},
						},
					},
				},
				PersistKv: map[string]*Persist{
					"test": {Type: "dynamodb"},
				},
			},
			id: "test",
			want: Persist{
				Type: "dynamodb",
				InfraParams: InfraParams{
					"key1": "value1",
				},
			},
		},
		{
			name: "get config and add other default params",
			cfg: Application{
				Defaults: Defaults{
					PersistKv: KindDefaults{
						Type: "dynamodb",
						InfraParamsByType: map[string]InfraParams{
							"dynamodb": {"key1": "value1", "key2": "value2"},
						},
					},
				},
				PersistKv: map[string]*Persist{
					"test": {
						Type:        "dynamodb",
						InfraParams: map[string]interface{}{"key2": "value200"},
					},
				},
			},
			id: "test",
			want: Persist{
				Type: "dynamodb",
				InfraParams: InfraParams{
					"key1": "value1",
					"key2": "value200",
				},
			},
		},
		{
			name: "get config and with no default params",
			cfg: Application{
				Defaults: Defaults{
					PersistKv: KindDefaults{
						Type: "dynamodb",
						InfraParamsByType: map[string]InfraParams{
							"dynamodb": {"key1": "value1", "key2": "value2"},
						},
					},
				},
				PersistKv: map[string]*Persist{
					"test": {
						Type:        "other",
						InfraParams: map[string]interface{}{"key2": "value200"},
					},
				},
			},
			id: "test",
			want: Persist{
				Type: "other",
				InfraParams: InfraParams{
					"key2": "value200",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			testcfg := tt.cfg.GetPersistKv(tt.id)
			assert.Equal(tt.want, testcfg)

		})
	}
}

func Test_GetPersistFs(t *testing.T) {
	tests := []struct {
		name string
		cfg  Application
		id   string
		want Persist
	}{
		{
			name: "get base fs",
			cfg: Application{
				Defaults: Defaults{
					PersistFs: KindDefaults{
						Type: "dynamodb",
						InfraParamsByType: map[string]InfraParams{
							"dynamodb": {"key1": "value1"},
						},
					},
				},
			},
			id: "test",
			want: Persist{
				Type: "dynamodb",
				InfraParams: InfraParams{
					"key1": "value1",
				},
			},
		},
		{
			name: "get base type params exec unit",
			cfg: Application{
				Defaults: Defaults{
					PersistFs: KindDefaults{
						Type: "dynamodb",
						InfraParamsByType: map[string]InfraParams{
							"dynamodb": {"key1": "value1"},
						},
					},
				},
				PersistFs: map[string]*Persist{
					"test": {Type: "dynamodb"},
				},
			},
			id: "test",
			want: Persist{
				Type: "dynamodb",
				InfraParams: InfraParams{
					"key1": "value1",
				},
			},
		},
		{
			name: "get config and add other default params",
			cfg: Application{
				Defaults: Defaults{
					PersistFs: KindDefaults{
						Type: "dynamodb",
						InfraParamsByType: map[string]InfraParams{
							"dynamodb": {"key1": "value1", "key2": "value2"},
						},
					},
				},
				PersistFs: map[string]*Persist{
					"test": {
						Type:        "dynamodb",
						InfraParams: map[string]interface{}{"key2": "value200"},
					},
				},
			},
			id: "test",
			want: Persist{
				Type: "dynamodb",
				InfraParams: InfraParams{
					"key1": "value1",
					"key2": "value200",
				},
			},
		},
		{
			name: "get config and with no default params",
			cfg: Application{
				Defaults: Defaults{
					PersistFs: KindDefaults{
						Type: "dynamodb",
						InfraParamsByType: map[string]InfraParams{
							"dynamodb": {"key1": "value1", "key2": "value2"},
						},
					},
				},
				PersistFs: map[string]*Persist{
					"test": {
						Type:        "other",
						InfraParams: map[string]interface{}{"key2": "value200"},
					},
				},
			},
			id: "test",
			want: Persist{
				Type: "other",
				InfraParams: InfraParams{
					"key2": "value200",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			testcfg := tt.cfg.GetPersistFs(tt.id)
			assert.Equal(tt.want, testcfg)

		})
	}
}

func Test_GetPersistSecrets(t *testing.T) {
	tests := []struct {
		name string
		cfg  Application
		id   string
		want Persist
	}{
		{
			name: "get base secrets",
			cfg: Application{
				Defaults: Defaults{
					PersistSecrets: KindDefaults{
						Type: "dynamodb",
						InfraParamsByType: map[string]InfraParams{
							"dynamodb": {"key1": "value1"},
						},
					},
				},
			},
			id: "test",
			want: Persist{
				Type: "dynamodb",
				InfraParams: InfraParams{
					"key1": "value1",
				},
			},
		},
		{
			name: "get base type params",
			cfg: Application{
				Defaults: Defaults{
					PersistSecrets: KindDefaults{
						Type: "dynamodb",
						InfraParamsByType: map[string]InfraParams{
							"dynamodb": {"key1": "value1"},
						},
					},
				},
				PersistSecrets: map[string]*Persist{
					"test": {Type: "dynamodb"},
				},
			},
			id: "test",
			want: Persist{
				Type: "dynamodb",
				InfraParams: InfraParams{
					"key1": "value1",
				},
			},
		},
		{
			name: "get config and add other default params",
			cfg: Application{
				Defaults: Defaults{
					PersistSecrets: KindDefaults{
						Type: "dynamodb",
						InfraParamsByType: map[string]InfraParams{
							"dynamodb": {"key1": "value1", "key2": "value2"},
						},
					},
				},
				PersistSecrets: map[string]*Persist{
					"test": {
						Type:        "dynamodb",
						InfraParams: map[string]interface{}{"key2": "value200"},
					},
				},
			},
			id: "test",
			want: Persist{
				Type: "dynamodb",
				InfraParams: InfraParams{
					"key1": "value1",
					"key2": "value200",
				},
			},
		},
		{
			name: "get config and with no default params",
			cfg: Application{
				Defaults: Defaults{
					PersistSecrets: KindDefaults{
						Type: "dynamodb",
						InfraParamsByType: map[string]InfraParams{
							"dynamodb": {"key1": "value1", "key2": "value2"},
						},
					},
				},
				PersistSecrets: map[string]*Persist{
					"test": {
						Type:        "other",
						InfraParams: map[string]interface{}{"key2": "value200"},
					},
				},
			},
			id: "test",
			want: Persist{
				Type: "other",
				InfraParams: InfraParams{
					"key2": "value200",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			testcfg := tt.cfg.GetPersistSecrets(tt.id)
			assert.Equal(tt.want, testcfg)

		})
	}
}

func Test_GetPersistOrm(t *testing.T) {
	tests := []struct {
		name string
		cfg  Application
		id   string
		want Persist
	}{
		{
			name: "get base secrets",
			cfg: Application{
				Defaults: Defaults{
					PersistOrm: KindDefaults{
						Type: "dynamodb",
						InfraParamsByType: map[string]InfraParams{
							"dynamodb": {"key1": "value1"},
						},
					},
				},
			},
			id: "test",
			want: Persist{
				Type: "dynamodb",
				InfraParams: InfraParams{
					"key1": "value1",
				},
			},
		},
		{
			name: "get base type params",
			cfg: Application{
				Defaults: Defaults{
					PersistOrm: KindDefaults{
						Type: "dynamodb",
						InfraParamsByType: map[string]InfraParams{
							"dynamodb": {"key1": "value1"},
						},
					},
				},
				PersistOrm: map[string]*Persist{
					"test": {Type: "dynamodb"},
				},
			},
			id: "test",
			want: Persist{
				Type: "dynamodb",
				InfraParams: InfraParams{
					"key1": "value1",
				},
			},
		},
		{
			name: "get config and add other default params",
			cfg: Application{
				Defaults: Defaults{
					PersistOrm: KindDefaults{
						Type: "dynamodb",
						InfraParamsByType: map[string]InfraParams{
							"dynamodb": {"key1": "value1", "key2": "value2"},
						},
					},
				},
				PersistOrm: map[string]*Persist{
					"test": {
						Type:        "dynamodb",
						InfraParams: map[string]interface{}{"key2": "value200"},
					},
				},
			},
			id: "test",
			want: Persist{
				Type: "dynamodb",
				InfraParams: InfraParams{
					"key1": "value1",
					"key2": "value200",
				},
			},
		},
		{
			name: "get config and with no default params",
			cfg: Application{
				Defaults: Defaults{
					PersistOrm: KindDefaults{
						Type: "dynamodb",
						InfraParamsByType: map[string]InfraParams{
							"dynamodb": {"key1": "value1", "key2": "value2"},
						},
					},
				},
				PersistOrm: map[string]*Persist{
					"test": {
						Type:        "other",
						InfraParams: map[string]interface{}{"key2": "value200"},
					},
				},
			},
			id: "test",
			want: Persist{
				Type: "other",
				InfraParams: InfraParams{
					"key2": "value200",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			testcfg := tt.cfg.GetPersistOrm(tt.id)
			assert.Equal(tt.want, testcfg)

		})
	}
}

func Test_GetPersistRedisNode(t *testing.T) {
	tests := []struct {
		name string
		cfg  Application
		id   string
		want Persist
	}{
		{
			name: "get base secrets",
			cfg: Application{
				Defaults: Defaults{
					PersistRedisNode: KindDefaults{
						Type: "dynamodb",
						InfraParamsByType: map[string]InfraParams{
							"dynamodb": {"key1": "value1"},
						},
					},
				},
			},
			id: "test",
			want: Persist{
				Type: "dynamodb",
				InfraParams: InfraParams{
					"key1": "value1",
				},
			},
		},
		{
			name: "get base type params",
			cfg: Application{
				Defaults: Defaults{
					PersistRedisNode: KindDefaults{
						Type: "dynamodb",
						InfraParamsByType: map[string]InfraParams{
							"dynamodb": {"key1": "value1"},
						},
					},
				},
				PersistRedisNode: map[string]*Persist{
					"test": {Type: "dynamodb"},
				},
			},
			id: "test",
			want: Persist{
				Type: "dynamodb",
				InfraParams: InfraParams{
					"key1": "value1",
				},
			},
		},
		{
			name: "get config and add other default params",
			cfg: Application{
				Defaults: Defaults{
					PersistRedisNode: KindDefaults{
						Type: "dynamodb",
						InfraParamsByType: map[string]InfraParams{
							"dynamodb": {"key1": "value1", "key2": "value2"},
						},
					},
				},
				PersistRedisNode: map[string]*Persist{
					"test": {
						Type:        "dynamodb",
						InfraParams: map[string]interface{}{"key2": "value200"},
					},
				},
			},
			id: "test",
			want: Persist{
				Type: "dynamodb",
				InfraParams: InfraParams{
					"key1": "value1",
					"key2": "value200",
				},
			},
		},
		{
			name: "get config and with no default params",
			cfg: Application{
				Defaults: Defaults{
					PersistRedisNode: KindDefaults{
						Type: "dynamodb",
						InfraParamsByType: map[string]InfraParams{
							"dynamodb": {"key1": "value1", "key2": "value2"},
						},
					},
				},
				PersistRedisNode: map[string]*Persist{
					"test": {
						Type:        "other",
						InfraParams: map[string]interface{}{"key2": "value200"},
					},
				},
			},
			id: "test",
			want: Persist{
				Type: "other",
				InfraParams: InfraParams{
					"key2": "value200",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			testcfg := tt.cfg.GetPersistRedisNode(tt.id)
			assert.Equal(tt.want, testcfg)

		})
	}
}

func Test_GetPersistRedisCluster(t *testing.T) {
	tests := []struct {
		name string
		cfg  Application
		id   string
		want Persist
	}{
		{
			name: "get base secrets",
			cfg: Application{
				Defaults: Defaults{
					PersistRedisCluster: KindDefaults{
						Type: "dynamodb",
						InfraParamsByType: map[string]InfraParams{
							"dynamodb": {"key1": "value1"},
						},
					},
				},
			},
			id: "test",
			want: Persist{
				Type: "dynamodb",
				InfraParams: InfraParams{
					"key1": "value1",
				},
			},
		},
		{
			name: "get base type params",
			cfg: Application{
				Defaults: Defaults{
					PersistRedisCluster: KindDefaults{
						Type: "dynamodb",
						InfraParamsByType: map[string]InfraParams{
							"dynamodb": {"key1": "value1"},
						},
					},
				},
				PersistRedisCluster: map[string]*Persist{
					"test": {Type: "dynamodb"},
				},
			},
			id: "test",
			want: Persist{
				Type: "dynamodb",
				InfraParams: InfraParams{
					"key1": "value1",
				},
			},
		},
		{
			name: "get config and add other default params",
			cfg: Application{
				Defaults: Defaults{
					PersistRedisCluster: KindDefaults{
						Type: "dynamodb",
						InfraParamsByType: map[string]InfraParams{
							"dynamodb": {"key1": "value1", "key2": "value2"},
						},
					},
				},
				PersistRedisCluster: map[string]*Persist{
					"test": {
						Type:        "dynamodb",
						InfraParams: map[string]interface{}{"key2": "value200"},
					},
				},
			},
			id: "test",
			want: Persist{
				Type: "dynamodb",
				InfraParams: InfraParams{
					"key1": "value1",
					"key2": "value200",
				},
			},
		},
		{
			name: "get config and with no default params",
			cfg: Application{
				Defaults: Defaults{
					PersistRedisCluster: KindDefaults{
						Type: "dynamodb",
						InfraParamsByType: map[string]InfraParams{
							"dynamodb": {"key1": "value1", "key2": "value2"},
						},
					},
				},
				PersistRedisCluster: map[string]*Persist{
					"test": {
						Type:        "other",
						InfraParams: map[string]interface{}{"key2": "value200"},
					},
				},
			},
			id: "test",
			want: Persist{
				Type: "other",
				InfraParams: InfraParams{
					"key2": "value200",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			testcfg := tt.cfg.GetPersistRedisCluster(tt.id)
			assert.Equal(tt.want, testcfg)

		})
	}
}
