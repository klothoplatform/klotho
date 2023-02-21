package golang

import (
	"github.com/klothoplatform/klotho/pkg/core"
)

type NoopRuntime struct{}

func (n NoopRuntime) AddExecRuntimeFiles(unit *core.ExecutionUnit, result *core.CompilationResult, dependencies *core.Dependencies) error {
	return nil
}
func (n NoopRuntime) GetFsImports() []Import {
	return []Import{
		{Package: "gocloud.dev/blob"},
		{Alias: "_", Package: "gocloud.dev/blob/s3blob"},
	}
}
func (n NoopRuntime) GetSecretsImports() []Import {
	return []Import{
		{Package: "gocloud.dev/runtimevar"},
		{Alias: "_", Package: "gocloud.dev/runtimevar/awssecretsmanager"},
	}
}

func (n NoopRuntime) SetConfigType(id string, isSecret bool) {
}
