package validation

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/graph"
	"github.com/klothoplatform/klotho/pkg/provider/providers"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func Test_validation_checkAnnotationForResource(t *testing.T) {
	tests := []struct {
		name    string
		annot   core.Annotation
		want    core.AnnotationKey
		result  []core.Construct
		wantErr bool
	}{
		{
			name: "exec unit match",
			annot: core.Annotation{
				Capability: &annotation.Capability{Name: annotation.ExecutionUnitCapability, ID: "test"},
			},
			result: []core.Construct{&core.ExecutionUnit{
				AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability},
			}},
			want: core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability},
		},
		{
			name: "pubsub match",
			annot: core.Annotation{
				Capability: &annotation.Capability{Name: annotation.PubSubCapability, ID: "test"},
			},
			result: []core.Construct{&core.PubSub{
				AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.PubSubCapability},
			}},
			want: core.AnnotationKey{ID: "test", Capability: annotation.PubSubCapability},
		},
		{
			name: "gateway match",
			annot: core.Annotation{
				Capability: &annotation.Capability{Name: annotation.ExposeCapability, ID: "test"},
			},
			result: []core.Construct{&core.Gateway{
				AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability},
			}},
			want: core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability},
		},
		{
			name: "embed assets match",
			annot: core.Annotation{
				Capability: &annotation.Capability{Name: annotation.AssetCapability, ID: "test"},
			},
			result: []core.Construct{},
		},
		{
			name: "persist fs match",
			annot: core.Annotation{
				Capability: &annotation.Capability{Name: annotation.PersistCapability, ID: "test"},
			},
			result: []core.Construct{&core.Fs{
				AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.PersistCapability},
			}},
			want: core.AnnotationKey{ID: "test", Capability: annotation.PersistCapability},
		},
		{
			name: "persist kv match",
			annot: core.Annotation{
				Capability: &annotation.Capability{Name: annotation.PersistCapability, ID: "test"},
			},
			result: []core.Construct{&core.Kv{
				AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.PersistCapability},
			}},
			want: core.AnnotationKey{ID: "test", Capability: annotation.PersistCapability},
		},
		{
			name: "persist orm match",
			annot: core.Annotation{
				Capability: &annotation.Capability{Name: annotation.PersistCapability, ID: "test"},
			},
			result: []core.Construct{&core.Orm{
				AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.PersistCapability},
			}},
			want: core.AnnotationKey{ID: "test", Capability: annotation.PersistCapability},
		},
		{
			name: "persist redis cluster match",
			annot: core.Annotation{
				Capability: &annotation.Capability{Name: annotation.PersistCapability, ID: "test"},
			},
			result: []core.Construct{&core.RedisCluster{
				AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.PersistCapability},
			}},
			want: core.AnnotationKey{ID: "test", Capability: annotation.PersistCapability},
		},
		{
			name: "persist redis node match",
			annot: core.Annotation{
				Capability: &annotation.Capability{Name: annotation.PersistCapability, ID: "test"},
			},
			result: []core.Construct{&core.RedisNode{
				AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.PersistCapability},
			}},
			want: core.AnnotationKey{ID: "test", Capability: annotation.PersistCapability},
		},
		{
			name: "persist secret match",
			annot: core.Annotation{
				Capability: &annotation.Capability{Name: annotation.PersistCapability, ID: "test"},
			},
			result: []core.Construct{&core.Secrets{
				AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.PersistCapability},
			}},
			want: core.AnnotationKey{ID: "test", Capability: annotation.PersistCapability},
		},
		{
			name: "no match on capability should return empty resource",
			annot: core.Annotation{
				Capability: &annotation.Capability{Name: annotation.ExecutionUnitCapability, ID: "test"},
			},
			result: []core.Construct{&core.Kv{
				AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.PersistCapability},
			}},
			want: core.AnnotationKey{},
		},
		{
			name: "no match on id should return empty resource",
			annot: core.Annotation{
				Capability: &annotation.Capability{Name: annotation.PersistCapability, ID: "test"},
			},
			result: []core.Construct{&core.Kv{
				AnnotationKey: core.AnnotationKey{ID: "nope", Capability: annotation.PersistCapability},
			}},
			want: core.AnnotationKey{},
		},
		{
			name: "only one match should succeed",
			annot: core.Annotation{
				Capability: &annotation.Capability{Name: annotation.PersistCapability, ID: "test"},
			},
			result: []core.Construct{
				&core.Kv{
					AnnotationKey: core.AnnotationKey{ID: "nope", Capability: annotation.PersistCapability},
				},
				&core.Kv{
					AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.PersistCapability},
				}},
			want: core.AnnotationKey{ID: "test", Capability: annotation.PersistCapability},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			p := Plugin{}
			result := graph.NewDirected[core.Construct]()
			for _, c := range tt.result {
				result.AddVertex(c)
			}
			log := zap.L().With().Sugar()

			resource := p.checkAnnotationForResource(&tt.annot, result, log)
			assert.Equal(tt.want, resource)

		})
	}
}

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
				AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability},
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
				AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability},
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

			provider, _ := providers.GetProvider(&tt.cfg)
			p := Plugin{
				Provider: provider,
				Config:   &tt.cfg,
			}
			result := graph.NewDirected[core.Construct]()
			for _, c := range tt.result {
				result.AddVertex(c)
			}

			err := p.handleProviderValidation(result)
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

