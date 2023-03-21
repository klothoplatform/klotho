package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_GetStaticUnit(t *testing.T) {
	tests := []struct {
		name string
		cfg  Application
		id   string
		want StaticUnit
	}{
		{
			name: "get base fs",
			cfg: Application{
				Defaults: Defaults{
					StaticUnit: KindDefaults{
						Type: "dynamodb",
						InfraParamsByType: map[string]InfraParams{
							"dynamodb": {"key1": "value1"},
						},
					},
				},
			},
			id: "test",
			want: StaticUnit{
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
					StaticUnit: KindDefaults{
						Type: "dynamodb",
						InfraParamsByType: map[string]InfraParams{
							"dynamodb": {"key1": "value1"},
						},
					},
				},
				StaticUnit: map[string]*StaticUnit{
					"test": {Type: "dynamodb"},
				},
			},
			id: "test",
			want: StaticUnit{
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
					StaticUnit: KindDefaults{
						Type: "dynamodb",
						InfraParamsByType: map[string]InfraParams{
							"dynamodb": {"key1": "value1", "key2": "value2"},
						},
					},
				},
				StaticUnit: map[string]*StaticUnit{
					"test": {
						Type:        "dynamodb",
						InfraParams: map[string]interface{}{"key2": "value200"},
					},
				},
			},
			id: "test",
			want: StaticUnit{
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
					StaticUnit: KindDefaults{
						Type: "dynamodb",
						InfraParamsByType: map[string]InfraParams{
							"dynamodb": {"key1": "value1", "key2": "value2"},
						},
					},
				},
				StaticUnit: map[string]*StaticUnit{
					"test": {
						Type:        "other",
						InfraParams: map[string]interface{}{"key2": "value200"},
					},
				},
			},
			id: "test",
			want: StaticUnit{
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

			testcfg := tt.cfg.GetStaticUnit(tt.id)
			assert.Equal(tt.want, testcfg)

		})
	}
}
