package validation

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func Test_validation_checkAnnotationForResource(t *testing.T) {
	tests := []struct {
		name    string
		annot   types.Annotation
		want    construct.Construct
		result  []construct.Construct
		wantErr bool
	}{
		{
			name: "exec unit match",
			annot: types.Annotation{
				Capability: &annotation.Capability{Name: annotation.ExecutionUnitCapability, ID: "test"},
			},
			result: []construct.Construct{&types.ExecutionUnit{
				Name: "test",
			}},
			want: &types.ExecutionUnit{
				Name: "test",
			},
		},
		{
			name: "pubsub match",
			annot: types.Annotation{
				Capability: &annotation.Capability{Name: annotation.PubSubCapability, ID: "test"},
			},
			result: []construct.Construct{&types.PubSub{
				Name: "test",
			}},
			want: &types.PubSub{
				Name: "test",
			},
		},
		{
			name: "gateway match",
			annot: types.Annotation{
				Capability: &annotation.Capability{Name: annotation.ExposeCapability, ID: "test"},
			},
			result: []construct.Construct{&types.Gateway{
				Name: "test",
			}},
			want: &types.Gateway{
				Name: "test",
			},
		},
		{
			name: "embed assets match",
			annot: types.Annotation{
				Capability: &annotation.Capability{Name: annotation.AssetCapability, ID: "test"},
			},
			result: []construct.Construct{},
		},
		{
			name: "persist fs match",
			annot: types.Annotation{
				Capability: &annotation.Capability{Name: annotation.PersistCapability, ID: "test"},
			},
			result: []construct.Construct{&types.Fs{
				Name: "test",
			}},
			want: &types.Fs{
				Name: "test",
			},
		},
		{
			name: "persist kv match",
			annot: types.Annotation{
				Capability: &annotation.Capability{Name: annotation.PersistCapability, ID: "test"},
			},
			result: []construct.Construct{&types.Kv{
				Name: "test",
			}},
			want: &types.Kv{
				Name: "test",
			},
		},
		{
			name: "persist orm match",
			annot: types.Annotation{
				Capability: &annotation.Capability{Name: annotation.PersistCapability, ID: "test"},
			},
			result: []construct.Construct{&types.Orm{
				Name: "test",
			}},
			want: &types.Orm{
				Name: "test",
			},
		},
		{
			name: "persist redis cluster match",
			annot: types.Annotation{
				Capability: &annotation.Capability{Name: annotation.PersistCapability, ID: "test"},
			},
			result: []construct.Construct{&types.RedisCluster{
				Name: "test",
			}},
			want: &types.RedisCluster{
				Name: "test",
			},
		},
		{
			name: "persist redis node match",
			annot: types.Annotation{
				Capability: &annotation.Capability{Name: annotation.PersistCapability, ID: "test"},
			},
			result: []construct.Construct{&types.RedisNode{
				Name: "test",
			}},
			want: &types.RedisNode{
				Name: "test",
			},
		},
		{
			name: "persist secret match",
			annot: types.Annotation{
				Capability: &annotation.Capability{Name: annotation.PersistCapability, ID: "test"},
			},
			result: []construct.Construct{&types.Secrets{
				Name: "test",
			}},
			want: &types.Secrets{
				Name: "test",
			},
		},
		{
			name: "no match on capability should return empty resource",
			annot: types.Annotation{
				Capability: &annotation.Capability{Name: annotation.ExecutionUnitCapability, ID: "test"},
			},
			result: []construct.Construct{&types.Kv{
				Name: "test",
			}},
		},
		{
			name: "no match on id should return empty resource",
			annot: types.Annotation{
				Capability: &annotation.Capability{Name: annotation.PersistCapability, ID: "test"},
			},
			result: []construct.Construct{&types.Kv{
				Name: "nope",
			}},
		},
		{
			name: "only one match should succeed",
			annot: types.Annotation{
				Capability: &annotation.Capability{Name: annotation.PersistCapability, ID: "test"},
			},
			result: []construct.Construct{
				&types.Kv{
					Name: "nope",
				},
				&types.Kv{
					Name: "test",
				}},
			want: &types.Kv{
				Name: "test",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			p := ConstructValidation{}
			result := construct.NewConstructGraph()
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
		result  []construct.Construct
		wantErr bool
	}{
		{
			name: "diff resources duplicate ids",
			result: []construct.Construct{
				&types.ExecutionUnit{
					Name: "test",
				},
				&types.Gateway{
					Name: "test",
				},
			},
			wantErr: false,
		},
		{
			name: "persist different ids",
			result: []construct.Construct{
				&types.Kv{
					Name: "test",
				},
				&types.Kv{
					Name: "another",
				},
			},
			wantErr: false,
		},
		{
			name: "persist duplicate ids",
			result: []construct.Construct{
				&types.Kv{
					Name: "test",
				},
				&types.Kv{
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
			result := construct.NewConstructGraph()
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
		result []construct.Construct
		cfg    config.Application
		want   string
	}{
		{
			name: "exec unit match",
			result: []construct.Construct{&types.ExecutionUnit{
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
			result: []construct.Construct{&types.ExecutionUnit{
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
			result: []construct.Construct{&types.Kv{
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
			result: []construct.Construct{&types.Kv{
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
			result: []construct.Construct{&types.Gateway{
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
			result: []construct.Construct{&types.Gateway{
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
			result: []construct.Construct{&types.StaticUnit{
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
			result: []construct.Construct{&types.StaticUnit{
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
			result: []construct.Construct{&types.Config{
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
			result: []construct.Construct{&types.Config{
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
			result := construct.NewConstructGraph()
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
