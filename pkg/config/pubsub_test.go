package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_GetPubsub(t *testing.T) {
	tests := []struct {
		name string
		cfg  Application
		id   string
		want PubSub
	}{
		{
			name: "get base exec unit",
			cfg: Application{
				Defaults: Defaults{
					PubSub: KindDefaults{
						Type: "sns",
						InfraParamsByType: map[string]InfraParams{
							"sns": {"key1": "value1"},
						},
					},
				},
			},
			id: "test",
			want: PubSub{
				Type: "sns",
				InfraParams: InfraParams{
					"key1": "value1",
				},
			},
		},
		{
			name: "get base type params exec unit",
			cfg: Application{
				Defaults: Defaults{
					PubSub: KindDefaults{
						Type: "sns",
						InfraParamsByType: map[string]InfraParams{
							"sns": {"key1": "value1"},
						},
					},
				},
				PubSub: map[string]*PubSub{
					"test": {Type: "sns"},
				},
			},
			id: "test",
			want: PubSub{
				Type: "sns",
				InfraParams: InfraParams{
					"key1": "value1",
				},
			},
		},
		{
			name: "get config and add other default params",
			cfg: Application{
				Defaults: Defaults{
					PubSub: KindDefaults{
						Type: "sns",
						InfraParamsByType: map[string]InfraParams{
							"sns": {"key1": "value1", "key2": "value2"},
						},
					},
				},
				PubSub: map[string]*PubSub{
					"test": {
						Type:        "sns",
						InfraParams: map[string]interface{}{"key2": "value200"},
					},
				},
			},
			id: "test",
			want: PubSub{
				Type: "sns",
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
					PubSub: KindDefaults{
						Type: "sns",
						InfraParamsByType: map[string]InfraParams{
							"sns": {"key1": "value1", "key2": "value2"},
						},
					},
				},
				PubSub: map[string]*PubSub{
					"test": {
						Type:        "sqs",
						InfraParams: map[string]interface{}{"key2": "value200"},
					},
				},
			},
			id: "test",
			want: PubSub{
				Type: "sqs",
				InfraParams: InfraParams{
					"key2": "value200",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			testcfg := tt.cfg.GetPubSub(tt.id)
			assert.Equal(tt.want, testcfg)

		})
	}
}
