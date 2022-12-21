package validation

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	execunit "github.com/klothoplatform/klotho/pkg/exec_unit"
	"github.com/klothoplatform/klotho/pkg/provider/providers"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func Test_validation_checkAnnotationForResource(t *testing.T) {
	tests := []struct {
		name    string
		annot   core.Annotation
		want    core.ResourceKey
		result  []core.CloudResource
		wantErr bool
	}{
		{
			name: "exec unit match",
			annot: core.Annotation{
				Capability: &annotation.Capability{Name: annotation.ExecutionUnitCapability, ID: "test"},
			},
			result: []core.CloudResource{&core.ExecutionUnit{
				Name: "test",
			}},
			want: core.ResourceKey{Name: "test", Kind: "exec_unit"},
		},
		{
			name: "pubsub match",
			annot: core.Annotation{
				Capability: &annotation.Capability{Name: annotation.PubSubCapability, ID: "test"},
			},
			result: []core.CloudResource{&core.PubSub{
				Name: "test",
			}},
			want: core.ResourceKey{Name: "test", Kind: "pubsub"},
		},
		{
			name: "gateway match",
			annot: core.Annotation{
				Capability: &annotation.Capability{Name: annotation.ExposeCapability, ID: "test"},
			},
			result: []core.CloudResource{&core.Gateway{
				Name: "test",
			}},
			want: core.ResourceKey{Name: "test", Kind: "gateway"},
		},
		{
			name: "embed assets match",
			annot: core.Annotation{
				Capability: &annotation.Capability{Name: annotation.AssetCapability, ID: "test"},
			},
			result: []core.CloudResource{},
		},
		{
			name: "persist fs match",
			annot: core.Annotation{
				Capability: &annotation.Capability{Name: annotation.PersistCapability, ID: "test"},
			},
			result: []core.CloudResource{&core.Persist{
				Name: "test",
				Kind: core.PersistFileKind,
			}},
			want: core.ResourceKey{Name: "test", Kind: "persist_fs"},
		},
		{
			name: "persist kv match",
			annot: core.Annotation{
				Capability: &annotation.Capability{Name: annotation.PersistCapability, ID: "test"},
			},
			result: []core.CloudResource{&core.Persist{
				Name: "test",
				Kind: core.PersistFileKind,
			}},
			want: core.ResourceKey{Name: "test", Kind: "persist_fs"},
		},
		{
			name: "persist fs match",
			annot: core.Annotation{
				Capability: &annotation.Capability{Name: annotation.PersistCapability, ID: "test"},
			},
			result: []core.CloudResource{&core.Persist{
				Name: "test",
				Kind: core.PersistKVKind,
			}},
			want: core.ResourceKey{Name: "test", Kind: "persist_kv"},
		},
		{
			name: "persist orm match",
			annot: core.Annotation{
				Capability: &annotation.Capability{Name: annotation.PersistCapability, ID: "test"},
			},
			result: []core.CloudResource{&core.Persist{
				Name: "test",
				Kind: core.PersistORMKind,
			}},
			want: core.ResourceKey{Name: "test", Kind: "persist_orm"},
		},
		{
			name: "persist redis cluster match",
			annot: core.Annotation{
				Capability: &annotation.Capability{Name: annotation.PersistCapability, ID: "test"},
			},
			result: []core.CloudResource{&core.Persist{
				Name: "test",
				Kind: core.PersistRedisClusterKind,
			}},
			want: core.ResourceKey{Name: "test", Kind: "persist_redis_cluster"},
		},
		{
			name: "persist redis node match",
			annot: core.Annotation{
				Capability: &annotation.Capability{Name: annotation.PersistCapability, ID: "test"},
			},
			result: []core.CloudResource{&core.Persist{
				Name: "test",
				Kind: core.PersistRedisNodeKind,
			}},
			want: core.ResourceKey{Name: "test", Kind: "persist_redis_node"},
		},
		{
			name: "persist secret match",
			annot: core.Annotation{
				Capability: &annotation.Capability{Name: annotation.PersistCapability, ID: "test"},
			},
			result: []core.CloudResource{&core.Persist{
				Name: "test",
				Kind: core.PersistSecretKind,
			}},
			want: core.ResourceKey{Name: "test", Kind: "persist_secret"},
		},
		{
			name: "persist secret match",
			annot: core.Annotation{
				Capability: &annotation.Capability{Name: annotation.PersistCapability, ID: "test"},
			},
			result: []core.CloudResource{&core.Persist{
				Name: "test",
				Kind: core.PersistSecretKind,
			}},
			want: core.ResourceKey{Name: "test", Kind: "persist_secret"},
		},
		{
			name: "no match on capability should return empty resource",
			annot: core.Annotation{
				Capability: &annotation.Capability{Name: annotation.ExecutionUnitCapability, ID: "test"},
			},
			result: []core.CloudResource{&core.Persist{
				Name: "test",
				Kind: core.PersistSecretKind,
			}},
			want: core.ResourceKey{},
		},
		{
			name: "no match on id should return empty resource",
			annot: core.Annotation{
				Capability: &annotation.Capability{Name: annotation.PersistCapability, ID: "test"},
			},
			result: []core.CloudResource{&core.Persist{
				Name: "notTest",
				Kind: core.PersistSecretKind,
			}},
			want: core.ResourceKey{},
		},
		{
			name: "only one match should succeed",
			annot: core.Annotation{
				Capability: &annotation.Capability{Name: annotation.PersistCapability, ID: "test"},
			},
			result: []core.CloudResource{&core.Persist{
				Name: "notTest",
				Kind: core.PersistSecretKind,
			},
				&core.Persist{
					Name: "test",
					Kind: core.PersistSecretKind,
				}},
			want: core.ResourceKey{Name: "test", Kind: "persist_secret"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			p := Plugin{}
			result := core.CompilationResult{}
			result.AddAll(tt.result)
			log := zap.L().With().Sugar()

			resource := p.checkAnnotationForResource(&tt.annot, &result, log)
			assert.Equal(tt.want, resource)

		})
	}
}

