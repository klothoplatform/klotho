package javascript

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/stretchr/testify/assert"
)

func TestRuntimePath(t *testing.T) {
	tests := []struct {
		name        string
		srcPath     string
		runtimePath string
		want        string
		wantErr     bool
	}{
		{
			name:        "root file",
			srcPath:     "index.js",
			runtimePath: "my_module",
			want:        "./klotho_runtime/my_module",
		},
		{
			name:        "subdir file",
			srcPath:     "folder/index.js",
			runtimePath: "my_module",
			want:        "../klotho_runtime/my_module",
		},
		{
			name:        "deep subdir file",
			srcPath:     "a/b/c/d/index.js",
			runtimePath: "my_module",
			want:        "../../../../klotho_runtime/my_module",
		},
		{
			name:    "no absolute",
			srcPath: "/index.js",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			got, err := RuntimePath(tt.srcPath, tt.runtimePath)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.want, got)
		})
	}
}

type NoopRuntime struct{}

func (NoopRuntime) AddKvRuntimeFiles(unit *types.ExecutionUnit) error { return nil }
func (NoopRuntime) AddFsRuntimeFiles(unit *types.ExecutionUnit, envVarName string, id string) error {
	return nil
}
func (NoopRuntime) AddSecretRuntimeFiles(unit *types.ExecutionUnit) error       { return nil }
func (NoopRuntime) AddOrmRuntimeFiles(unit *types.ExecutionUnit) error          { return nil }
func (NoopRuntime) AddRedisNodeRuntimeFiles(unit *types.ExecutionUnit) error    { return nil }
func (NoopRuntime) AddRedisClusterRuntimeFiles(unit *types.ExecutionUnit) error { return nil }
func (NoopRuntime) AddPubsubRuntimeFiles(unit *types.ExecutionUnit) error       { return nil }
func (NoopRuntime) AddProxyRuntimeFiles(unit *types.ExecutionUnit, proxyType string) error {
	return nil
}
func (NoopRuntime) AddExecRuntimeFiles(unit *types.ExecutionUnit, constructGraph *construct.ConstructGraph) error {
	return nil
}
func (NoopRuntime) TransformPersist(file *types.SourceFile, annot *types.Annotation, construct construct.Construct) error {
	return nil
}
