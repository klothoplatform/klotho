package python

import (
	"fmt"
	"testing"

	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/stretchr/testify/assert"
)

func TestAddRequirements(t *testing.T) {
	t.Run("requirements.txt provided", func(t *testing.T) {
		// If we have multiple requirements.txt files (for whatever reason), we'll add the requirements to each.
		// We can revisit that later if needed; for now, it's an easy implementation.
		// Duplicate requirements are fine, as long as they don't contradict.
		assert := assert.New(t)

		unit := &types.ExecutionUnit{}
		pip1, pip2 := &RequirementsTxt{path: "requirements.txt"}, &RequirementsTxt{path: "extra-requirements.txt"}
		unit.Add(pip1)
		unit.Add(pip2)

		AddRequirements(unit, "my reqs")
		assert.Len(unit.Files(), 2)
		assert.Equal([]string{"my reqs"}, pip1.extras)
		assert.Equal([]string{"my reqs"}, pip2.extras)
	})
	t.Run("requirements.txt missing", func(t *testing.T) {
		assert := assert.New(t)

		unit := &types.ExecutionUnit{}

		assert.Len(unit.Files(), 0) // to compare with the check in two lines
		AddRequirements(unit, "my reqs")
		if !assert.Len(unit.Files(), 1) { // one got generated
			return
		}

		pip := unit.Files()["requirements.txt"].(*RequirementsTxt)
		assert.Equal([]string{"my reqs"}, pip.extras)
	})

}

type NoopRuntime struct{}

func (n NoopRuntime) AddExecRuntimeFiles(unit *types.ExecutionUnit, constructGraph *construct.ConstructGraph) error {
	return nil
}
func (n NoopRuntime) AddExposeRuntimeFiles(unit *types.ExecutionUnit) error { return nil }

func (n NoopRuntime) AddKvRuntimeFiles(unit *types.ExecutionUnit) error { return nil }

func (n NoopRuntime) AddFsRuntimeFiles(unit *types.ExecutionUnit, envVarName string, id string) error {
	return nil
}

func (n NoopRuntime) AddSecretRuntimeFiles(unit *types.ExecutionUnit) error { return nil }

func (n NoopRuntime) AddOrmRuntimeFiles(unit *types.ExecutionUnit) error { return nil }

func (n NoopRuntime) GetFsRuntimeImportClass(id string, varName string) string {
	return fmt.Sprintf("import klotho_runtime.fs_%s as %s", id, varName)
}

func (n NoopRuntime) AddProxyRuntimeFiles(unit *types.ExecutionUnit, proxyType string) error {
	return nil
}

func (n NoopRuntime) GetSecretRuntimeImportClass(varName string) string {
	return fmt.Sprintf("import klotho_runtime.secret as %s", varName)
}

func (n NoopRuntime) GetKvRuntimeConfig() KVConfig {
	return KVConfig{
		Imports: "import keyvalue",
		CacheClassArg: FunctionArg{
			Name:  "cache_class",
			Value: "keyvalue.KVStore",
		},
		AdditionalCacheConstructorArgs: []FunctionArg{{
			Name:  "serializer",
			Value: "keyvalue.NoOpSerializer()",
		}},
	}
}

func (n NoopRuntime) GetAppName() string { return "app" }

func (r NoopRuntime) ValidateRedisClient(id string, clientType string) string { return "" }
