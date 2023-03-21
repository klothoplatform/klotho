package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_GetExpose(t *testing.T) {
	tests := []struct {
		name string
		cfg  Application
		id   string
		want Expose
	}{
		{
			name: "get base exec unit",
			cfg: Application{
				Defaults: Defaults{
					Expose: KindDefaults{
						Type: "apigateway",
						InfraParamsByType: map[string]InfraParams{
							"apigateway": {"key1": "value1"},
						},
					},
				},
			},
			id: "test",
			want: Expose{
				Type: "apigateway",
				InfraParams: InfraParams{
					"key1": "value1",
				},
			},
		},
		{
			name: "get base type params exec unit",
			cfg: Application{
				Defaults: Defaults{
					Expose: KindDefaults{
						Type: "apigateway",
						InfraParamsByType: map[string]InfraParams{
							"apigateway": {"key1": "value1"},
						},
					},
				},
				Exposed: map[string]*Expose{
					"test": {Type: "apigateway"},
				},
			},
			id: "test",
			want: Expose{
				Type: "apigateway",
				InfraParams: InfraParams{
					"key1": "value1",
				},
			},
		},
		{
			name: "get config and add other default params",
			cfg: Application{
				Defaults: Defaults{
					Expose: KindDefaults{
						Type: "apigateway",
						InfraParamsByType: map[string]InfraParams{
							"apigateway": {"key1": "value1", "key2": "value2"},
						},
					},
				},
				Exposed: map[string]*Expose{
					"test": {
						Type:        "apigateway",
						InfraParams: map[string]interface{}{"key2": "value200"},
					},
				},
			},
			id: "test",
			want: Expose{
				Type: "apigateway",
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
					Expose: KindDefaults{
						Type: "apigateway",
						InfraParamsByType: map[string]InfraParams{
							"apigateway": {"key1": "value1", "key2": "value2"},
						},
					},
				},
				Exposed: map[string]*Expose{
					"test": {
						Type:        "alb",
						InfraParams: map[string]interface{}{"key2": "value200"},
					},
				},
			},
			id: "test",
			want: Expose{
				Type: "alb",
				InfraParams: InfraParams{
					"key2": "value200",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			testcfg := tt.cfg.GetExpose(tt.id)
			assert.Equal(tt.want, testcfg)

		})
	}
}
