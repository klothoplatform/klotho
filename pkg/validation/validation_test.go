package validation

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func Test_validation_checkAnnotationForResource(t *testing.T) {
	tests := []struct {
		name    string
		annot   core.Annotation
		want    core.Construct
		result  []core.Construct
		wantErr bool
	}{
		{
			name: "exec unit match",
			annot: core.Annotation{
				Capability: &annotation.Capability{Name: annotation.ExecutionUnitCapability, ID: "test"},
			},
			result: []core.Construct{&core.ExecutionUnit{
				Name: "test",
			}},
			want: &core.ExecutionUnit{
				Name: "test",
			},
		},
		{
			name: "pubsub match",
			annot: core.Annotation{
				Capability: &annotation.Capability{Name: annotation.PubSubCapability, ID: "test"},
			},
			result: []core.Construct{&core.PubSub{
				Name: "test",
			}},
			want: &core.PubSub{
				Name: "test",
			},
		},
		{
			name: "gateway match",
			annot: core.Annotation{
				Capability: &annotation.Capability{Name: annotation.ExposeCapability, ID: "test"},
			},
			result: []core.Construct{&core.Gateway{
				Name: "test",
			}},
			want: &core.Gateway{
				Name: "test",
			},
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
				Name: "test",
			}},
			want: &core.Fs{
				Name: "test",
			},
		},
		{
			name: "persist kv match",
			annot: core.Annotation{
				Capability: &annotation.Capability{Name: annotation.PersistCapability, ID: "test"},
			},
			result: []core.Construct{&core.Kv{
				Name: "test",
			}},
			want: &core.Kv{
				Name: "test",
			},
		},
		{
			name: "persist orm match",
			annot: core.Annotation{
				Capability: &annotation.Capability{Name: annotation.PersistCapability, ID: "test"},
			},
			result: []core.Construct{&core.Orm{
				Name: "test",
			}},
			want: &core.Orm{
				Name: "test",
			},
		},
		{
			name: "persist redis cluster match",
			annot: core.Annotation{
				Capability: &annotation.Capability{Name: annotation.PersistCapability, ID: "test"},
			},
			result: []core.Construct{&core.RedisCluster{
				Name: "test",
			}},
			want: &core.RedisCluster{
				Name: "test",
			},
		},
		{
			name: "persist redis node match",
			annot: core.Annotation{
				Capability: &annotation.Capability{Name: annotation.PersistCapability, ID: "test"},
			},
			result: []core.Construct{&core.RedisNode{
				Name: "test",
			}},
			want: &core.RedisNode{
				Name: "test",
			},
		},
		{
			name: "persist secret match",
			annot: core.Annotation{
				Capability: &annotation.Capability{Name: annotation.PersistCapability, ID: "test"},
			},
			result: []core.Construct{&core.Secrets{
				Name: "test",
			}},
			want: &core.Secrets{
				Name: "test",
			},
		},
		{
			name: "no match on capability should return empty resource",
			annot: core.Annotation{
				Capability: &annotation.Capability{Name: annotation.ExecutionUnitCapability, ID: "test"},
			},
			result: []core.Construct{&core.Kv{
				Name: "test",
			}},
		},
		{
			name: "no match on id should return empty resource",
			annot: core.Annotation{
				Capability: &annotation.Capability{Name: annotation.PersistCapability, ID: "test"},
			},
			result: []core.Construct{&core.Kv{
				Name: "nope",
			}},
		},
		{
			name: "only one match should succeed",
			annot: core.Annotation{
				Capability: &annotation.Capability{Name: annotation.PersistCapability, ID: "test"},
			},
			result: []core.Construct{
				&core.Kv{
					Name: "nope",
				},
				&core.Kv{
					Name: "test",
				}},
			want: &core.Kv{
				Name: "test",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			p := ConstructValidation{}
			result := core.NewConstructGraph()
			for _, c := range tt.result {
				result.AddConstruct(c)
			}
			log := zap.L().With().Sugar()

			resource := p.checkAnnotationForResource(&tt.annot, result, log)
			assert.Equal(tt.want, resource)

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
					Name: "test",
				},
				&core.Gateway{
					Name: "test",
				},
			},
			wantErr: false,
		},
		{
			name: "persist different ids",
			result: []core.Construct{
				&core.Kv{
					Name: "test",
				},
				&core.Kv{
					Name: "another",
				},
			},
			wantErr: false,
		},
		{
			name: "persist duplicate ids",
			result: []core.Construct{
				&core.Kv{
					Name: "test",
				},
				&core.Kv{
					Name: "test",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			p := ConstructValidation{}
			result := core.NewConstructGraph()
			for _, c := range tt.result {
				result.AddConstruct(c)
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
			result: []core.Construct{&core.ExecutionUnit{
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
			result: []core.Construct{&core.Kv{
				Name: "test",
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
				Name: "test",
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
			result: []core.Construct{&core.Gateway{
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
			name: "static unit match",
			result: []core.Construct{&core.StaticUnit{
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
			result: []core.Construct{&core.StaticUnit{
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
		{
			name: "config resource match",
			result: []core.Construct{&core.Config{
				Name: "test",
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
				Name: "test",
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

			p := ConstructValidation{
				UserConfigOverrides: tt.cfg,
			}
			result := core.NewConstructGraph()
			for _, c := range tt.result {
				result.AddConstruct(c)
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