func Test_validation_handleResources(t *testing.T) {
	tests := []struct {
		name    string
		result  []core.Construct
		wantErr bool
	}{
		{
			name: "diff resources duplicate ids",
			result: []core.Construct{
				&core.ExecutionUnit{
					AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability},
				},
				&core.Gateway{
					AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExposeCapability},
				},
			},
			wantErr: false,
		},
		{
			name: "persist different ids",
			result: []core.Construct{
				&core.Kv{
					AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.PersistCapability},
				},
				&core.Kv{
					AnnotationKey: core.AnnotationKey{ID: "another", Capability: annotation.PersistCapability},
				},
			},
			wantErr: false,
		},
		{
			name: "persist duplicate ids",
			result: []core.Construct{
				&core.Kv{
					AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.PersistCapability},
				},
				&core.Kv{
					AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.PersistCapability},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			p := Plugin{}
			result := graph.NewDirected[core.Construct]()
			for _, c := range tt.result {
				result.AddVertex(c)
			}

			err := p.handleResources(result)
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
		result []core.Construct
		cfg    config.Application
		want   string
	}{
		{
			name: "exec unit match",
			result: []core.Construct{&core.ExecutionUnit{
				AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability},
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
			result: []core.Construct{&core.ExecutionUnit{
				AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability},
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
			result: []core.Construct{&core.Orm{
				AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.PersistCapability},
			}},
			cfg: config.Application{
				Provider: "aws",
				PersistKv: map[string]*config.Persist{
					"test": {},
				},
			},
		},
		{
			name: "persist mismatch",
			result: []core.Construct{&core.Kv{
				AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.PersistCapability},
			}},
			cfg: config.Application{
				Provider: "aws",
				PersistKv: map[string]*config.Persist{
					"nottest": {},
				},
			},
			want: `Unknown persist_kv in config override, "nottest".`,
		},
		{
			name: "expose match",
			result: []core.Construct{&core.Gateway{
				AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExposeCapability},
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
			result: []core.Construct{&core.Gateway{
				AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExposeCapability},
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
			result: []core.Construct{&core.PubSub{
				AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.PubSubCapability},
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
			result: []core.Construct{&core.PubSub{
				AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.PubSubCapability},
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
			result: []core.Construct{&core.StaticUnit{
				AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.StaticUnitCapability},
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
			result: []core.Construct{&core.StaticUnit{
				AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.StaticUnitCapability},
			}},
			cfg: config.Application{
				Provider: "aws",
				StaticUnit: map[string]*config.StaticUnit{
					"nottest": {},
				},
			},
			want: `Unknown static unit in config override, "nottest".`,
		},
		{
			name: "config resource match",
			result: []core.Construct{&core.Config{
				AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ConfigCapability},
			}},
			cfg: config.Application{
				Provider: "aws",
				Config: map[string]*config.Config{
					"test": {},
				},
			},
		},
		{
			name: "config resource mismatch",
			result: []core.Construct{&core.Config{
				AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ConfigCapability},
			}},
			cfg: config.Application{
				Provider: "aws",
				Config: map[string]*config.Config{
					"nottest": {},
				},
			},
			want: `Unknown config resource in config override, "nottest".`,
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
			result := graph.NewDirected[core.Construct]()
			for _, c := range tt.result {
				result.AddVertex(c)
			}

			observedZapCore, observedLogs := observer.New(zap.WarnLevel)
			observedLogger := zap.New(observedZapCore)

			p.validateConfigOverrideResourcesExist(result, observedLogger.Sugar())
			if tt.want != "" {
				assert.Equal(observedLogs.Len(), 1)
				assert.Equal(tt.want, observedLogs.All()[0].Message)
			} else {
				assert.Equal(observedLogs.Len(), 0)
			}
		})
	}
}
