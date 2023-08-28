package execunit

import (
	"strings"
	"testing"

	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/io"
	"github.com/stretchr/testify/assert"
)

func Test_environmentVarsAddedToUnit(t *testing.T) {
	tests := []struct {
		name         string
		envVars      map[string]string
		want         types.EnvironmentVariables
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
			want: types.EnvironmentVariables{
				{
					Name:  "key1",
					Value: "value1",
				},
				{
					Name:  "key2",
					Value: "value2",
				},
				{
					Name:  "APP_NAME",
					Value: "test",
				},
				{
					Name:  "EXECUNIT_NAME",
					Value: "main",
				},
			},
			wantExecUnit: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			cfg := config.Application{
				AppName: "test",
				ExecutionUnits: map[string]*config.ExecutionUnit{
					"main": {EnvironmentVariables: tt.envVars},
				},
			}
			p := ExecUnitPlugin{Config: &cfg}
			result := construct.NewConstructGraph()

			inputFiles := &types.InputFiles{}
			if tt.wantExecUnit {
				f, err := types.NewSourceFile("test", strings.NewReader("test"), testAnnotationLang)
				if assert.Nil(err) {
					inputFiles.Add(f)
				}
			} else {
				inputFiles.Add(&io.FileRef{
					FPath: "test",
				})
			}

			err := p.Transform(inputFiles, &types.FileDependencies{}, result)
			if !assert.NoError(err) {
				return
			}
			units := construct.GetConstructsOfType[*types.ExecutionUnit](result)
			if tt.wantExecUnit {
				assert.Len(units, 1)
				assert.ElementsMatch(tt.want, units[0].EnvironmentVariables)
			} else {
				assert.Len(units, 0)
			}

		})
	}
}
