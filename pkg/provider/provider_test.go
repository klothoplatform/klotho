package provider

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/stretchr/testify/assert"
)

func Test_validation_handleProviderValidation(t *testing.T) {
	tests := []struct {
		name    string
		result  []core.Construct
		cfg     config.Application
		wantErr bool
	}{
		{
			name: "exec unit match",
			result: []core.Construct{&core.ExecutionUnit{
				Name: "test",
			}},
			cfg: config.Application{
				Provider: "aws",
				ExecutionUnits: map[string]*config.ExecutionUnit{
					"test": {
						Type: "lambda",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "exec unit mismatch",
			result: []core.Construct{&core.ExecutionUnit{
				Name: "test",
			}},
			cfg: config.Application{
				Provider: "aws",
				ExecutionUnits: map[string]*config.ExecutionUnit{
					"test": {
						Type: "wrong",
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			provider := TestProvider{}
			result := core.NewConstructGraph()
			for _, c := range tt.result {
				result.AddConstruct(c)
			}

			err := HandleProviderValidation(provider, &tt.cfg, result)
			if tt.wantErr {
				assert.Error(err)
				return
			} else {
				assert.NoError(err)
				return
			}
		})
	}
}

type TestProvider struct {
}

func (p TestProvider) Translate(result *core.ConstructGraph, dag *core.ResourceGraph) error {
	return nil
}
func (p TestProvider) Name() string { return "test" }
func (p TestProvider) Validate(config *config.Application, constructGraph *core.ConstructGraph) error {
	return HandleProviderValidation(p, config, constructGraph)
}
func (p TestProvider) GetDefaultConfig() config.Defaults {
	return config.Defaults{}

}
func (p TestProvider) LoadGraph(graph core.OutputGraph, dag *core.ConstructGraph) error {
	return nil
}
func (p TestProvider) CreateResourceFromId(id core.ResourceId, dag *core.ResourceGraph) (core.Resource, error) {
	return nil, nil
}
func (p TestProvider) ExpandConstruct(construct core.Construct, dag *core.ResourceGraph) ([]core.Resource, error) {
	return nil, nil
}
func (a TestProvider) GetKindTypeMappings(construct core.Construct) []string {
	switch construct.(type) {
	case *core.ExecutionUnit:
		return []string{"lambda"}
	}
	return nil
}
