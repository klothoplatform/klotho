package golang

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/graph"
)

type NoopRuntime struct{}

func (n NoopRuntime) AddExecRuntimeFiles(unit *core.ExecutionUnit, constructGraph *graph.Directed[core.Construct]) error {
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

func (n NoopRuntime) ActOnExposeListener(unit *core.ExecutionUnit, f *core.SourceFile, listener *HttpListener, routerName string) error {
	return nil
}
