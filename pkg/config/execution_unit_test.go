package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_GetExecutionUnit(t *testing.T) {
	tests := []struct {
		name string
		cfg  Application
		id   string
		want ExecutionUnit
	}{
		{
			name: "get base exec unit",
			cfg: Application{
				Defaults: Defaults{
					ExecutionUnit: KindDefaults{
						Type: "lambda",
						InfraParamsByType: map[string]InfraParams{
							"lambda": {"key1": "value1"},
						},
					},
				},
			},
			id: "test",
			want: ExecutionUnit{
				Type: "lambda",
				InfraParams: InfraParams{
					"key1": "value1",
				},
				NetworkPlacement:     "private",
				EnvironmentVariables: make(map[string]string),
			},
		},
		{
			name: "get base type params exec unit",
			cfg: Application{
				Defaults: Defaults{
					ExecutionUnit: KindDefaults{
						Type: "lambda",
						InfraParamsByType: map[string]InfraParams{
							"lambda": {"key1": "value1"},
						},
					},
				},
				ExecutionUnits: map[string]*ExecutionUnit{
					"test": {Type: "lambda"},
				},
			},
			id: "test",
			want: ExecutionUnit{
				Type: "lambda",
				InfraParams: InfraParams{
					"key1": "value1",
				},
				NetworkPlacement:     "private",
				EnvironmentVariables: make(map[string]string),
				HelmChartOptions:     &HelmChartOptions{},
			},
		},
		{
			name: "get config and add other default params",
			cfg: Application{
				Defaults: Defaults{
					ExecutionUnit: KindDefaults{
						Type: "lambda",
						InfraParamsByType: map[string]InfraParams{
							"lambda": {"key1": "value1", "key2": "value2"},
						},
					},
				},
				ExecutionUnits: map[string]*ExecutionUnit{
					"test": {
						Type:        "lambda",
						InfraParams: map[string]interface{}{"key2": "value200"},
					},
				},
			},
			id: "test",
			want: ExecutionUnit{
				Type: "lambda",
				InfraParams: InfraParams{
					"key1": "value1",
					"key2": "value200",
				},
				NetworkPlacement:     "private",
				EnvironmentVariables: make(map[string]string),
				HelmChartOptions:     &HelmChartOptions{},
			},
		},
		{
			name: "get config and add other default params plus overrides",
			cfg: Application{
				Defaults: Defaults{
					ExecutionUnit: KindDefaults{
						Type: "lambda",
						InfraParamsByType: map[string]InfraParams{
							"lambda": {"key1": "value1", "key2": "value2"},
						},
					},
				},
				ExecutionUnits: map[string]*ExecutionUnit{
					"test": {
						Type:             "ecs",
						InfraParams:      map[string]interface{}{"key2": "value200"},
						NetworkPlacement: "public",
						EnvironmentVariables: map[string]string{
							"1": "2",
						},
					},
				},
			},
			id: "test",
			want: ExecutionUnit{
				Type: "ecs",
				InfraParams: InfraParams{
					"key2": "value200",
				},
				NetworkPlacement: "public",
				EnvironmentVariables: map[string]string{
					"1": "2",
				},
				HelmChartOptions: &HelmChartOptions{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			testcfg := tt.cfg.GetExecutionUnit(tt.id)
			assert.Equal(tt.want, testcfg)

		})
	}
}