func Test_validation_handleProviderValidation(t *testing.T) {
	tests := []struct {
		name    string
		result  []core.CloudResource
		cfg     config.Application
		wantErr bool
	}{
		{
			name: "exec unit match",
			result: []core.CloudResource{&core.ExecutionUnit{
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
			result: []core.CloudResource{&core.ExecutionUnit{
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
		{
			name:   "input files kind",
			result: []core.CloudResource{&core.InputFiles{}},
			cfg: config.Application{
				Provider: "aws",
			},
			wantErr: false,
		},
		{
			name:   "input file deps kind",
			result: []core.CloudResource{&execunit.FileDependencies{}},
			cfg: config.Application{
				Provider: "aws",
			},
			wantErr: false,
		},
		{
			name:   "topology kind",
			result: []core.CloudResource{&core.Topology{}},
			cfg: config.Application{
				Provider: "aws",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			provider, _ := providers.GetProvider(&tt.cfg)
			p := Plugin{
				Provider: provider,
				Config:   &tt.cfg,
			}
			result := core.CompilationResult{}
			result.AddAll(tt.result)

			err := p.handleProviderValidation(&result)
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

func Test_validateConfigOverrideResourcesExist(t *testing.T) {
	tests := []struct {
		name   string
		result []core.CloudResource
		cfg    config.Application
		want   string
	}{
		{
			name: "exec unit match",
			result: []core.CloudResource{&core.ExecutionUnit{
				Name: "test",
			}},
			cfg: config.Application{
				Provider: "aws",
				ExecutionUnits: map[string]*config.ExecutionUnit{
					"test": {},
				},
			},
		},
		{
			name: "exec unit mismatch",
			result: []core.CloudResource{&core.ExecutionUnit{
				Name: "test",
			}},
			cfg: config.Application{
				Provider: "aws",
				ExecutionUnits: map[string]*config.ExecutionUnit{
					"nottest": {},
				},
			},
			want: `Unknown execution unit in config override, "nottest".`,
		},
		{
			name: "persist match",
			result: []core.CloudResource{&core.Persist{
				Name: "test",
				Kind: core.PersistKVKind,
			}},
			cfg: config.Application{
				Provider: "aws",
				Persisted: map[string]*config.Persist{
					"test": {},
				},
			},
		},
		{
			name: "persist mismatch",
			result: []core.CloudResource{&core.Persist{
				Name: "test",
			}},
			cfg: config.Application{
				Provider: "aws",
				Persisted: map[string]*config.Persist{
					"nottest": {},
				},
			},
			want: `Unknown persist in config override, "nottest".`,
		},
		{
			name: "expose match",
			result: []core.CloudResource{&core.Gateway{
				Name: "test",
			}},
			cfg: config.Application{
				Provider: "aws",
				Exposed: map[string]*config.Expose{
					"test": {},
				},
			},
		},
		{
			name: "expose mismatch",
			result: []core.CloudResource{&core.Gateway{
				Name: "test",
			}},
			cfg: config.Application{
				Provider: "aws",
				Exposed: map[string]*config.Expose{
					"nottest": {},
				},
			},
			want: `Unknown expose in config override, "nottest".`,
		},
		{
			name: "pubsub match",
			result: []core.CloudResource{&core.PubSub{
				Name: "test",
			}},
			cfg: config.Application{
				Provider: "aws",
				PubSub: map[string]*config.PubSub{
					"test": {},
				},
			},
		},
		{
			name: "pubsub mismatch",
			result: []core.CloudResource{&core.PubSub{
				Name: "test",
			}},
			cfg: config.Application{
				Provider: "aws",
				PubSub: map[string]*config.PubSub{
					"nottest": {},
				},
			},
			want: `Unknown pubsub in config override, "nottest".`,
		},
		{
			name: "static unit match",
			result: []core.CloudResource{&core.StaticUnit{
				Name: "test",
			}},
			cfg: config.Application{
				Provider: "aws",
				StaticUnit: map[string]*config.StaticUnit{
					"test": {},
				},
			},
		},
		{
			name: "static unit mismatch",
			result: []core.CloudResource{&core.StaticUnit{
				Name: "test",
			}},
			cfg: config.Application{
				Provider: "aws",
				StaticUnit: map[string]*config.StaticUnit{
					"nottest": {},
				},
			},
			want: `Unknown static unit in config override, "nottest".`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			provider, _ := providers.GetProvider(&tt.cfg)
			p := Plugin{
				Provider:            provider,
				UserConfigOverrides: tt.cfg,
			}
			result := core.CompilationResult{}
			result.AddAll(tt.result)
			observedZapCore, observedLogs := observer.New(zap.WarnLevel)
			observedLogger := zap.New(observedZapCore)

			p.validateConfigOverrideResourcesExist(&result, observedLogger.Sugar())
			if tt.want != "" {
				assert.Equal(observedLogs.Len(), 1)
				assert.Equal(tt.want, observedLogs.All()[0].Message)
			} else {
				assert.Equal(observedLogs.Len(), 0)
			}
		})
	}
}
