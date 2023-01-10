package javascript

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
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

func (NoopRuntime) AddKvRuntimeFiles(unit *core.ExecutionUnit) error                      { return nil }
func (NoopRuntime) AddFsRuntimeFiles(unit *core.ExecutionUnit) error                      { return nil }
func (NoopRuntime) AddSecretRuntimeFiles(unit *core.ExecutionUnit) error                  { return nil }
func (NoopRuntime) AddOrmRuntimeFiles(unit *core.ExecutionUnit) error                     { return nil }
func (NoopRuntime) AddRedisNodeRuntimeFiles(unit *core.ExecutionUnit) error               { return nil }
func (NoopRuntime) AddRedisClusterRuntimeFiles(unit *core.ExecutionUnit) error            { return nil }
func (NoopRuntime) AddPubsubRuntimeFiles(unit *core.ExecutionUnit) error                  { return nil }
func (NoopRuntime) AddProxyRuntimeFiles(unit *core.ExecutionUnit, proxyType string) error { return nil }
func (NoopRuntime) AddExecRuntimeFiles(unit *core.ExecutionUnit, result *core.CompilationResult, deps *core.Dependencies) error {
	return nil
}
func (NoopRuntime) TransformPersist(file *core.SourceFile, annot *core.Annotation, kind core.PersistKind, content string) (TransformResult, error) {
	return TransformResult{NewFileContent: content, NewAnnotationContent: annot.Node.Content(file.Program())}, nil
}
