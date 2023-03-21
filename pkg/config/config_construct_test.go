package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_GetConfig(t *testing.T) {
	tests := []struct {
		name string
		cfg  Application
		id   string
		want Config
	}{
		{
			name: "get base config",
			cfg: Application{
				Defaults: Defaults{
					Config: KindDefaults{
						Type: "s3",
						InfraParamsByType: map[string]InfraParams{
							"s3": {"key1": "value1"},
						},
					},
				},
			},
			id: "test",
			want: Config{
				Type: "s3",
				InfraParams: InfraParams{
					"key1": "value1",
				},
			},
		},
		{
			name: "get base type params",
			cfg: Application{
				Defaults: Defaults{
					Config: KindDefaults{
						Type: "s3",
						InfraParamsByType: map[string]InfraParams{
							"s3": {"key1": "value1"},
						},
					},
				},
				Config: map[string]*Config{
					"test": {Type: "s3"},
				},
			},
			id: "test",
			want: Config{
				Type: "s3",
				InfraParams: InfraParams{
					"key1": "value1",
				},
			},
		},
		{
			name: "get config and add other default params",
			cfg: Application{
				Defaults: Defaults{
					Config: KindDefaults{
						Type: "s3",
						InfraParamsByType: map[string]InfraParams{
							"s3": {"key1": "value1", "key2": "value2"},
						},
					},
				},
				Config: map[string]*Config{
					"test": {
						Type:        "s3",
						InfraParams: map[string]interface{}{"key2": "value200"},
					},
				},
			},
			id: "test",
			want: Config{
				Type: "s3",
				InfraParams: InfraParams{
					"key1": "value1",
					"key2": "value200",
				},
			},
		},
		{
			name: "get config and no default params",
			cfg: Application{
				Defaults: Defaults{
					Config: KindDefaults{
						Type: "s3",
						InfraParamsByType: map[string]InfraParams{
							"s3": {"key1": "value1", "key2": "value2"},
						},
					},
				},
				Config: map[string]*Config{
					"test": {
						Type:        "secrets_manager",
						InfraParams: map[string]interface{}{"key2": "value200"},
					},
				},
			},
			id: "test",
			want: Config{
				Type: "secrets_manager",
				InfraParams: InfraParams{
					"key2": "value200",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			testcfg := tt.cfg.GetConfig(tt.id)
			assert.Equal(tt.want, testcfg)

		})
	}
}
