package execunit

import (
	"strings"
	"testing"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/stretchr/testify/assert"
)

func Test_environmentVarsAddedToUnit(t *testing.T) {
	tests := []struct {
		name         string
		envVars      map[string]string
		want         core.EnvironmentVariables
		wantExecUnit bool
	}{
		{
			name: "no exec unit",
			envVars: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			wantExecUnit: false,
		},
		{
			name: "add env vars",
			envVars: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			want: core.EnvironmentVariables{
				{
					Name:  "key1",
					Value: "value1",
				},
				{
					Name:  "key2",
					Value: "value2",
				},
			},
			wantExecUnit: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			cfg := config.Application{
				ExecutionUnits: map[string]*config.ExecutionUnit{
					"main": {EnvironmentVariables: tt.envVars},
				},
			}
			p := ExecUnitPlugin{Config: &cfg}
			result := core.NewConstructGraph()

			inputFiles := &core.InputFiles{}
			if tt.wantExecUnit {
				f, err := core.NewSourceFile("test", strings.NewReader("test"), testAnnotationLang)
				if assert.Nil(err) {
					inputFiles.Add(f)
				}
			} else {
				inputFiles.Add(&core.FileRef{
					FPath: "test",
				})
			}

			err := p.Transform(inputFiles, &core.FileDependencies{}, result)
			if !assert.NoError(err) {
				return
			}
			units := core.GetResourcesOfType[*core.ExecutionUnit](result)
			if tt.wantExecUnit {
				assert.Len(units, 1)
				assert.ElementsMatch(tt.want, units[0].EnvironmentVariables)
			} else {
				assert.Len(units, 0)
			}

		})
	}
}
